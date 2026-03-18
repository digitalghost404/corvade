package handlers

import (
	"net/http"
	"strings"

	"github.com/corvade/corvade/internal/capture"
)

// DiffHandler handles the session diff endpoint.
type DiffHandler struct {
	store *capture.Store
}

// NewDiffHandler creates a new DiffHandler.
func NewDiffHandler(store *capture.Store) *DiffHandler {
	return &DiffHandler{store: store}
}

// AlignedPair holds two matched nodes and their similarity score.
type AlignedPair struct {
	Left       *capture.GraphNode `json:"left"`
	Right      *capture.GraphNode `json:"right"`
	MatchScore float64            `json:"match_score"`
}

// DiffResponse is the JSON response for the session diff endpoint.
type DiffResponse struct {
	AlignedNodes     []AlignedPair       `json:"aligned_nodes"`
	LeftOnly         []capture.GraphNode `json:"left_only"`
	RightOnly        []capture.GraphNode `json:"right_only"`
	DivergencePoints []capture.GraphNode `json:"divergence_points"`
}

const divergenceThreshold = 0.5

// Diff handles GET /api/sessions/:id1/diff/:id2.
func (h *DiffHandler) Diff(w http.ResponseWriter, r *http.Request, id1, id2 string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	leftNodes, _, err := h.store.GetSessionGraph(id1)
	if err != nil {
		http.Error(w, "failed to load left session graph: "+err.Error(), http.StatusInternalServerError)
		return
	}
	rightNodes, _, err := h.store.GetSessionGraph(id2)
	if err != nil {
		http.Error(w, "failed to load right session graph: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if leftNodes == nil {
		leftNodes = []capture.GraphNode{}
	}
	if rightNodes == nil {
		rightNodes = []capture.GraphNode{}
	}

	resp := computeDiff(leftNodes, rightNodes)
	writeJSON(w, http.StatusOK, resp)
}

// computeDiff performs greedy structural matching between two node slices and
// returns the categorised diff result.
func computeDiff(left, right []capture.GraphNode) DiffResponse {
	usedRight := make([]bool, len(right))

	var aligned []AlignedPair
	var leftOnly []capture.GraphNode
	var divergencePoints []capture.GraphNode

	for i := range left {
		bestScore := -1.0
		bestJ := -1

		for j := range right {
			if usedRight[j] {
				continue
			}
			score := nodeMatchScore(&left[i], &right[j])
			if score > bestScore {
				bestScore = score
				bestJ = j
			}
		}

		// Only form a pair if the best score is meaningfully positive.
		if bestJ >= 0 && bestScore > 0 {
			usedRight[bestJ] = true
			pair := AlignedPair{
				Left:       &left[i],
				Right:      &right[bestJ],
				MatchScore: bestScore,
			}
			aligned = append(aligned, pair)
			if bestScore < divergenceThreshold {
				divergencePoints = append(divergencePoints, left[i])
			}
		} else {
			leftOnly = append(leftOnly, left[i])
		}
	}

	var rightOnly []capture.GraphNode
	for j := range right {
		if !usedRight[j] {
			rightOnly = append(rightOnly, right[j])
		}
	}

	// Normalise nil slices to empty arrays for JSON serialisation.
	if aligned == nil {
		aligned = []AlignedPair{}
	}
	if leftOnly == nil {
		leftOnly = []capture.GraphNode{}
	}
	if rightOnly == nil {
		rightOnly = []capture.GraphNode{}
	}
	if divergencePoints == nil {
		divergencePoints = []capture.GraphNode{}
	}

	return DiffResponse{
		AlignedNodes:     aligned,
		LeftOnly:         leftOnly,
		RightOnly:        rightOnly,
		DivergencePoints: divergencePoints,
	}
}

// nodeMatchScore computes a [0,1] similarity score between two graph nodes.
// It combines exact label matches (agent, step, type, model) with Jaccard
// similarity on context snapshot tokens.
func nodeMatchScore(a, b *capture.GraphNode) float64 {
	score := 0.0
	weight := 0.0

	// Type match (mandatory signal — low weight but required for sensible alignment)
	weight += 1.0
	if a.Type == b.Type {
		score += 1.0
	}

	// Agent match
	weight += 1.5
	if strPtrEqual(a.Agent, b.Agent) {
		score += 1.5
	}

	// Step label match
	weight += 2.0
	if strPtrEqual(a.Step, b.Step) {
		score += 2.0
	}

	// Model match
	weight += 1.0
	if strPtrEqual(a.Model, b.Model) {
		score += 1.0
	}

	// Jaccard similarity on context snapshot tokens
	weight += 2.0
	jac := jaccardSimilarity(a.ContextSnapshot, b.ContextSnapshot)
	score += jac * 2.0

	return score / weight
}

// strPtrEqual returns true when both pointers are nil or point to equal strings.
func strPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// jaccardSimilarity computes the Jaccard index over the word-token sets of
// two optional strings.  Returns 1.0 when both are nil (identical absence).
func jaccardSimilarity(a, b *string) float64 {
	if a == nil && b == nil {
		return 1.0
	}
	if a == nil || b == nil {
		return 0.0
	}

	setA := tokenSet(*a)
	setB := tokenSet(*b)

	intersection := 0
	for tok := range setA {
		if setB[tok] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 1.0
	}
	return float64(intersection) / float64(union)
}

// tokenSet splits s into whitespace-delimited tokens and returns them as a set.
func tokenSet(s string) map[string]bool {
	tokens := strings.Fields(s)
	set := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		set[t] = true
	}
	return set
}
