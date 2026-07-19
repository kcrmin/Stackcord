package ui_test

import (
	"sort"
	"testing"

	contextpkg "github.com/kcrmin/Stackcord/cli/internal/context"
	uiimport "github.com/kcrmin/Stackcord/cli/internal/ui"
	"github.com/stretchr/testify/require"
)

func TestCoverageReportIsDeterministicallySorted(t *testing.T) {
	snapshot := contextpkg.Snapshot{Index: map[string]contextpkg.IndexEntry{
		"journey.zeta":  {ID: "journey.zeta", Kind: "journey"},
		"journey.alpha": {ID: "journey.alpha", Kind: "journey"},
		"journey.mid":   {ID: "journey.mid", Kind: "journey"},
	}, Impact: map[string][]string{}}
	for range 100 {
		report := uiimport.CheckCoverage(snapshot)
		require.True(t, sort.StringsAreSorted(report.MissingJourneys), report.MissingJourneys)
	}
}
