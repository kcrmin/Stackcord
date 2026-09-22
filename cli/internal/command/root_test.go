package command

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestHarnessAndLegacyAliasPreserveReadOnlyDomainResult(t *testing.T) {
	for _, prefix := range [][]string{nil, {"harness"}} {
		var out bytes.Buffer
		cmd := New("test", &out, &bytes.Buffer{})
		args := append(append([]string{}, prefix...), "status", "--root", t.TempDir(), "--json")
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(out.Bytes(), &value); err != nil {
			t.Fatal(err, out.String())
		}
		if value["overall"] != "blocked" {
			t.Fatal(value)
		}
	}
}
