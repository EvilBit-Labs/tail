package ratelimiter

import (
	"testing"
	"time"
)

func TestPour(t *testing.T) {
	bucket := NewLeakyBucket(60, time.Second)
	bucket.Lastupdate = time.Unix(0, 0)

	bucket.Now = func() time.Time { return time.Unix(1, 0) }

	if bucket.Pour(61) {
		t.Error("Expected false")
	}

	if !bucket.Pour(10) {
		t.Error("Expected true")
	}

	if !bucket.Pour(49) {
		t.Error("Expected true")
	}

	if bucket.Pour(2) {
		t.Error("Expected false")
	}

	bucket.Now = func() time.Time { return time.Unix(61, 0) }
	if !bucket.Pour(60) {
		t.Error("Expected true")
	}

	if bucket.Pour(1) {
		t.Error("Expected false")
	}

	bucket.Now = func() time.Time { return time.Unix(70, 0) }

	if !bucket.Pour(1) {
		t.Error("Expected true")
	}
}

func TestTimeSinceLastUpdate(t *testing.T) {
	bucket := NewLeakyBucket(60, time.Second)
	bucket.Now = func() time.Time { return time.Unix(1, 0) }
	bucket.Pour(1)
	bucket.Now = func() time.Time { return time.Unix(2, 0) }

	sinceLast := bucket.TimeSinceLastUpdate()
	if sinceLast != time.Second*1 {
		t.Errorf("Expected time since last update to be less than 1 second, got %d", sinceLast)
	}
}

func TestTimeToDrain(t *testing.T) {
	bucket := NewLeakyBucket(60, time.Second)
	bucket.Now = func() time.Time { return time.Unix(1, 0) }
	bucket.Pour(10)

	if bucket.TimeToDrain() != time.Second*10 {
		t.Error("Time to drain should be 10 seconds")
	}

	bucket.Now = func() time.Time { return time.Unix(2, 0) }

	if bucket.TimeToDrain() != time.Second*9 {
		t.Error("Time to drain should be 9 seconds")
	}
}

func TestDrainedAt(t *testing.T) {
	bucket := NewLeakyBucket(60, time.Second)
	bucket.Now = func() time.Time { return time.Unix(100, 0) }
	bucket.Pour(10)

	drainedAt := bucket.DrainedAt()
	// Fill is 10, leak interval is 1s, so drained 10s after Lastupdate
	expected := bucket.Lastupdate.Add(10 * time.Second)
	if !drainedAt.Equal(expected) {
		t.Errorf("DrainedAt() = %v, want %v", drainedAt, expected)
	}
}

func TestDrainedAtEmpty(t *testing.T) {
	bucket := NewLeakyBucket(60, time.Second)
	bucket.Now = func() time.Time { return time.Unix(100, 0) }

	drainedAt := bucket.DrainedAt()
	if !drainedAt.Equal(bucket.Lastupdate) {
		t.Errorf("DrainedAt() on empty bucket = %v, want %v", drainedAt, bucket.Lastupdate)
	}
}

func TestSerialiseDeSerialise(t *testing.T) {
	bucket := NewLeakyBucket(60, time.Second)
	bucket.Now = func() time.Time { return time.Unix(1, 0) }
	bucket.Pour(10)

	ser := bucket.Serialise()

	if ser.Size != bucket.Size {
		t.Errorf("Serialise().Size = %d, want %d", ser.Size, bucket.Size)
	}
	if ser.Fill != bucket.Fill {
		t.Errorf("Serialise().Fill = %f, want %f", ser.Fill, bucket.Fill)
	}
	if ser.LeakInterval != bucket.LeakInterval {
		t.Errorf("Serialise().LeakInterval = %v, want %v", ser.LeakInterval, bucket.LeakInterval)
	}
	if !ser.Lastupdate.Equal(bucket.Lastupdate) {
		t.Errorf("Serialise().Lastupdate = %v, want %v", ser.Lastupdate, bucket.Lastupdate)
	}

	restored := ser.DeSerialise()

	if restored.Size != bucket.Size {
		t.Errorf("DeSerialise().Size = %d, want %d", restored.Size, bucket.Size)
	}
	if restored.Fill != bucket.Fill {
		t.Errorf("DeSerialise().Fill = %f, want %f", restored.Fill, bucket.Fill)
	}
	if restored.LeakInterval != bucket.LeakInterval {
		t.Errorf("DeSerialise().LeakInterval = %v, want %v", restored.LeakInterval, bucket.LeakInterval)
	}
	if !restored.Lastupdate.Equal(bucket.Lastupdate) {
		t.Errorf("DeSerialise().Lastupdate = %v, want %v", restored.Lastupdate, bucket.Lastupdate)
	}
	if restored.Now == nil {
		t.Error("DeSerialise().Now should not be nil")
	}
}

func TestSerialiseRoundTripPreservesBehavior(t *testing.T) {
	bucket := NewLeakyBucket(60, time.Second)
	bucket.Now = func() time.Time { return time.Unix(1, 0) }
	bucket.Pour(10)

	restored := bucket.Serialise().DeSerialise()
	restored.Now = func() time.Time { return time.Unix(2, 0) }

	// After 1 second, 1 unit should have leaked, so we can pour 51 more (60 - 10 + 1)
	if !restored.Pour(51) {
		t.Error("expected Pour(51) to succeed after 1s leak on restored bucket")
	}
	if restored.Pour(1) {
		t.Error("expected Pour(1) to fail on full restored bucket")
	}
}
