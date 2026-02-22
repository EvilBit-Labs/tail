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
