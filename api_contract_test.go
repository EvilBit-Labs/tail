package tail

import (
	"errors"
	"io"
	"log"
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
	var target error = ErrStop
	if !errors.Is(target, ErrStop) {
		t.Fatal("ErrStop must satisfy errors.Is")
	}
}

func TestDefaultLoggerExists(t *testing.T) {
	if DefaultLogger == nil {
		t.Fatal("DefaultLogger must not be nil")
	}
	// Verify it's a *log.Logger
	var _ *log.Logger = DefaultLogger
}

func TestDiscardingLoggerExists(t *testing.T) {
	if DiscardingLogger == nil {
		t.Fatal("DiscardingLogger must not be nil")
	}
	var _ *log.Logger = DiscardingLogger
}

// --- Type shape checks ---

func TestLineFields(t *testing.T) {
	l := Line{
		Text:     "hello",
		Num:      1,
		SeekInfo: SeekInfo{Offset: 0, Whence: io.SeekStart},
		Time:     time.Now(),
		Err:      nil,
	}
	if l.Text != "hello" {
		t.Error("Line.Text mismatch")
	}
	if l.Num != 1 {
		t.Error("Line.Num mismatch")
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
	if !cfg.ReOpen {
		t.Error("Config.ReOpen mismatch")
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
}

func TestTailStructFields(t *testing.T) {
	// Verify the Tail struct exposes the expected public fields.
	// We don't start a real tail; just verify the fields compile.
	var tl Tail
	_ = tl.Filename
	_ = tl.Lines
	_ = tl.Config
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

func TestTailFileReturnsWorkingTail(t *testing.T) {
	// Create a temp file to tail
	f, err := os.CreateTemp("", "api-contract-test-")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.WriteString("line1\nline2\n")
	f.Close()
	defer os.Remove(name)

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
	// Create a temp file so we get a valid *Tail
	f, err := os.CreateTemp("", "api-methods-test-")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.WriteString("hello\n")
	f.Close()
	defer os.Remove(name)

	tl, err := TailFile(name, Config{Follow: false, MustExist: true})
	if err != nil {
		t.Fatalf("TailFile failed: %v", err)
	}

	// Drain lines so tail finishes
	for range tl.Lines {
	}

	// Wait for the goroutine to fully complete before accessing Tail fields.
	// This avoids a race between closeFile() and Tell().
	_ = tl.Wait()

	// Tell — after Wait(), the goroutine is done; calling Tell is safe
	_, tellErr := tl.Tell()
	_ = tellErr

	// Cleanup
	tl.Cleanup()

	// Verify method signatures compile (Stop is already exercised via Wait above)
	var _ func() error = tl.Stop
	var _ func() error = tl.StopAtEOF
}

func TestStopAtEOFMethodExists(t *testing.T) {
	f, err := os.CreateTemp("", "api-stopeof-test-")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.WriteString("data\n")
	f.Close()
	defer os.Remove(name)

	tl, err := TailFile(name, Config{Follow: true, MustExist: true})
	if err != nil {
		t.Fatalf("TailFile failed: %v", err)
	}
	defer tl.Cleanup()

	// StopAtEOF should exist and not panic
	go func() {
		time.Sleep(100 * time.Millisecond)
		tl.StopAtEOF()
	}()

	for range tl.Lines {
	}
}
