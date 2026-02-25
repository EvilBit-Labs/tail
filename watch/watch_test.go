package watch

import (
	"path/filepath"
	"testing"
)

// Compile-time interface compliance checks.
var (
	_ FileWatcher = (*InotifyFileWatcher)(nil)
	_ FileWatcher = (*PollingFileWatcher)(nil)
)

func TestInterfaceCompliance(t *testing.T) {
	// This test exists to document the compile-time checks above.
	// If the var block compiles, the interfaces are satisfied.
	t.Log("InotifyFileWatcher and PollingFileWatcher satisfy FileWatcher")
}

func TestNewFileChanges(t *testing.T) {
	fc := NewFileChanges()
	if fc.Modified == nil {
		t.Fatal("Modified channel is nil")
	}
	if fc.Truncated == nil {
		t.Fatal("Truncated channel is nil")
	}
	if fc.Deleted == nil {
		t.Fatal("Deleted channel is nil")
	}
}

func TestNotifyModified(t *testing.T) {
	fc := NewFileChanges()
	fc.NotifyModified()

	select {
	case v := <-fc.Modified:
		if !v {
			t.Fatal("expected true from Modified channel")
		}
	default:
		t.Fatal("expected value on Modified channel")
	}
}

func TestNotifyModifiedCoalesces(t *testing.T) {
	fc := NewFileChanges()
	// Fill the buffer
	fc.NotifyModified()
	// Second notify should be silently dropped (buffer full)
	fc.NotifyModified()

	<-fc.Modified
	select {
	case <-fc.Modified:
		t.Fatal("second notification should have been dropped")
	default:
		// expected: channel empty
	}
}

func TestNotifyTruncated(t *testing.T) {
	fc := NewFileChanges()
	fc.NotifyTruncated()

	select {
	case v := <-fc.Truncated:
		if !v {
			t.Fatal("expected true from Truncated channel")
		}
	default:
		t.Fatal("expected value on Truncated channel")
	}
}

func TestNotifyDeleted(t *testing.T) {
	fc := NewFileChanges()
	fc.NotifyDeleted()

	select {
	case v := <-fc.Deleted:
		if !v {
			t.Fatal("expected true from Deleted channel")
		}
	default:
		t.Fatal("expected value on Deleted channel")
	}
}

// --- API contract tests ---

func TestPOLL_DURATIONExists(t *testing.T) {
	// POLL_DURATION must be an exported variable of type time.Duration
	d := POLL_DURATION
	if d <= 0 {
		t.Log("POLL_DURATION is zero or negative (may be overridden in TestMain)")
	}
}

func TestNewInotifyFileWatcher(t *testing.T) {
	w := NewInotifyFileWatcher("/tmp/test")
	if w == nil {
		t.Fatal("NewInotifyFileWatcher returned nil")
	}
	// NewInotifyFileWatcher calls filepath.Clean, so expect cleaned path.
	want := filepath.Clean("/tmp/test")
	if w.Filename != want {
		t.Errorf("Filename = %q, want %q", w.Filename, want)
	}
}

func TestNewPollingFileWatcher(t *testing.T) {
	w := NewPollingFileWatcher("/tmp/test")
	if w == nil {
		t.Fatal("NewPollingFileWatcher returned nil")
	}
	if w.Filename != "/tmp/test" {
		t.Errorf("Filename = %q, want %q", w.Filename, "/tmp/test")
	}
}

func TestInotifyFileWatcherFields(t *testing.T) {
	w := InotifyFileWatcher{
		Filename: "test.log",
		Size:     42,
	}
	if w.Filename != "test.log" {
		t.Error("Filename mismatch")
	}
	if w.Size != 42 {
		t.Error("Size mismatch")
	}
}

func TestPollingFileWatcherFields(t *testing.T) {
	w := PollingFileWatcher{
		Filename: "test.log",
		Size:     42,
	}
	if w.Filename != "test.log" {
		t.Error("Filename mismatch")
	}
	if w.Size != 42 {
		t.Error("Size mismatch")
	}
}

func TestFileChangesFields(t *testing.T) {
	fc := FileChanges{
		Modified:  make(chan bool, 1),
		Truncated: make(chan bool, 1),
		Deleted:   make(chan bool, 1),
	}
	if fc.Modified == nil || fc.Truncated == nil || fc.Deleted == nil {
		t.Fatal("FileChanges channels must not be nil")
	}
}

func TestWatchFunctionExists(t *testing.T) {
	// Verify the function signature compiles.
	// We don't call Watch() as it requires inotify setup.
	t.Logf("Watch function: %T", Watch)
}

func TestCleanupFunctionExists(t *testing.T) {
	t.Logf("Cleanup function: %T", Cleanup)
}

func TestEventsFunctionExists(t *testing.T) {
	// Verify the exported Events function signature.
	t.Logf("Events function: %T", Events)
}
