package metrics

import (
	"testing"
	"time"
)

func TestDaily(t *testing.T) {
	now := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	invs := []Invocation{
		{StartedAt: now, Outcome: "executed"},
		{StartedAt: now, Outcome: "denied"},
		{StartedAt: now.AddDate(0, 0, -1), Outcome: "executed"},
	}

	buckets := Daily(invs, 3, now)

	if len(buckets) != 3 {
		t.Fatalf("want 3 buckets, got %d", len(buckets))
	}

	t.Run("today has 2 invocations", func(t *testing.T) {
		today := buckets[2]
		if today.Stats.Executed != 1 || today.Stats.Denied != 1 {
			t.Errorf("today: want 1 executed 1 denied, got %+v", today.Stats)
		}
	})

	t.Run("yesterday has 1 executed", func(t *testing.T) {
		yesterday := buckets[1]
		if yesterday.Stats.Executed != 1 {
			t.Errorf("yesterday: want 1 executed, got %+v", yesterday.Stats)
		}
	})

	t.Run("2 days ago is empty bucket", func(t *testing.T) {
		twoDaysAgo := buckets[0]
		if twoDaysAgo.Stats.Executed != 0 {
			t.Errorf("2 days ago: want 0 executed, got %+v", twoDaysAgo.Stats)
		}
	})
}
