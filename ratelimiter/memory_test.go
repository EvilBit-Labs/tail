package ratelimiter

import (
	"fmt"
	"testing"
	"time"
)

// Compile-time check: Memory implements Storage.
var _ Storage = (*Memory)(nil)

func mustSetBucket(t *testing.T, m *Memory, key string, bucket LeakyBucket) {
	t.Helper()
	if err := m.SetBucketFor(key, bucket); err != nil {
		t.Fatalf("SetBucketFor(%q) failed: %v", key, err)
	}
}

func TestNewMemory(t *testing.T) {
	m := NewMemory()
	if m == nil {
		t.Fatal("NewMemory() returned nil")
	}
}

func TestMemoryGetBucketForMissing(t *testing.T) {
	m := NewMemory()
	_, err := m.GetBucketFor("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestMemorySetAndGetBucket(t *testing.T) {
	m := NewMemory()
	bucket := *NewLeakyBucket(100, time.Second)

	mustSetBucket(t, m, "test-key", bucket)

	got, err := m.GetBucketFor("test-key")
	if err != nil {
		t.Fatalf("GetBucketFor failed: %v", err)
	}
	if got.Size != bucket.Size {
		t.Errorf("Size = %d, want %d", got.Size, bucket.Size)
	}
	if got.LeakInterval != bucket.LeakInterval {
		t.Errorf("LeakInterval = %v, want %v", got.LeakInterval, bucket.LeakInterval)
	}
}

func TestMemoryOverwritesBucket(t *testing.T) {
	m := NewMemory()

	bucket1 := *NewLeakyBucket(10, time.Second)
	mustSetBucket(t, m, "key", bucket1)

	bucket2 := *NewLeakyBucket(20, time.Minute)
	mustSetBucket(t, m, "key", bucket2)

	got, err := m.GetBucketFor("key")
	if err != nil {
		t.Fatalf("GetBucketFor failed: %v", err)
	}
	if got.Size != 20 {
		t.Errorf("Size = %d, want 20 (overwritten value)", got.Size)
	}
}

func TestMemoryGarbageCollect(t *testing.T) {
	m := NewMemory()
	// Force lastGCCollected to the past so GC is allowed to run
	m.lastGCCollected = time.Now().Add(-2 * GC_PERIOD)

	now := time.Now()

	// Add a drained bucket (DrainedAt in the past)
	drained := *NewLeakyBucket(10, time.Second)
	drained.Now = func() time.Time { return now }
	drained.Lastupdate = now.Add(-time.Hour)
	drained.Fill = 0
	mustSetBucket(t, m, "drained", drained)

	// Add a non-drained bucket (still has fill)
	active := *NewLeakyBucket(10, time.Second)
	active.Now = func() time.Time { return now }
	active.Lastupdate = now
	active.Fill = 5
	mustSetBucket(t, m, "active", active)

	m.GarbageCollect()

	if _, err := m.GetBucketFor("drained"); err == nil {
		t.Error("expected drained bucket to be garbage collected")
	}
	if _, err := m.GetBucketFor("active"); err != nil {
		t.Error("expected active bucket to survive garbage collection")
	}
}

func TestMemoryGarbageCollectRateLimited(t *testing.T) {
	m := NewMemory()
	// lastGCCollected is recent (set by NewMemory to time.Now())

	drained := *NewLeakyBucket(10, time.Second)
	drained.Lastupdate = time.Now().Add(-time.Hour)
	drained.Fill = 0
	mustSetBucket(t, m, "drained", drained)

	// GC should be rate-limited and not run
	m.GarbageCollect()

	// Bucket should still exist because GC was rate-limited
	if _, err := m.GetBucketFor("drained"); err != nil {
		t.Error("expected bucket to survive rate-limited GC")
	}
}

func TestMemorySetBucketTriggersGC(t *testing.T) {
	m := NewMemory()
	m.lastGCCollected = time.Now().Add(-2 * GC_PERIOD)

	now := time.Now()

	// Fill past GC_SIZE with drained buckets
	for i := 0; i < GC_SIZE+1; i++ {
		b := *NewLeakyBucket(10, time.Second)
		b.Now = func() time.Time { return now }
		b.Lastupdate = now.Add(-time.Hour)
		b.Fill = 0
		mustSetBucket(t, m, fmt.Sprintf("key-%d", i), b)
	}

	// This SetBucketFor should trigger GC because store > GC_SIZE
	active := *NewLeakyBucket(10, time.Second)
	active.Now = func() time.Time { return now }
	active.Lastupdate = now
	active.Fill = 5
	mustSetBucket(t, m, "active", active)

	// The active bucket should survive
	if _, err := m.GetBucketFor("active"); err != nil {
		t.Error("expected active bucket to survive after GC trigger")
	}
}

// --- API contract tests ---

func TestGC_SIZEConstant(t *testing.T) {
	if GC_SIZE != 100 {
		t.Errorf("GC_SIZE = %d, want 100", GC_SIZE)
	}
}

func TestGC_PERIODConstant(t *testing.T) {
	if GC_PERIOD != 60*time.Second {
		t.Errorf("GC_PERIOD = %v, want 60s", GC_PERIOD)
	}
}

func TestLeakyBucketSerFields(t *testing.T) {
	now := time.Now()
	ser := LeakyBucketSer{
		Size:         10,
		Fill:         5.0,
		LeakInterval: time.Second,
		Lastupdate:   now,
	}
	if ser.Size != 10 {
		t.Error("LeakyBucketSer.Size mismatch")
	}
	if ser.Fill != 5.0 {
		t.Error("LeakyBucketSer.Fill mismatch")
	}
	if ser.LeakInterval != time.Second {
		t.Error("LeakyBucketSer.LeakInterval mismatch")
	}
	if !ser.Lastupdate.Equal(now) {
		t.Error("LeakyBucketSer.Lastupdate mismatch")
	}
}

func TestNewLeakyBucketContract(t *testing.T) {
	b := NewLeakyBucket(100, 2*time.Second)
	if b.Size != 100 {
		t.Errorf("Size = %d, want 100", b.Size)
	}
	if b.Fill != 0 {
		t.Errorf("Fill = %f, want 0", b.Fill)
	}
	if b.LeakInterval != 2*time.Second {
		t.Errorf("LeakInterval = %v, want 2s", b.LeakInterval)
	}
	if b.Now == nil {
		t.Error("Now function must not be nil")
	}
}
