package channel

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCodePolicyRejectsAmbiguousRelativeRemote(t *testing.T) {
	exe, _ := os.Executable()
	if err := validateCodePolicy(&CodePolicy{Repository: "project", Remote: "origin", VerifyArgv: []string{exe}, VerifyTimeoutSeconds: 10}); err == nil {
		t.Fatal("relative code remote changes meaning inside a job clone")
	}
}

func codeRequest(id, scope string) Event {
	return Event{ID: id, Type: "request", To: "bob", Scope: []string{scope}, Code: &CodeRequest{Repository: "project", BaseCommit: strings.Repeat("a", 40), Resources: []string{"contract.api"}}}
}

func TestCodeClaimsRequireDependencies(t *testing.T) {
	first := codeRequest("first", "backend")
	second := codeRequest("second", "frontend")
	if err := validateCodeEvent(second, []Event{first}); err == nil {
		t.Fatal("semantic overlap accepted without prerequisite")
	}
	second.Dependencies = []string{"first"}
	if err := validateCodeEvent(second, []Event{first}); err != nil {
		t.Fatal(err)
	}
	second.Code.Resources = nil
	second.Dependencies = nil
	second.Scope = []string{"BACKEND/api.go"}
	if err := validateCodeEvent(second, []Event{first}); err == nil {
		t.Fatal("case-insensitive path overlap accepted")
	}
	second.Scope = []string{"frontend"}
	if err := validateCodeEvent(second, []Event{first}); err != nil {
		t.Fatal(err)
	}
	second.Dependencies = []string{"first"}
	second.Code.BaseCommit = strings.Repeat("b", 40)
	if err := validateCodeEvent(second, []Event{first}); err == nil {
		t.Fatal("different prerequisite base accepted")
	}
}

func TestCodeClaimsDifferentBasesAndFailedRetry(t *testing.T) {
	first := codeRequest("first", "src")
	second := codeRequest("second", "src")
	second.Code.BaseCommit = strings.Repeat("b", 40)
	if err := validateCodeEvent(second, []Event{first}); err == nil {
		t.Fatal("active conflicting ownership accepted on another base")
	}
	second.Code.BaseCommit = first.Code.BaseCommit
	if err := validateCodeEvent(second, []Event{first, {Type: "result", RequestID: first.ID, Status: "failed"}}); err != nil {
		t.Fatalf("failed isolated work permanently reserved scope: %v", err)
	}
}

func TestCodeScopeAndArtifactsFailClosed(t *testing.T) {
	for _, scope := range []string{"", "../outside", "/root", `.\file`, "a/../b", ".git", ".harness/local", "C:/file"} {
		r := codeRequest("request", scope)
		if err := validateCodeEvent(r, nil); err == nil {
			t.Fatalf("accepted scope %q", scope)
		}
	}
	r := codeRequest("request", "src")
	result := Event{Type: "result", RequestID: r.ID, Status: "success"}
	if err := validateCodeEvent(result, []Event{r}); err == nil {
		t.Fatal("self-reported code success accepted")
	}
	result.Artifact = &CodeArtifact{Repository: "project", BaseCommit: r.Code.BaseCommit, Commit: strings.Repeat("b", 40), Tree: strings.Repeat("c", 40), Verification: strings.Repeat("d", 64)}
	if err := validateCodeEvent(result, []Event{r}); err != nil {
		t.Fatal(err)
	}
	result.Artifact.BaseCommit = strings.Repeat("e", 40)
	if err := validateCodeEvent(result, []Event{r}); err == nil {
		t.Fatal("artifact from another base accepted")
	}
}

func TestManualCodeSuccessCannotBypassVerification(t *testing.T) {
	a, b := fixture(t)
	input := RequestInput{To: "bob", Kind: "implementation", Title: "checked code", Scope: []string{"src"}, Code: codeRequest("unused", "src").Code}
	r, err := a.Send(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Respond(context.Background(), ResultInput{RequestID: r.ID, Status: "success"}); err == nil {
		t.Fatal("manual success bypassed code checks")
	}
}
