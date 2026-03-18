# Dashboard UI Overhaul Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the Corvade dashboard from a working prototype into a polished, production-quality dev tool with branding, tab navigation, keyboard shortcuts, animations, and responsive design.

**Architecture:** All changes are in the `dashboard/` Next.js app. No Go backend changes. The dashboard uses Next.js 16 (App Router, static export), Tailwind CSS, React Flow, and Dagre. Components are client-side rendered. The Go binary embeds the built dashboard via `go:embed`.

**Tech Stack:** Next.js 16, TypeScript, Tailwind CSS, React Flow, Dagre

**Spec:** `docs/superpowers/specs/2026-03-18-dashboard-ui-overhaul-design.md`

---

## File Map

### New Files

| File | Responsibility |
|------|---------------|
| `dashboard/public/logo.svg` | Wired Eye logo mark in violet |
| `dashboard/public/favicon.svg` | Simplified eye for browser tab |
| `dashboard/src/components/Header.tsx` | Fixed top bar: logo + wordmark + live stats |
| `dashboard/src/components/TabNav.tsx` | Tab navigation with active state and URL routing |
| `dashboard/src/components/StatsBar.tsx` | Metric cards row (traces, tokens, cost, agents, models) |
| `dashboard/src/components/SkeletonRows.tsx` | Loading skeleton for trace table |
| `dashboard/src/components/EmptyState.tsx` | Reusable empty state with illustration + code snippet |
| `dashboard/src/components/KeyboardHelp.tsx` | Shortcut overlay modal |
| `dashboard/src/components/SessionCard.tsx` | Session card for grid layout |
| `dashboard/src/hooks/useKeyboard.ts` | Keyboard shortcut hook with focus scoping |
| `dashboard/src/components/LayoutShell.tsx` | Client component wrapping Header, TabNav, KeyboardHelp, and children — needed because layout.tsx is a server component |
| `dashboard/src/lib/wsClient.ts` | Singleton CorvadeWS instance — shared across Header and StatsBar to avoid multiple WebSocket connections |
| `dashboard/src/app/sessions/page.tsx` | Sessions tab page |
| `dashboard/src/app/stats/page.tsx` | Stats tab page |

### Modified Files

| File | Changes |
|------|---------|
| `dashboard/src/app/layout.tsx` | Add Header + TabNav, update metadata, remove inline body classes |
| `dashboard/src/app/page.tsx` | Add StatsBar, remove inline header |
| `dashboard/src/app/globals.css` | Skeleton keyframes, custom scrollbar, focus ring utilities, animation classes |
| `dashboard/src/components/Timeline.tsx` | Row density, hover states, relative timestamps, keyboard nav, alternating rows, debounced search, clear filters |
| `dashboard/src/components/DetailInspector.tsx` | Slide animation, token badges on tabs, line numbers in JSON viewer |
| `dashboard/src/components/SessionDiff.tsx` | Summary bar, improved two-column layout |
| `dashboard/src/app/diff/page.tsx` | Session picker comboboxes |
| `dashboard/src/app/session/page.tsx` | Two-panel layout (graph + narrative), session header |
| `dashboard/src/lib/api.ts` | Add fetchStats with time range params |

### Deleted Files

| File | Reason |
|------|--------|
| `dashboard/public/file.svg` | Default Next.js asset |
| `dashboard/public/globe.svg` | Default Next.js asset |
| `dashboard/public/next.svg` | Default Next.js asset |
| `dashboard/public/vercel.svg` | Default Next.js asset |
| `dashboard/public/window.svg` | Default Next.js asset |

Note: `dashboard/public/favicon.ico` is replaced (overwritten), not deleted.

---

## Task 1: Branding Assets & Global CSS

**Files:**
- Create: `dashboard/public/logo.svg`, `dashboard/public/favicon.svg`
- Modify: `dashboard/public/favicon.ico` (replace)
- Modify: `dashboard/src/app/globals.css`
- Delete: `dashboard/public/file.svg`, `globe.svg`, `next.svg`, `vercel.svg`, `window.svg`

- [ ] **Step 1: Create the Wired Eye logo SVG**

Create `dashboard/public/logo.svg` — the D3 Wired Eye mark. An eye shape with circuit/topology paths extending from it. Violet (#8b5cf6) on transparent background. Viewbox 80x80.

Key elements:
- Eye outline (almond shape)
- Diamond-shaped pupil
- Solid iris fill in violet
- Circuit traces extending from corners (4 paths going up/down/left/right)
- Small terminal nodes (circles) at circuit path ends
- Traces from top of eye branching left and right

- [ ] **Step 2: Create the simplified favicon SVG**

Create `dashboard/public/favicon.svg` — simplified version of the eye. Just the eye shape + diamond pupil, no circuit traces (too small at 16px). Viewbox 32x32.

- [ ] **Step 3: Replace favicon.ico**

The existing favicon is at `dashboard/src/app/favicon.ico` (Next.js App Router convention). Replace it with a Corvade favicon generated from the favicon SVG. Overwrite `dashboard/src/app/favicon.ico` in place. Also add `dashboard/public/favicon.svg` as the primary favicon (referenced in layout.tsx), keeping the .ico as a fallback.

- [ ] **Step 4: Delete default Next.js assets**

```bash
rm dashboard/public/file.svg dashboard/public/globe.svg dashboard/public/next.svg dashboard/public/vercel.svg dashboard/public/window.svg
```

- [ ] **Step 5: Update globals.css**

Replace the current globals.css with Corvade-specific styles:

```css
@import "tailwindcss";

@theme inline {
  --font-sans: var(--font-geist-sans);
  --font-mono: var(--font-geist-mono);
}

/* Skeleton loading animation */
@keyframes skeleton-pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 0.7; }
}

.skeleton {
  animation: skeleton-pulse 1.5s ease-in-out infinite;
  background: #27272a;
  border-radius: 4px;
}

/* New trace flash animation */
@keyframes trace-flash {
  0% { background-color: rgba(139, 92, 246, 0.1); }
  100% { background-color: transparent; }
}

.trace-flash {
  animation: trace-flash 1s ease-out;
}

/* Stats counter update flash */
.stat-flash {
  transition: opacity 200ms ease;
}

.stat-flash.updating {
  opacity: 0.7;
}

/* Violet focus ring for inputs */
input:focus, select:focus, textarea:focus {
  outline: none;
  box-shadow: 0 0 0 2px rgba(139, 92, 246, 0.3);
  border-color: #8b5cf6;
}

/* Custom scrollbar */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: #09090b;
}

::-webkit-scrollbar-thumb {
  background: #3f3f46;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #52525b;
}

/* Tabular nums for stats/numbers */
.tabular-nums {
  font-variant-numeric: tabular-nums;
}

/* Detail inspector animation */
.inspector-enter {
  max-height: 0;
  opacity: 0;
  overflow: hidden;
  transition: max-height 150ms ease, opacity 150ms ease;
}

.inspector-enter.open {
  max-height: 600px;
  opacity: 1;
}

body {
  background: #09090b;
  color: #fafafa;
}
```

- [ ] **Step 6: Verify dashboard builds**

```bash
cd dashboard && npm run build
```

- [ ] **Step 7: Commit**

```bash
git add dashboard/public/ dashboard/src/app/globals.css
git commit -m "feat: add Corvade branding assets, custom CSS, and remove Next.js defaults"
```

---

## Task 2: Layout Shell — Header & Tab Navigation

**Files:**
- Create: `dashboard/src/components/Header.tsx`, `dashboard/src/components/TabNav.tsx`
- Modify: `dashboard/src/app/layout.tsx`
- Modify: `dashboard/src/app/page.tsx`

- [ ] **Step 1: Create Header component**

`dashboard/src/components/Header.tsx` — client component:
- Left side: inline SVG of the wired eye logo (24px) + "Corvade" text (font-semibold, zinc-50)
- Right side: live stats fetched from `/api/stats` — format: "{n} traces · ${cost} · {relative time}"
- Fixed to top, full width, 48px height, zinc-900 bg, zinc-800 bottom border
- Stats refresh on mount and update via WebSocket `trace:new` events (increment counter, recalc cost)
- **WebSocket singleton:** Create `dashboard/src/lib/wsClient.ts` that exports a single `CorvadeWS` instance: `export const ws = new CorvadeWS()`. Header calls `ws.connect()` in a `useEffect` on mount and `ws.disconnect()` on unmount. StatsBar (Task 4) subscribes to the same singleton via `ws.on('trace:new', handler)` — no second connection.
- On WebSocket disconnect: values freeze (no error shown). Auto-reconnect is handled by CorvadeWS internals.
- Counter updates: add/remove `.stat-flash.updating` class briefly on value change

- [ ] **Step 2: Create TabNav component**

`dashboard/src/components/TabNav.tsx` — client component:
- Four tabs: Traces (`/`), Sessions (`/sessions`), Diff (`/diff`), Stats (`/stats`)
- Uses `usePathname()` from `next/navigation` to determine active tab
- Active: violet-500 bottom border (2px), zinc-50 text
- Inactive: transparent bottom border, zinc-500 text, hover zinc-300
- Tabs are `<a>` links (proper navigation)
- Full width, zinc-950 bg, zinc-800 bottom border
- Instant switch (no transition)

- [ ] **Step 3: Update layout.tsx**

Update `dashboard/src/app/layout.tsx`:
- Change metadata: title "Corvade", description "See what your agents see."
- Add favicon link pointing to `/favicon.svg`
- Render `<Header />` and `<TabNav />` above `{children}`
- Remove the default Geist font variables from body class if unused, keep antialiased
- Add padding-top to account for fixed header height

- [ ] **Step 4: Update page.tsx (Traces tab)**

Update `dashboard/src/app/page.tsx`:
- Remove the inline header (h1 "Corvade" + subtitle) — now in Header component
- Keep just `<Timeline />`
- Remove the outer `<main>` wrapper — layout.tsx handles the shell

- [ ] **Step 5: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 6: Commit**

```bash
git add dashboard/src/
git commit -m "feat: add Header and TabNav layout shell with live stats"
```

---

## Task 3: Keyboard Hook & Help Overlay

**Files:**
- Create: `dashboard/src/hooks/useKeyboard.ts`, `dashboard/src/components/KeyboardHelp.tsx`
- Modify: `dashboard/src/app/layout.tsx` (wire in keyboard hook + overlay)

- [ ] **Step 1: Create useKeyboard hook**

`dashboard/src/hooks/useKeyboard.ts`:
- Accepts a map of `{ key: handler }` bindings
- Registers a global `keydown` listener
- **Focus scoping:** If `document.activeElement` is an `input`, `textarea`, or `select`, ignore all shortcuts (return early)
- Handles modifier keys: only fire on plain key press (no Ctrl/Alt/Meta)
- Cleans up listener on unmount
- Returns nothing (side-effect only hook)

- [ ] **Step 2: Create KeyboardHelp overlay**

`dashboard/src/components/KeyboardHelp.tsx`:
- Props: `{ open: boolean; onClose: () => void }`
- Centered modal with zinc-950/80 backdrop
- Fade-in: opacity transition 100ms
- Three groups of shortcuts:
  - **Global:** `1`-`4` switch tabs, `?` toggle help
  - **Traces:** `j`/`k` navigate, `Enter`/`Space` toggle inspector, `Esc` close, `/` focus search
  - **Inspector:** `Tab` switch request/response (future)
- Each shortcut: `<kbd>` styled key + description
- Close on `Escape` or clicking backdrop

- [ ] **Step 3: Wire into layout.tsx**

In layout.tsx:
- Add state for keyboard help overlay visibility
- Use `useKeyboard` hook for global shortcuts: `?` toggles help overlay, `1`-`4` navigate to tabs (use `router.push`)
- Render `<KeyboardHelp />` conditionally

Note: Since layout.tsx is a server component by default, extract the interactive part into `dashboard/src/components/LayoutShell.tsx` — a client component with this interface:

```tsx
// Props: { children: React.ReactNode }
// Renders: <Header />, <TabNav />, <KeyboardHelp />, {children}
// Manages: keyboard help overlay state, global keyboard shortcuts (?, 1-4)
// Uses: useKeyboard hook, router.push for tab switching
```

layout.tsx remains a server component and simply renders `<LayoutShell>{children}</LayoutShell>`. This separation is required by Next.js App Router — server components cannot use hooks or state.

- [ ] **Step 4: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 5: Commit**

```bash
git add dashboard/src/
git commit -m "feat: add keyboard shortcut hook and help overlay"
```

---

## Task 4: StatsBar, SkeletonRows, EmptyState Components

**Files:**
- Create: `dashboard/src/components/StatsBar.tsx`, `dashboard/src/components/SkeletonRows.tsx`, `dashboard/src/components/EmptyState.tsx`

- [ ] **Step 1: Create StatsBar component**

`dashboard/src/components/StatsBar.tsx`:
- Fetches from `/api/stats` on mount
- Renders 5 metric cards in a flex row: traces count, total tokens, total cost, unique agents, unique models
- Each card: zinc-900 bg, zinc-800 border, rounded
- Number in violet-400 with `tabular-nums`, label in zinc-400 text-xs uppercase
- `.stat-flash` class on each number element for update animation

- [ ] **Step 2: Create SkeletonRows component**

`dashboard/src/components/SkeletonRows.tsx`:
- Props: `{ rows?: number }` (default 5)
- Renders N table rows where each cell is a `<div className="skeleton">` with height matching the column content
- Columns match the trace table: time (80px wide), model (120px), agent (100px), tokens (60px), cost (80px), latency (60px), status (40px)

- [ ] **Step 3: Create EmptyState component**

`dashboard/src/components/EmptyState.tsx`:
- Props: `{ title: string; description: string; code?: string }`
- Centered layout with padding
- Inline SVG illustration: simplified wired eye with dashed connection lines (zinc-700 stroke)
- Title in zinc-200, text-lg
- Description in zinc-500, text-sm
- If `code` prop provided: monospace code block with zinc-800 bg, zinc-300 text, and a copy button (copies to clipboard on click, shows "Copied!" briefly)

- [ ] **Step 4: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 5: Commit**

```bash
git add dashboard/src/components/StatsBar.tsx dashboard/src/components/SkeletonRows.tsx dashboard/src/components/EmptyState.tsx
git commit -m "feat: add StatsBar, SkeletonRows, and EmptyState components"
```

---

## Task 5: Timeline Overhaul

**Files:**
- Modify: `dashboard/src/components/Timeline.tsx`
- Modify: `dashboard/src/app/page.tsx`

- [ ] **Step 1: Update page.tsx to include StatsBar**

```tsx
'use client';
import Timeline from '@/components/Timeline';
import StatsBar from '@/components/StatsBar';

export default function Home() {
  return (
    <div className="p-6 max-w-7xl mx-auto space-y-4">
      <StatsBar />
      <Timeline />
    </div>
  );
}
```

- [ ] **Step 2: Overhaul Timeline.tsx**

Major changes to `dashboard/src/components/Timeline.tsx`:

**Filter bar:**
- Add violet focus ring (handled by globals.css)
- Add "Clear filters" button: only visible when any filter has a value. Text button: "Clear" in zinc-500, hover zinc-300
- Debounce search: use a 300ms timeout before triggering the API fetch

**Table:**
- Tighter row padding: `py-2` instead of `py-3`
- Alternating row backgrounds: even rows `bg-zinc-950`, odd rows `bg-zinc-900/30`
- Hover state: `hover:bg-zinc-800/50` + 2px violet left border via `hover:border-l-2 hover:border-l-violet-500` (add `transition-all duration-100`)
- Selected row: `bg-zinc-800/70 border-l-2 border-l-violet-500`
- Relative timestamps: replace `toLocaleTimeString` with a `timeAgo` function ("2s ago", "5m ago", "2h ago", "3d ago"). Full ISO timestamp as `title` attribute for hover tooltip.
- Cost: keep `$0.000425` format (already correct)

**Loading state:**
- Replace "Loading..." text with `<SkeletonRows rows={5} />`

**Empty state:**
- Replace "No traces captured yet..." with `<EmptyState title="No traces yet" description="Point your agents at the proxy to get started" code="OPENAI_BASE_URL=http://localhost:4400/v1" />`

**Keyboard navigation:**
- Track `selectedIndex` state (number, -1 = none selected)
- `j` / `↓` increments index (wraps at end)
- `k` / `↑` decrements index (wraps at start)
- `Enter` / `Space` toggles detail inspector for selected trace
- `Escape` closes inspector and clears selection
- `/` focuses the search input (use a ref)
- Selected row scrolls into view via `scrollIntoView({ block: 'nearest' })`
- **Event propagation:** The Timeline keyboard handler should call `event.stopPropagation()` for handled keys (j/k/Enter/Space/Escape//) to prevent the global LayoutShell handler from also processing them. The existing DetailInspector `Escape` listener should be removed — Timeline now handles `Escape` centrally.

- [ ] **Step 3: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add dashboard/src/components/Timeline.tsx dashboard/src/app/page.tsx
git commit -m "feat: overhaul Timeline with stats bar, keyboard nav, skeletons, and polish"
```

---

## Task 6: Detail Inspector Polish

**Files:**
- Modify: `dashboard/src/components/DetailInspector.tsx`

- [ ] **Step 1: Add slide animation**

**DOM strategy change required:** The current `Timeline.tsx` conditionally mounts `<DetailInspector>` (`{selectedTraceId && ...}`). CSS transitions don't animate on mount. Change the approach: always render `<DetailInspector>` when a trace is selected, but pass an `open` boolean prop. Inside DetailInspector, wrap content in a div with `.inspector-enter` class. Use a `useEffect` with a one-tick `requestAnimationFrame` delay to add the `.open` class after mount, triggering the CSS max-height + opacity transition (150ms ease, max-height 600px). On close, remove `.open` class first, wait 150ms, then unmount.

- [ ] **Step 2: Add token count badges on tabs**

The Request and Response tab buttons should show token counts:
- "Request ({tokens_prompt} tok)" and "Response ({tokens_completion} tok)"
- Badge: small zinc-700 bg, zinc-300 text, rounded, inline after tab label

- [ ] **Step 3: Add line numbers to JSON viewer**

Modify the JsonViewer to render line numbers:
- Split formatted JSON by newlines
- Render each line as a flex row: `<span className="text-zinc-600 text-right w-8 mr-3 select-none">{lineNum}</span><span>{content}</span>`
- Line numbers are not selectable (prevent copy)

- [ ] **Step 4: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 5: Commit**

```bash
git add dashboard/src/components/DetailInspector.tsx
git commit -m "feat: polish DetailInspector with slide animation, token badges, and line numbers"
```

---

## Task 7: Sessions Tab

**Files:**
- Create: `dashboard/src/components/SessionCard.tsx`, `dashboard/src/app/sessions/page.tsx`
- Modify: `dashboard/src/lib/api.ts`

- [ ] **Step 1: Add fetchSessions to API client if not already present**

Verify `dashboard/src/lib/api.ts` has `fetchSessions`. It should already exist. If not, add it.

- [ ] **Step 2: Create SessionCard component**

`dashboard/src/components/SessionCard.tsx`:
- Props: session object (id, agent, trace_count, total_tokens, total_cost, status, start_time, end_time)
- Card: zinc-900 bg, zinc-800 border, rounded-lg, p-4
- Agent name: text-lg font-semibold zinc-50 (or "Unknown Agent" in zinc-500 if null)
- Stats row: "{trace_count} traces · {total_tokens} tok · ${total_cost}"
- Status badge: small pill — active (violet bg/text), completed (green), error (red)
- Time range: start → end or start → "ongoing"
- Model chips: zinc-800 bg, zinc-300 text, text-xs, rounded, px-2 py-0.5 (requires fetching distinct models — can derive from session or skip for v1)
- Hover: border-color transitions to violet (100ms)
- Links to `/session?id={session.id}`

- [ ] **Step 3: Create Sessions page**

`dashboard/src/app/sessions/page.tsx` — client component:
- Fetch sessions from API on mount
- Responsive grid: `grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4`
- Render SessionCard for each session
- Empty state: `<EmptyState title="No sessions yet" description="Sessions are created when agents tag requests with X-Corvade-Session headers." />`
- Loading: grid of skeleton cards (3 cards with `.skeleton` class, matching card dimensions)

- [ ] **Step 4: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 5: Commit**

```bash
git add dashboard/src/components/SessionCard.tsx dashboard/src/app/sessions/page.tsx dashboard/src/lib/api.ts
git commit -m "feat: add Sessions tab with card grid and session cards"
```

---

## Task 8: Session Detail Page Overhaul

**Files:**
- Modify: `dashboard/src/app/session/page.tsx`

- [ ] **Step 1: Redesign session detail page**

Update `dashboard/src/app/session/page.tsx`. **Remove the existing `<main className="min-h-screen bg-zinc-950 ...">` wrapper** — the layout shell now handles background/padding.
- Session header: agent name, truncated session ID, status badge, time range, total cost
- Two-panel flex layout:
  - Left (`flex: 3`): TopologyGraph component in a container with explicit height `calc(100vh - 200px)` for React Flow
  - Right (`flex: 2`): Segmented control (Graph | Narrative) toggling between the existing TopologyGraph focus states and NarrativeView
- Segmented control: two buttons, zinc-800 bg normally, violet-500 bg when active, rounded
- Fetch session details and graph data on mount

- [ ] **Step 2: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add dashboard/src/app/session/page.tsx
git commit -m "feat: overhaul session detail page with two-panel layout"
```

---

## Task 9: Diff Tab Overhaul

**Files:**
- Modify: `dashboard/src/app/diff/page.tsx`
- Modify: `dashboard/src/components/SessionDiff.tsx`

- [ ] **Step 1: Add session picker comboboxes to diff page**

Update `dashboard/src/app/diff/page.tsx`:
- Replace plain text inputs with combobox inputs:
  - Text input that filters sessions as you type (by agent name or ID prefix)
  - Dropdown below showing matching sessions: agent name + trace count + relative time
  - Clicking a dropdown item or pasting a full ID selects it
  - Selected state shows: agent name + truncated ID
- "vs" label between the two inputs
- "Compare" button: violet bg, disabled state when both aren't selected
- Fetch sessions list on mount for dropdown options
- Before comparison is triggered, show `<EmptyState title="Compare agent sessions" description="Compare two agent sessions to see where they diverged. Run the same agent twice with different models or prompts." />` in the results area

- [ ] **Step 2: Improve SessionDiff layout**

Update `dashboard/src/components/SessionDiff.tsx`:
- Add summary bar at top: "N matched · N diverged · N left-only · N right-only" in zinc-400
- Matched pairs: two-column grid with match % as a colored progress bar (green >80%, yellow 50-80%)
- Divergence points: add a small pulsing violet dot (CSS animation) before each
- Left-only/Right-only: keep red/green tints, improve spacing
- Empty/no-diff state message

- [ ] **Step 3: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add dashboard/src/app/diff/page.tsx dashboard/src/components/SessionDiff.tsx
git commit -m "feat: overhaul Diff tab with session picker comboboxes and improved layout"
```

---

## Task 10: Stats Tab

**Files:**
- Create: `dashboard/src/app/stats/page.tsx`
- Modify: `dashboard/src/lib/api.ts`

- [ ] **Step 1: Update API client**

Add to `dashboard/src/lib/api.ts`:
- Ensure `fetchStats` accepts optional `from` param and passes it as query string
- Add `fetchTraces` variant that returns by-model and by-agent breakdowns (or reuse existing stats endpoint which already returns `by_model` and `by_agent`)

- [ ] **Step 2: Create Stats page**

`dashboard/src/app/stats/page.tsx` — client component:

**Time range selector:**
- Horizontal button group: Last hour | Last 24h | Last 7d | Last 30d | All time
- Active: violet-500 bg, white text. Inactive: zinc-800 bg, zinc-400 text
- On click: compute ISO 8601 `from` timestamp relative to now, fetch stats with that param

**Metric cards:**
- Same style as StatsBar but larger: Total Traces, Total Cost, Total Tokens, Avg Cost/Request
- 4-card grid row

**By Model breakdown:**
- Section heading: "By Model"
- For each model in `by_model`: row with model name (left), CSS horizontal bar proportional to max count (middle, violet bg), count (right)
- Bar width: `(count / maxCount) * 100%`

**By Agent breakdown:**
- Same layout as By Model

**Cost Over Time sparkline:**
- Section heading: "Cost Over Time"
- SVG element ~full width, 60px tall
- Draw a simple `<path>` from data points
- Purely decorative: no axes, no tooltips, no data points
- If only 1 data point or no data: show a flat line
- Data source: call `fetchTraces({ limit: '200' })`, group traces by hour (using `created_at`), sum cost per bucket, and render the SVG path from those data points. This is computed client-side — no new API endpoint needed. If fewer than 2 data points, show a flat line.

**Empty state:**
- `<EmptyState title="No data yet" description="Start capturing traces to see usage analytics." />`

- [ ] **Step 3: Verify**

```bash
cd dashboard && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add dashboard/src/app/stats/page.tsx dashboard/src/lib/api.ts
git commit -m "feat: add Stats tab with time range selector, metric cards, and breakdowns"
```

---

## Task 11: Responsive Design

**Files:**
- Modify: `dashboard/src/components/TabNav.tsx`
- Modify: `dashboard/src/components/Timeline.tsx`
- Modify: `dashboard/src/components/SessionDiff.tsx`
- Modify: `dashboard/src/app/stats/page.tsx`

- [ ] **Step 1: Make tabs horizontally scrollable on small screens**

In `TabNav.tsx`: add `overflow-x-auto` on the tab container for `<768px`. Tabs should not wrap.

- [ ] **Step 2: Hide Latency and Tokens columns on small screens**

In `Timeline.tsx`: add `hidden sm:table-cell` class to Latency and Tokens `<th>` and `<td>` elements.

- [ ] **Step 3: Stack diff panels on medium screens**

In `SessionDiff.tsx`: change the two-column layout to use `flex-col md:flex-row` so panels stack vertically on small screens.

- [ ] **Step 4: Responsive stat cards and session cards**

In `stats/page.tsx`: metric cards grid should be `grid-cols-2 md:grid-cols-4`. In `sessions/page.tsx`: already responsive from Task 7 (`grid-cols-1 md:grid-cols-2 xl:grid-cols-3`).

- [ ] **Step 5: Verify at different widths**

Resize browser window and check each page renders correctly at ~1280px, ~900px, and ~600px widths.

- [ ] **Step 6: Commit**

```bash
git add dashboard/src/
git commit -m "feat: add responsive design for tabs, table, diff, and stats"
```

---

## Task 12: Final Build & Integration

**Files:**
- Modify: `Makefile` (rebuild dashboard embed)

- [ ] **Step 1: Full dashboard build**

```bash
cd dashboard && npm run build
```

Verify all pages build without errors.

- [ ] **Step 2: Rebuild Go binary with new dashboard**

```bash
make build
```

- [ ] **Step 3: Test demo mode end-to-end**

```bash
rm -f ~/.corvade/corvade.db
./bin/corvade start --demo
```

Open http://localhost:4401 and verify:
- Header with logo and live stats
- Tab navigation works (Traces, Sessions, Diff, Stats)
- Traces tab shows 7 demo traces with proper formatting
- Clicking a trace opens detail inspector with slide animation
- Keyboard shortcuts work (j/k, Enter, Escape, ?, 1-4)
- Sessions tab shows empty state (demo data has no sessions)
- Diff tab shows combobox inputs
- Stats tab shows metric cards and breakdowns

- [ ] **Step 4: Run Go tests**

```bash
go test ./... -v
```

All tests must pass.

- [ ] **Step 5: Commit**

```bash
git add .
git commit -m "feat: complete dashboard UI overhaul with full rebuild"
```

---

## Summary

| Task | Description | Key Files |
|------|------------|-----------|
| 1 | Branding assets & global CSS | logo.svg, favicon, globals.css |
| 2 | Layout shell — Header & TabNav | Header.tsx, TabNav.tsx, layout.tsx |
| 3 | Keyboard hook & help overlay | useKeyboard.ts, KeyboardHelp.tsx |
| 4 | StatsBar, SkeletonRows, EmptyState | 3 new components |
| 5 | Timeline overhaul | Timeline.tsx, page.tsx |
| 6 | Detail Inspector polish | DetailInspector.tsx |
| 7 | Sessions tab | SessionCard.tsx, sessions/page.tsx |
| 8 | Session detail page overhaul | session/page.tsx |
| 9 | Diff tab overhaul | diff/page.tsx, SessionDiff.tsx |
| 10 | Stats tab | stats/page.tsx |
| 11 | Responsive design | Multiple files |
| 12 | Final build & integration | Makefile, full rebuild |
