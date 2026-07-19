package output_test

import (
	"regexp"
	"sort"
	"testing"

	"github.com/kcrmin/Stackcord/cli/internal/output"
	"github.com/stretchr/testify/require"
)

func TestLocaleCatalogsHaveSemanticParity(t *testing.T) {
	english, err := output.LoadCatalog("en")
	require.NoError(t, err)
	korean, err := output.LoadCatalog("ko")
	require.NoError(t, err)
	require.Equal(t, sortedKeys(english), sortedKeys(korean))
	placeholder := regexp.MustCompile(`\{[a-z_]+\}`)
	for key, left := range english {
		right := korean[key]
		require.NotEmpty(t, right.Text, key)
		require.Equal(t, left.Severity, right.Severity, key)
		require.Equal(t, left.DocSection, right.DocSection, key)
		require.ElementsMatch(t, placeholder.FindAllString(left.Text, -1), placeholder.FindAllString(right.Text, -1), key)
	}
}

func TestLocalizeFallsBackToEnglish(t *testing.T) {
	message := output.Localize("fr", "context.audit.complete", map[string]string{"count": "4"})
	require.Equal(t, "Context audit completed with 4 documents.", message)
}

func sortedKeys(catalog map[string]output.Message) []string {
	result := make([]string, 0, len(catalog))
	for key := range catalog {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
