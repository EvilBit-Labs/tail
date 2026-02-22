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
