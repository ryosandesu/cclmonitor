package metrics

// Stats は期間内の集計結果を保持する。
// Compliance / Coverage が -1 のときは分母ゼロ（N/A）を意味する。
type Stats struct {
	Compliance  float64
	Coverage    float64
	Executed    int
	Denied      int
	Cancelled   int
	Unknown     int
	Interrupted int
}

// Summarize は Invocation のスライスから Stats を算出する。
func Summarize(invs []Invocation) Stats {
	s := Stats{}
	for _, inv := range invs {
		switch inv.Outcome {
		case "executed":
			s.Executed++
		case "denied":
			s.Denied++
		case "cancelled":
			s.Cancelled++
		case "unknown":
			s.Unknown++
		case "interrupted":
			s.Interrupted++
		}
	}

	compDenom := s.Executed + s.Denied + s.Cancelled
	if compDenom == 0 {
		s.Compliance = -1
	} else {
		s.Compliance = float64(s.Executed) / float64(compDenom)
	}

	covDenom := s.Executed + s.Denied + s.Unknown
	if covDenom == 0 {
		s.Coverage = -1
	} else {
		s.Coverage = float64(s.Executed+s.Denied) / float64(covDenom)
	}

	return s
}
