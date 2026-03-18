package cli

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"time"

	"github.com/corvade/corvade/internal/capture"
)

//go:embed testdata/*.json
var demoFS embed.FS

// fixtureTrace mirrors the JSON shape used in testdata fixtures.
type fixtureTrace struct {
	ID         string  `json:"id"`
	Agent      *string `json:"agent"`
	Provider   string  `json:"provider"`
	Model      string  `json:"model"`
	Request    string  `json:"request"`
	Response   *string `json:"response"`
	StatusCode int     `json:"status_code"`
	CreatedAt  string  `json:"created_at"`
}

// loadDemoData reads all embedded fixture JSON files and inserts each trace
// into the store. Returns the total number of traces inserted.
func loadDemoData(store *capture.Store) (int, error) {
	entries, err := fs.ReadDir(demoFS, "testdata")
	if err != nil {
		return 0, fmt.Errorf("reading embedded testdata: %w", err)
	}

	total := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := demoFS.ReadFile("testdata/" + entry.Name())
		if err != nil {
			return total, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		var fixtures []fixtureTrace
		if err := json.Unmarshal(data, &fixtures); err != nil {
			return total, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		for _, f := range fixtures {
			createdAt, err := time.Parse(time.RFC3339, f.CreatedAt)
			if err != nil {
				createdAt = time.Now().UTC()
			}

			tr := capture.Trace{
				Agent:      f.Agent,
				Provider:   f.Provider,
				Model:      f.Model,
				Request:    f.Request,
				Response:   f.Response,
				StatusCode: f.StatusCode,
				CreatedAt:  createdAt,
			}

			if _, err := store.InsertTrace(tr); err != nil {
				return total, fmt.Errorf("inserting trace from %s: %w", entry.Name(), err)
			}
			total++
		}
	}

	return total, nil
}
