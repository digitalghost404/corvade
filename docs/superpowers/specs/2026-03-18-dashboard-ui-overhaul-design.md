# Corvade Dashboard UI Overhaul

**Date:** 2026-03-18
**Status:** Design approved, pending implementation plan

## Overview

Production polish pass on the Corvade dashboard — branding, layout restructure, component redesign, animations, and keyboard navigation. Takes the dashboard from "working prototype" to "real product."

**Visual direction:** Dark, minimal dev-tool aesthetic (Linear meets Grafana) with a violet corvid edge.

**Tagline:** "See what your agents see."

## Section 1: Branding & Assets

### Logo — Wired Eye (D3)
An eye shape with topology circuit paths radiating from it — the eye IS a graph node that everything flows through. Merges both brand concepts (observation + topology) into one mark.

- `dashboard/public/logo.svg` — full wired eye mark in violet (#8b5cf6)
- `dashboard/public/favicon.svg` — simplified eye (no circuit traces, just eye + diamond pupil) for browser tab
- `dashboard/public/favicon.ico` — replaces the default Next.js favicon with Corvade's simplified eye, generated from favicon.svg

### Color Tokens
| Token | Value | Usage |
|-------|-------|-------|
| Primary | #8b5cf6 (violet-500) | Logo, active states, accents, links |
| Primary hover | #7c3aed (violet-600) | Hover states on primary elements |
| Primary muted | #8b5cf6 at 10% opacity | Background highlights, new-trace flash |
| Status green | #22c55e | Success / 200 status codes |
| Status yellow | #eab308 | Warning / 429 rate limits |
| Status red | #ef4444 | Error / 4xx+ status codes |
| Surface 1 | #09090b (zinc-950) | Page background |
| Surface 2 | #18181b (zinc-900) | Cards, header, elevated surfaces |
| Surface 3 | #27272a (zinc-800) | Borders, dividers, input backgrounds |
| Text primary | #fafafa (zinc-50) | Headings, important values |
| Text secondary | #a1a1aa (zinc-400) | Labels, secondary info |
| Text muted | #71717a (zinc-500) | Placeholders, disabled states |

### Metadata
- Title: "Corvade"
- Description: "See what your agents see."
- Replace default Next.js favicon with Corvade wired eye

## Section 2: Layout Shell & Navigation

### Header (fixed top bar)
- **Left:** Wired eye logo SVG (24px) + "Corvade" wordmark in zinc-50, font-semibold
- **Right:** Live stats from /api/stats — "{n} traces · ${cost} · {time since last trace}"
- Background: zinc-900, bottom border zinc-800
- Height: 48px

### Tab Navigation
Directly below header, full width. Tabs are proper links with URLs:

| Tab | URL | Content |
|-----|-----|---------|
| Traces | `/` | Timeline view (default) |
| Sessions | `/sessions` | Session card grid |
| Diff | `/diff` | Session comparison |
| Stats | `/stats` | Cost and usage analytics |

- Active tab: violet-500 bottom border (2px), zinc-50 text
- Inactive: no border, zinc-500 text, hover → zinc-300
- Tab switches are instant (no fade/transition)
- Background: zinc-950, bottom border zinc-800

### Global Keyboard Shortcuts
| Key | Action |
|-----|--------|
| `1` / `2` / `3` / `4` | Switch to tab 1-4 |
| `?` | Show keyboard shortcut help overlay |

**Focus scoping:** All keyboard shortcuts are no-ops when `document.activeElement` is an `input`, `textarea`, or `select` element. This prevents `?`, `/`, `1`-`4`, `j`/`k` from firing while the user is typing in search or filter inputs.

### Keyboard Help Overlay
- Triggered by `?` key
- Centered modal, zinc-950/80 backdrop, 100ms fade-in
- Lists shortcuts grouped: Global (tab switching, help), Traces (j/k navigation, inspector), Inspector (tabs, close)
- Sessions, Diff, and Stats tabs have no tab-specific shortcuts in v1
- Dismiss with `Escape` or `?` again

## Section 3: Traces Tab

### Stats Summary Bar
Row of metric cards at top of traces tab:
```
┌──────────┬──────────┬──────────┬──────────┬──────────┐
│ 7 traces │ 365 tok  │  $0.003  │ 2 agents │ 2 models │
└──────────┴──────────┴──────────┴──────────┴──────────┘
```
- zinc-900 background cards with zinc-800 border
- Violet accent on the numbers (`tabular-nums` font-variant for stable width), zinc-400 labels
- Updates live via WebSocket `trace:new` events (increment counters on each event)
- On WebSocket disconnect: values freeze at last known state (no error indicator — non-goal). Reconnection happens automatically via the existing CorvadeWS auto-reconnect.
- Counter updates: fade the element opacity to 0.7 then back to 1.0 over 200ms on value change (CSS transition on opacity, triggered by a brief class toggle in JS). Not a number-counting animation.

### Filter Bar
- Violet focus ring on inputs (ring-violet-500/50)
- "Clear filters" button appears when any filter is active (zinc-500 text, hover zinc-300)
- Debounced search input (300ms delay)

### Trace Table
- **Row density:** py-2 padding (tighter than current py-3)
- **Hover:** zinc-800/50 background + 2px violet left border (100ms transition)
- **Selected:** Violet left border, zinc-800/70 background
- **Alternating rows:** Very subtle zinc-950 / zinc-900/30 alternation
- **Timestamps:** Relative ("2s ago", "5m ago") with full ISO timestamp in hover tooltip
- **Cost:** Plain dollar format ($0.000425)

### Detail Inspector
- Expands below selected row with max-height + opacity animation (150ms ease). Use a generous `max-height: 600px` for the transition target — the timing mismatch is acceptable at 150ms. If content exceeds 600px, it simply appears at full height after the transition completes.
- Full-width, part of the table flow (not a floating card)
- Request/Response tabs show token count badge: "Request (25 tok)" / "Response (15 tok)"
- JSON viewer has line numbers in left gutter (zinc-600, monospace)
- Collapses with same animation on deselect

### Loading State
- 5 skeleton rows matching table column widths
- Pulse animation: zinc-800 → zinc-700 → zinc-800, 1.5s loop
- Pure CSS `@keyframes`, no JS

### Empty State
- Centered layout
- Simplified wired eye SVG illustration (dashed connection lines)
- "No traces yet" heading (zinc-200)
- "Point your agents at the proxy to get started" subtext (zinc-500)
- Code snippet in monospace: `OPENAI_BASE_URL=http://localhost:4400/v1`
- Copy button on the code snippet

### Keyboard Navigation (Traces)
| Key | Action |
|-----|--------|
| `j` / `↓` | Move selection down |
| `k` / `↑` | Move selection up |
| `Enter` / `Space` | Toggle detail inspector |
| `Escape` | Close detail inspector |
| `/` | Focus search input |

## Section 4: Sessions Tab

### Session List
Card grid layout (responsive: 1 col on small, 2 on medium, 3 on large).

Each card shows:
- Agent name (large text) or "Unknown Agent" if null
- Trace count, total tokens, total cost
- Status badge: active (violet), completed (green), error (red)
- Time range: "10:01 → 10:05" or "10:01 → ongoing"
- Model chips: small pills for each model used (zinc-800 bg, zinc-300 text)

Cards link to session detail page at `/session?id=<sessionId>` (existing page, uses query param due to static export). Hover: border shifts to violet (100ms).

### Session Detail Page (`dashboard/src/app/session/page.tsx` — modified)
- Header: session ID (truncated), agent name, status badge, time range, total cost
- Two-panel layout using flex:
  - **Left (60%, `flex: 3`):** Topology graph (existing React Flow component) — container must have explicit height (`calc(100vh - 200px)`) for React Flow to render
  - **Right (40%, `flex: 2`):** Segmented control toggling between Narrative View and Trace List
- Segmented control: two buttons with zinc-800 bg, active gets violet bg

### Empty State
"No sessions yet. Sessions are created when agents tag requests with X-Corvade-Session headers."

## Section 5: Diff Tab

### Session Picker
Two combobox inputs side by side with "vs" label between them and "Compare" button.

Each combobox is a text input with a dropdown:
- Typing filters the session list by agent name or ID prefix
- Dropdown shows recent sessions: agent name + trace count + relative time
- User can also paste a full session ID directly into the input (no dropdown interaction needed)
- Selected session shows as: agent name + truncated ID

### Diff Results
- **Summary bar:** "12 matched · 3 diverged · 2 left-only · 1 right-only"
- **Matched pairs:** Two-column layout, match % shown as colored bar (green >80%, yellow 50-80%)
- **Divergence points:** Highlighted with pulsing violet dot
- **Left-only / Right-only:** Red/green tinted sections with clear labels

### Empty State
"Compare two agent sessions to see where they diverged. Run the same agent twice with different models or prompts."

## Section 6: Stats Tab

### Time Range Selector
Horizontal button group: Last hour | Last 24h | Last 7d | Last 30d | All time. Active button: violet bg. Others: zinc-800 bg.

Each button computes `from` as an ISO 8601 timestamp relative to now and calls `GET /api/stats?from=<iso8601>`. "All time" omits the `from` parameter. The `to` parameter is always omitted (defaults to now).

### Metric Cards
```
┌──────────────┬──────────────┬──────────────┬──────────────┐
│ Total Traces │ Total Cost   │ Total Tokens │   Avg Cost   │
│     847      │   $12.43     │   245,891    │  $0.015/req  │
└──────────────┴──────────────┴──────────────┴──────────────┘
```

### Breakdowns
- **By Model:** Horizontal CSS bar chart — model name, bar proportional to count, cost on right
- **By Agent:** Same layout, grouped by agent name
- **Cost Over Time:** Hand-rolled SVG sparkline showing daily/hourly spend

**No charting library.** CSS bars for breakdowns, SVG path for the sparkline. Keeps bundle size small.

The sparkline is purely decorative — no axes, no hover tooltips, no data point indicators. A single SVG `<path>` drawn from aggregated cost data points. Renders at ~200px wide, 40px tall. If there's only 1 data point, show a flat line.

### Empty State
"Start capturing traces to see usage analytics."

## Section 7: Animations & Transitions

All CSS-only, no animation libraries.

| Element | Animation | Duration |
|---------|-----------|----------|
| Tab switch | Instant, no transition | 0ms |
| Detail inspector expand/collapse | max-height + opacity | 150ms ease |
| Table row hover | Left border + background | 100ms |
| Button hover | Background opacity shift | 100ms |
| Card hover | Border color → violet | 100ms |
| Loading skeletons | Pulse zinc-800 ↔ zinc-700 | 1.5s loop |
| New trace arrival | Flash violet-500/10 bg, fade out | 1s |
| Stats counter update | Opacity flash (1.0 → 0.7 → 1.0) on value change via class toggle | 200ms |
| Keyboard help overlay | Opacity fade in | 100ms |

## Section 8: Responsive Design

| Breakpoint | Behavior |
|-----------|----------|
| ≥1280px (xl) | Full layout, 3-col session cards, side-by-side diff |
| 768-1279px (md) | 2-col session cards, stacked diff panels |
| <768px (sm) | 1-col everything, tabs become horizontally scrollable, table columns hide Latency and Tokens |

No mobile-first — this is a dev tool primarily used on desktop. Responsive is a graceful degradation, not a primary target.

## Section 9: Files Changed

### New Files
- `dashboard/public/logo.svg` — Wired eye logo
- `dashboard/public/favicon.svg` — Simplified eye favicon
- `dashboard/src/components/Header.tsx` — Fixed header with logo + live stats
- `dashboard/src/components/TabNav.tsx` — Tab navigation component
- `dashboard/src/components/StatsBar.tsx` — Metric cards row for traces tab
- `dashboard/src/components/SkeletonRows.tsx` — Loading skeleton for trace table
- `dashboard/src/components/EmptyState.tsx` — Reusable empty state with illustration
- `dashboard/src/components/KeyboardHelp.tsx` — Shortcut overlay modal
- `dashboard/src/components/SessionCard.tsx` — Session card for grid layout
- `dashboard/src/app/sessions/page.tsx` — Sessions tab page
- `dashboard/src/app/stats/page.tsx` — Stats tab page
- `dashboard/src/hooks/useKeyboard.ts` — Keyboard shortcut hook

### Modified Files
- `dashboard/src/app/layout.tsx` — Add Header + TabNav, update metadata
- `dashboard/src/app/page.tsx` — Add StatsBar, remove inline header
- `dashboard/src/app/globals.css` — Add skeleton keyframes, custom scrollbar, focus ring utilities
- `dashboard/src/components/Timeline.tsx` — Row density, hover states, relative timestamps, keyboard nav, alternating rows
- `dashboard/src/components/DetailInspector.tsx` — Slide animation, token badges on tabs, line numbers in JSON viewer
- `dashboard/src/components/SessionDiff.tsx` — Summary bar, improved layout
- `dashboard/src/app/diff/page.tsx` — Session picker dropdowns
- `dashboard/src/app/session/page.tsx` — Two-panel layout (graph + narrative), session header

### Deleted Files
- `dashboard/public/file.svg` — Default Next.js assets
- `dashboard/public/globe.svg`
- `dashboard/public/next.svg`
- `dashboard/public/vercel.svg`
- `dashboard/public/window.svg`

Note: `dashboard/public/favicon.ico` is replaced (not just deleted) — the new Corvade favicon.ico overwrites it.

## Non-Goals
- Dark/light theme toggle (dark only for v1)
- User preferences or settings page
- Real-time streaming of partial responses in the inspector
- WebSocket reconnection indicator in the UI
- Export/download functionality from the dashboard (CLI handles this)
