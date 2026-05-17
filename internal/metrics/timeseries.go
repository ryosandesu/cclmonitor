package metrics

import (
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

// DailyBucket holds the aggregated stats for one day.
type DailyBucket struct {
	Date  time.Time
	Stats Stats
}

// Daily splits invs into daily buckets and returns them.
// It generates a contiguous sequence of days from (now - days + 1) to today,
// filling days with no data with zero-value Stats (Compliance/Coverage = -1).
func Daily(invs []Invocation, days int, now time.Time) []DailyBucket {
	today := eventlog.TruncateDay(now)
	buckets := make([]DailyBucket, days)
	for i := range buckets {
		d := today.AddDate(0, 0, -(days-1-i))
		buckets[i] = DailyBucket{Date: d, Stats: Stats{Compliance: -1, Coverage: -1}}
	}

	groups := map[time.Time][]Invocation{}
	for _, inv := range invs {
		d := eventlog.TruncateDay(inv.StartedAt)
		groups[d] = append(groups[d], inv)
	}

	for i, b := range buckets {
		if group, ok := groups[b.Date]; ok {
			buckets[i].Stats = Summarize(group)
		}
	}
	return buckets
}
