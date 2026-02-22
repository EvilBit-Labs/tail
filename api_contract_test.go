package tail

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/nxadm/tail/ratelimiter"
)

// --- Sentinel values ---

func TestErrStopExists(t *testing.T) {
	if ErrStop == nil {
		t.Fatal("ErrStop must not be nil")
	}
	target := ErrStop
	if !errors.Is(target, ErrStop) {
		t.Fatal("ErrStop must satisfy errors.Is")
	}
}

func TestDefaultLoggerExists(t *testing.T) {
	if DefaultLogger == nil {
		t.Fatal("DefaultLogger must not be nil")
	}
}

func TestDiscardingLoggerExists(t *testing.T) {
	if DiscardingLogger == nil {
		t.Fatal("DiscardingLogger must not be nil")
	}
}

// --- Type shape checks ---

func TestLineFields(t *testing.T) {
	now := time.Now()
	l := Line{
		Text:     "hello",
		Num:      1,
		SeekInfo: SeekInfo{Offset: 0, Whence: io.SeekStart},
		Time:     now,
		Err:      nil,
	}
	if l.Text != "hello" {
		t.Error("Line.Text mismatch")
	}
	if l.Num != 1 {
		t.Error("Line.Num mismatch")
	}
	if !l.Time.Equal(now) {
		t.Error("Line.Time mismatch")
	}
	if l.Err != nil {
		t.Error("Line.Err should be nil")
	}
}

func TestSeekInfoFields(t *testing.T) {
	si := SeekInfo{Offset: 42, Whence: io.SeekEnd}
	if si.Offset != 42 {
		t.Error("SeekInfo.Offset mismatch")
	}
	if si.Whence != io.SeekEnd {
		t.Error("SeekInfo.Whence mismatch")
	}
}

func TestConfigFields(t *testing.T) {
	rl := ratelimiter.NewLeakyBucket(10, time.Second)
	cfg := Config{
		Location:      &SeekInfo{Offset: 0, Whence: io.SeekStart},
		ReOpen:        true,
		MustExist:     true,
		Poll:          true,
		Pipe:          true,
		Follow:        true,
		MaxLineSize:   1024,
		CompleteLines: true,
		RateLimiter:   rl,
		Logger:        DiscardingLogger,
	}
	if cfg.Location == nil {
		t.Error("Config.Location mismatch")
	}
	if !cfg.ReOpen {
		t.Error("Config.ReOpen mismatch")
	}
	if !cfg.MustExist {
		t.Error("Config.MustExist mismatch")
	}
	if !cfg.Poll {
		t.Error("Config.Poll mismatch")
	}
	if !cfg.Pipe {
		t.Error("Config.Pipe mismatch")
	}
	if !cfg.Follow {
		t.Error("Config.Follow mismatch")
	}
	if !cfg.CompleteLines {
		t.Error("Config.CompleteLines mismatch")
	}
	if cfg.MaxLineSize != 1024 {
		t.Error("Config.MaxLineSize mismatch")
	}
	if cfg.RateLimiter == nil {
		t.Error("Config.RateLimiter mismatch")
	}
	if cfg.Logger == nil {
		t.Error("Config.Logger mismatch")
	}
}

func TestTailStructFields(t *testing.T) {
	// Verify the Tail struct exposes the expected public fields.
	// We don't start a real tail; just verify the fields compile.
	var tl Tail
	t.Logf("Tail.Filename type: %T, Lines type: %T", tl.Filename, tl.Lines)
}

// --- Deprecated but stable API ---

func TestNewLineDeprecatedButPresent(t *testing.T) {
	l := NewLine("test", 1)
	if l == nil {
		t.Fatal("NewLine must return non-nil")
	}
	if l.Text != "test" {
		t.Errorf("NewLine().Text = %q, want %q", l.Text, "test")
	}
	if l.Num != 1 {
		t.Errorf("NewLine().Num = %d, want 1", l.Num)
	}
	if l.Time.IsZero() {
		t.Error("NewLine().Time should not be zero")
	}
}

// --- Constructor validation ---

func TestTailFileRejectsReOpenWithoutFollow(t *testing.T) {
	_, err := TailFile("nonexistent", Config{ReOpen: true, Follow: false})
	if err == nil {
		t.Fatal("TailFile should reject ReOpen without Follow")
	}
}

func TestTailFileMustExistFailsForMissingFile(t *testing.T) {
	_, err := TailFile("/no/such/file/ever", Config{MustExist: true})
	if err == nil {
		t.Fatal("TailFile should fail when MustExist=true and file missing")
	}
}

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "tail-api-test-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestTailFileReturnsWorkingTail(t *testing.T) {
	name := createTempFile(t, "line1\nline2\n")

	tl, err := TailFile(name, Config{Follow: false, MustExist: true})
	if err != nil {
		t.Fatalf("TailFile failed: %v", err)
	}
	defer tl.Cleanup()

	var lines []string
	for line := range tl.Lines {
		lines = append(lines, line.Text)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

// --- Method existence checks ---

func TestTailMethodsExist(t *testing.T) {
	name := createTempFile(t, "hello\n")

	tl, err := TailFile(name, Config{Follow: false, MustExist: true})
	if err != nil {
		t.Fatalf("TailFile failed: %v", err)
	}

	// Drain lines so tail finishes
	for range tl.Lines { //nolint:revive // intentionally draining channel
	}

	// Wait for the goroutine to fully complete before accessing Tail fields.
	// This avoids a race between closeFile() and Tell().
	if err := tl.Wait(); err != nil {
		t.Logf("Wait returned: %v", err)
	}

	// Tell — after Wait(), the goroutine is done; calling Tell is safe
	if _, err := tl.Tell(); err != nil {
		t.Logf("Tell returned: %v", err)
	}

	// Cleanup
	tl.Cleanup()

	// Verify method signatures compile (Stop is already exercised via Wait above)
	_ = tl.Stop
	_ = tl.StopAtEOF
}

func TestStopAtEOFMethodExists(t *testing.T) {
	name := createTempFile(t, "data\n")

	tl, err := TailFile(name, Config{Follow: true, MustExist: true})
	if err != nil {
		t.Fatalf("TailFile failed: %v", err)
	}
	defer tl.Cleanup()

	// StopAtEOF should exist and not panic
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := tl.StopAtEOF(); err != nil {
			t.Logf("StopAtEOF returned: %v", err)
		}
	}()

	for range tl.Lines { //nolint:revive // intentionally draining channel
	}
}
