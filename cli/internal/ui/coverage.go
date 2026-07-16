package ui

import contextpkg "fullstack-orchestrator/cli/internal/context"

// CoverageReport records journeys without a linked UI baseline.
type CoverageReport struct {
	MissingJourneys []string `json:"missing_journeys"`
}

// CheckCoverage reports approved journeys lacking a UI reference edge.
func CheckCoverage(snapshot contextpkg.Snapshot) CoverageReport {
	report := CoverageReport{MissingJourneys: []string{}}
	for id, entry := range snapshot.Index {
		if entry.Kind == "journey" && len(snapshot.Impact[id]) == 0 {
			report.MissingJourneys = append(report.MissingJourneys, id)
		}
	}
	return report
}
