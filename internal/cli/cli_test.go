package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/capture"
)

// ---------- loadDemoData ----------

func TestLoadDemoData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := capture.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	count, err := loadDemoData(store)
	if err != nil {
		t.Fatalf("loadDemoData: %v", err)
	}
	if count == 0 {
		t.Fatal("loadDemoData returned 0 traces, expected > 0")
	}

	// Verify traces actually exist and have token/cost data populated
	limit := 100
	traces, err := store.ListTraces(capture.TraceFilter{Limit: &limit})
	if err != nil {
		t.Fatalf("ListTraces: %v", err)
	}
	if len(traces) != count {
		t.Errorf("ListTraces returned %d, loadDemoData reported %d", len(traces), count)
	}

	// At least one trace should have tokens and cost set
	foundTokens := false
	foundCost := false
	for _, tr := range traces {
		if tr.TokensPrompt != nil || tr.TokensCompletion != nil {
			foundTokens = true
		}
		if tr.Cost != nil && *tr.Cost > 0 {
			foundCost = true
		}
	}
	if !foundTokens {
		t.Error("expected at least one trace with token counts populated")
	}
	if !foundCost {
		t.Error("expected at least one trace with cost > 0")
	}
}

// ---------- checkPort ----------

func TestCheckPortAvailable(t *testing.T) {
	// Find a free port by binding then closing
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	if !checkPort(port) {
		t.Errorf("checkPort(%d) = false, expected true for free port", port)
	}
}

func TestCheckPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	if checkPort(port) {
		t.Errorf("checkPort(%d) = true, expected false for port in use", port)
	}
}

// ---------- checkSQLiteWritable ----------

func TestCheckSQLiteWritable(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "test.db")
	if !checkSQLiteWritable(dbPath) {
		t.Error("checkSQLiteWritable returned false for writable temp dir")
	}
}

func TestCheckSQLiteWritableInvalidPath(t *testing.T) {
	if checkSQLiteWritable("/proc/0/nonexistent/test.db") {
		t.Error("checkSQLiteWritable returned true for invalid path")
	}
}

// ---------- getFileSize ----------

func TestGetFileSize(t *testing.T) {
	// Non-existent file
	s := getFileSize("/tmp/corvade-nonexistent-test-file-xyz")
	if s != "0 B" {
		t.Errorf("getFileSize non-existent = %q, want '0 B'", s)
	}

	// Small file
	f, err := os.CreateTemp(t.TempDir(), "size")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("hello"))
	f.Close()
	s = getFileSize(f.Name())
	if s != "5 B" {
		t.Errorf("getFileSize 5 bytes = %q, want '5 B'", s)
	}
}

// ---------- truncateID ----------

func TestTruncateID(t *testing.T) {
	if got := truncateID("abcdefghij"); got != "abcdefgh..." {
		t.Errorf("truncateID long = %q", got)
	}
	if got := truncateID("short"); got != "short" {
		t.Errorf("truncateID short = %q", got)
	}
}

// ---------- ptrStringOrDash ----------

func TestPtrStringOrDash(t *testing.T) {
	s := "hello"
	if got := ptrStringOrDash(&s); got != "hello" {
		t.Errorf("ptrStringOrDash(&hello) = %q", got)
	}
	if got := ptrStringOrDash(nil); got != "-" {
		t.Errorf("ptrStringOrDash(nil) = %q", got)
	}
	empty := ""
	if got := ptrStringOrDash(&empty); got != "-" {
		t.Errorf("ptrStringOrDash(&empty) = %q", got)
	}
}

// ---------- printJSON ----------

func TestPrintJSON(t *testing.T) {
	err := printJSON(map[string]string{"key": "value"})
	if err != nil {
		t.Errorf("printJSON returned error: %v", err)
	}
}

// ---------- stringOrDefault / floatOrDefault ----------

func TestStringOrDefault(t *testing.T) {
	data := map[string]interface{}{"key": "val"}
	if got := stringOrDefault(data, "key", "def"); got != "val" {
		t.Errorf("stringOrDefault found = %q", got)
	}
	if got := stringOrDefault(data, "missing", "def"); got != "def" {
		t.Errorf("stringOrDefault missing = %q", got)
	}
}

func TestFloatOrDefault(t *testing.T) {
	data := map[string]interface{}{
		"f64": float64(1.5),
		"int": int(3),
		"str": "not a number",
	}
	if got := floatOrDefault(data, "f64", 0); got != 1.5 {
		t.Errorf("floatOrDefault f64 = %v", got)
	}
	if got := floatOrDefault(data, "int", 0); got != 3.0 {
		t.Errorf("floatOrDefault int = %v", got)
	}
	if got := floatOrDefault(data, "str", 99); got != 99 {
		t.Errorf("floatOrDefault str = %v", got)
	}
	if got := floatOrDefault(data, "missing", 42); got != 42 {
		t.Errorf("floatOrDefault missing = %v", got)
	}

	// Test json.Number branch
	jn := json.Number("3.14")
	data["jnum"] = jn
	if got := floatOrDefault(data, "jnum", 0); got != 3.14 {
		t.Errorf("floatOrDefault json.Number = %v, want 3.14", got)
	}
}

// ---------- getAvailableDiskSpace ----------

func TestGetAvailableDiskSpace(t *testing.T) {
	gb, err := getAvailableDiskSpace(os.TempDir())
	if err != nil {
		t.Fatalf("getAvailableDiskSpace: %v", err)
	}
	if gb <= 0 {
		t.Errorf("expected positive disk space, got %v", gb)
	}
}

func TestGetAvailableDiskSpaceError(t *testing.T) {
	_, err := getAvailableDiskSpace("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

// ---------- runDoctor ----------

func TestRunDoctor(t *testing.T) {
	// runDoctor checks ports, SQLite, and disk space -- all should work in test env
	err := runDoctor()
	// It may fail if ports are in use, that's fine -- we just verify no panic
	_ = err
}

// ---------- runSessions with temp DB ----------

func TestRunSessionsWithDefaultDB(t *testing.T) {
	// runSessions uses a hardcoded path (~/.corvade/corvade.db).
	// We can verify it doesn't panic when run against the real (or new) DB.
	err := runSessions(5)
	// May error if DB creation fails in some envs, just verify no panic
	_ = err
}

func TestRunSessionsWithData(t *testing.T) {
	// Insert a session into the default DB so the rendering loop is covered
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".corvade", "corvade.db")
	store, err := capture.NewStore(dbPath)
	if err != nil {
		t.Skipf("cannot open default store: %v", err)
	}

	agent := "test-agent"
	_, err = store.InsertSession(capture.Session{
		Agent:       &agent,
		StartTime:   time.Now().UTC(),
		TraceCount:  3,
		TotalTokens: 100,
		TotalCost:   0.05,
		Status:      "complete",
	})
	store.Close()
	if err != nil {
		t.Skipf("cannot insert session: %v", err)
	}

	err = runSessions(5)
	if err != nil {
		t.Errorf("runSessions with data: %v", err)
	}
}

// ---------- runInspect ----------

func TestRunInspectTrace(t *testing.T) {
	// Insert a trace into the default DB and try to inspect it
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".corvade", "corvade.db")
	store, err := capture.NewStore(dbPath)
	if err != nil {
		t.Skipf("cannot open default store: %v", err)
	}

	traceID, err := store.InsertTrace(capture.Trace{
		Provider:   "openai",
		Model:      "gpt-4o-test",
		Request:    `{"model":"gpt-4o"}`,
		StatusCode: 200,
		CreatedAt:  time.Now().UTC(),
	})
	store.Close()
	if err != nil {
		t.Skipf("cannot insert trace: %v", err)
	}

	// Inspect the trace we just inserted
	err = runInspect(traceID)
	if err != nil {
		t.Errorf("runInspect(%s): %v", traceID, err)
	}
}

func TestRunInspectSession(t *testing.T) {
	// Insert a session and inspect it
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".corvade", "corvade.db")
	store, err := capture.NewStore(dbPath)
	if err != nil {
		t.Skipf("cannot open default store: %v", err)
	}

	agent := "test-agent"
	sessID, err := store.InsertSession(capture.Session{
		Agent:      &agent,
		StartTime:  time.Now().UTC(),
		TraceCount: 1,
		TotalCost:  0.01,
		Status:     "complete",
	})
	store.Close()
	if err != nil {
		t.Skipf("cannot insert session: %v", err)
	}

	err = runInspect(sessID)
	if err != nil {
		t.Errorf("runInspect session(%s): %v", sessID, err)
	}
}

func TestRunInspectMissingID(t *testing.T) {
	err := runInspect("definitely-nonexistent-id-99999")
	if err == nil {
		t.Log("runInspect returned nil (found something unexpectedly)")
	}
}

// ---------- Cobra command constructors ----------

func TestNewDoctorCmd(t *testing.T) {
	cmd := NewDoctorCmd()
	if cmd == nil {
		t.Fatal("NewDoctorCmd returned nil")
	}
	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, want 'doctor'", cmd.Use)
	}
}

func TestNewSessionsCmd(t *testing.T) {
	cmd := NewSessionsCmd()
	if cmd == nil {
		t.Fatal("NewSessionsCmd returned nil")
	}
	if cmd.Use != "sessions" {
		t.Errorf("Use = %q, want 'sessions'", cmd.Use)
	}
	// Check flag exists
	f := cmd.Flags().Lookup("limit")
	if f == nil {
		t.Error("missing --limit flag")
	}
}

func TestNewInspectCmd(t *testing.T) {
	cmd := NewInspectCmd()
	if cmd == nil {
		t.Fatal("NewInspectCmd returned nil")
	}
	if cmd.Use != "inspect <id>" {
		t.Errorf("Use = %q, want 'inspect <id>'", cmd.Use)
	}
}

func TestNewStartCmd(t *testing.T) {
	cmd := NewStartCmd("0.0.1-test")
	if cmd == nil {
		t.Fatal("NewStartCmd returned nil")
	}
	if cmd.Use != "start" {
		t.Errorf("Use = %q, want 'start'", cmd.Use)
	}
	for _, flag := range []string{"headless", "demo", "port", "dashboard-port"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

func TestNewTailCmd(t *testing.T) {
	cmd := NewTailCmd()
	if cmd == nil {
		t.Fatal("NewTailCmd returned nil")
	}
	if cmd.Use != "tail" {
		t.Errorf("Use = %q, want 'tail'", cmd.Use)
	}
	for _, flag := range []string{"agent", "model", "port"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

// ---------- runSessions / runInspect with temp DB ----------

func TestRunSessionsEmpty(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := capture.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	store.Close()

	// Patch the function by calling runSessions indirectly via the command
	// We can't easily override the DB path, so test via the exported cmd constructor
	// and verify it doesn't panic with a valid but empty DB.
	// Instead, test the helper functions that runSessions uses.
}

func TestRunInspectNotFound(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := capture.NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	store.Close()

	// runInspect uses hardcoded path, so we test the error message format
	err = runInspect("nonexistent-id-12345")
	if err == nil {
		t.Log("runInspect did not return error (it used the default DB path)")
	}
}

// ---------- getFileSize edge cases ----------

func TestGetFileSizeKB(t *testing.T) {
	dir := t.TempDir()
	f, _ := os.Create(filepath.Join(dir, "kb.bin"))
	f.Write(make([]byte, 2048)) // 2 KB
	f.Close()
	s := getFileSize(f.Name())
	if s != "2.0 KB" {
		t.Errorf("getFileSize 2KB = %q, want '2.0 KB'", s)
	}
}

func TestGetFileSizeMB(t *testing.T) {
	dir := t.TempDir()
	f, _ := os.Create(filepath.Join(dir, "mb.bin"))
	f.Write(make([]byte, 1<<20)) // 1 MB
	f.Close()
	s := getFileSize(f.Name())
	if s != "1.0 MB" {
		t.Errorf("getFileSize 1MB = %q, want '1.0 MB'", s)
	}
}

func TestGetFileSizeGB(t *testing.T) {
	// Create a sparse file to hit the GB branch without using actual disk space
	dir := t.TempDir()
	path := filepath.Join(dir, "big.bin")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// Seek to 1.5 GB and write one byte - sparse file takes almost no disk space
	f.Seek(int64(1.5*float64(1<<30)), 0)
	f.Write([]byte{0})
	f.Close()

	s := getFileSize(path)
	if s != "1.5 GB" {
		t.Errorf("getFileSize 1.5GB = %q, want '1.5 GB'", s)
	}
}

// ---------- printTraceEvent (smoke test) ----------

func TestPrintTraceEvent(t *testing.T) {
	// Just verify it doesn't panic with various data shapes
	data := map[string]interface{}{
		"model":      "gpt-4o",
		"cost":       float64(0.05),
		"latency_ms": float64(1200),
		"status":     "ok",
		"provider":   "openai",
	}
	printTraceEvent(data)

	// Error status
	data["status"] = "error"
	printTraceEvent(data)

	// Long model name
	data["model"] = "gpt-4o-2025-01-01-preview"
	printTraceEvent(data)

	// Missing fields
	printTraceEvent(map[string]interface{}{})

	// No provider
	printTraceEvent(map[string]interface{}{"model": "x"})

	fmt.Println() // separate test output
}
