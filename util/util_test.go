package util

import (
	"testing"
)

func TestPartitionString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		chunkSize int
		want      []string
	}{
		{
			name:      "exact multiple",
			input:     "abcdef",
			chunkSize: 3,
			want:      []string{"abc", "def"},
		},
		{
			name:      "remainder",
			input:     "abcdefg",
			chunkSize: 3,
			want:      []string{"abc", "def", "g"},
		},
		{
			name:      "single chunk",
			input:     "ab",
			chunkSize: 5,
			want:      []string{"ab"},
		},
		{
			name:      "chunk size equals length",
			input:     "abc",
			chunkSize: 3,
			want:      []string{"abc"},
		},
		{
			name:      "single char chunks",
			input:     "abc",
			chunkSize: 1,
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "empty string",
			input:     "",
			chunkSize: 3,
			want:      []string{""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PartitionString(tt.input, tt.chunkSize)
			if len(got) != len(tt.want) {
				t.Fatalf("PartitionString(%q, %d) returned %d chunks, want %d",
					tt.input, tt.chunkSize, len(got), len(tt.want))
			}
			for i, chunk := range got {
				if chunk != tt.want[i] {
					t.Errorf("chunk[%d] = %q, want %q", i, chunk, tt.want[i])
				}
			}
		})
	}
}

func TestPartitionStringPanicsOnZeroChunkSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for chunkSize <= 0")
		}
	}()
	PartitionString("abc", 0)
}

func TestPartitionStringPanicsOnNegativeChunkSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for negative chunkSize")
		}
	}()
	PartitionString("abc", -1)
}

// --- API contract tests ---

func TestLOGGERExists(t *testing.T) {
	if LOGGER == nil {
		t.Fatal("LOGGER must not be nil")
	}
}

func TestLoggerTypeExists(t *testing.T) {
	// Verify Logger struct is exported and embeds *log.Logger
	var l Logger
	_ = l.Logger // embedded *log.Logger field
}

func TestFatalFunctionExists(t *testing.T) {
	// We can't call Fatal (it calls os.Exit), but we verify it compiles
	var fn func(string, ...interface{}) = Fatal
	_ = fn
}
