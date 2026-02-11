package errors

// Severity classifies the operational impact of an error.
// It drives logging levels and alerting decisions.
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

var severityStrings = map[Severity]string{
	SeverityLow:      "low",
	SeverityMedium:   "medium",
	SeverityHigh:     "high",
	SeverityCritical: "critical",
}

func (s Severity) String() string {
	if str, ok := severityStrings[s]; ok {
		return str
	}
	return "unknown"
}
