package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateReleaseEvidenceBindsEveryMigrationRecordDeterministically(t *testing.T) {
	left := aggregateReleaseEvidence(map[string]string{"evidence.b": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "evidence.a": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	right := aggregateReleaseEvidence(map[string]string{"evidence.a": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "evidence.b": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	oneOnly := aggregateReleaseEvidence(map[string]string{"evidence.a": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})

	require.Equal(t, left, right)
	require.NotEqual(t, left, oneOnly)
	require.True(t, isDigest(left))
}

func TestIntegrationRequirementIsScopedPerWork(t *testing.T) {
	require.True(t, hasReleaseEvidenceKind(map[string]bool{"child-merge": true}, "integration", "child-merge", "root-pointer"))
	require.False(t, hasReleaseEvidenceKind(map[string]bool{"test": true}, "integration", "child-merge", "root-pointer"))
}
