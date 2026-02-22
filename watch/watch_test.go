package watch

import "testing"

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
