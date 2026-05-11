package metrics

import "time"

// DailyBucket は 1 日分の集計結果。
type DailyBucket struct {
	Date  time.Time
	Stats Stats
}

// Daily は invs を日次バケットに分割して返す。
// now から days 日前まで（今日含む）の連続したバケットを生成し、
// データがない日は Stats がゼロ値（Compliance/Coverage は -1）のバケットで埋める。
func Daily(invs []Invocation, days int, now time.Time) []DailyBucket {
	today := truncateDay(now)
	buckets := make([]DailyBucket, days)
	for i := range buckets {
		d := today.AddDate(0, 0, -(days-1-i))
		buckets[i] = DailyBucket{Date: d, Stats: Stats{Compliance: -1, Coverage: -1}}
	}

	groups := map[time.Time][]Invocation{}
	for _, inv := range invs {
		d := truncateDay(inv.StartedAt)
		groups[d] = append(groups[d], inv)
	}

	for i, b := range buckets {
		if group, ok := groups[b.Date]; ok {
			buckets[i].Stats = Summarize(group)
		}
	}
	return buckets
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
