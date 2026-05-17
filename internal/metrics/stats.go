package metrics

// Stats holds aggregated results for a time period.
// Compliance / Coverage of -1 means the denominator was zero (N/A).
type Stats struct {
	Compliance  float64
	Coverage    float64
	Executed    int
	Denied      int
	Cancelled   int
	Unknown     int
	Interrupted int
}

// Summarize computes Stats from a slice of Invocations.
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
