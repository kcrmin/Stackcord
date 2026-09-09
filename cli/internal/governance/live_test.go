package governance

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	github "github.com/kcrmin/Stackcord/cli/internal/github"
)

const liveBase = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const liveHead = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const liveMoved = "cccccccccccccccccccccccccccccccccccccccc"

var liveNow = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

type liveTransport struct {
	policy, files, reviews                     string
	missing                                    bool
	finalHead, finalBase, finalRef, finalState string
	heads, prs, policyReads                    int
	repos                                      int
	finalDefault                               string
}

func (f *liveTransport) Run(_ context.Context, args ...string) ([]byte, error) {
	endpoint := args[3]
	switch endpoint {
	case "repos/owner/repo":
		f.repos++
		if f.repos > 1 && f.finalDefault != "" {
			return []byte(fmt.Sprintf(`{"full_name":"owner/repo","default_branch":%q}`, f.finalDefault)), nil
		}
		return []byte(`{"full_name":"owner/repo","default_branch":"main"}`), nil
	case "repos/owner/repo/commits/main":
		f.heads++
		sha := liveBase
		if f.heads > 1 && f.finalBase != "" {
			sha = f.finalBase
		}
		return json.Marshal(map[string]string{"sha": sha})
	case "repos/owner/repo/contents/.harness/governance.yaml?ref=" + liveBase:
		f.policyReads++
		if f.missing {
			return nil, github.ErrNotFound
		}
		return json.Marshal(map[string]string{"encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(f.policy))})
	case "repos/owner/repo/pulls/1":
		f.prs++
		head, ref, state := liveHead, "main", "open"
		if f.prs > 1 {
			if f.finalHead != "" {
				head = f.finalHead
			}
			if f.finalRef != "" {
				ref = f.finalRef
			}
			if f.finalState != "" {
				state = f.finalState
			}
		}
		return []byte(fmt.Sprintf(`{"number":1,"state":%q,"head":{"sha":%q},"base":{"ref":%q,"repo":{"full_name":"owner/repo"}},"user":{"login":"author"}}`, state, head, ref)), nil
	case "repos/owner/repo/pulls/1/files?per_page=100":
		return []byte(f.files), nil
	case "repos/owner/repo/pulls/1/reviews?per_page=100":
		return []byte(f.reviews), nil
	default:
		return nil, fmt.Errorf("unexpected endpoint %s", endpoint)
	}
}

func TestLiveDefaultBranchSettingMovementFailsClosed(t *testing.T) {
	f := &liveTransport{policy: livePolicy("weak"), files: `[[{"filename":"README.md"}]]`, finalDefault: "release"}
	r, e := runLive(t, f)
	if e == nil || r.Approved {
		t.Fatal("changed repository default branch accepted", r, e)
	}
}

func TestLiveUsesTrustedProtectedKinds(t *testing.T) {
	for _, tc := range []struct {
		file, protected string
		wantApproved    bool
		wantKinds       string
	}{
		{"contracts/api.yaml", "product", true, ""},
		{"specs/product.md", "contract", true, ""},
		{"specs/product.md", "product", false, "product"},
		{"specs/product.md", "policy, business", false, "business,policy"},
		{".harness/governance.yaml", "contract", false, "policy"},
	} {
		p := strings.Replace(livePolicy("strong"), "product, policy, business, contract", tc.protected, 1)
		f := &liveTransport{policy: p, files: fmt.Sprintf(`[[{"filename":%q}]]`, tc.file), reviews: `[[]]`}
		r, e := runLive(t, f)
		if e != nil || r.Approved != tc.wantApproved || strings.Join(r.Kinds, ",") != tc.wantKinds {
			t.Fatalf("%s [%s]: %+v %v", tc.file, tc.protected, r, e)
		}
	}
}

func TestLiveSelectionSurvivesCommittedCandidateDowngrade(t *testing.T) {
	for _, remote := range []string{"https://github.com/owner/repo.git", "git@github.com:owner/repo.git", "ssh://git@github.com/owner/repo.git"} {
		root := t.TempDir()
		git := func(args ...string) {
			t.Helper()
			cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
			if out, e := cmd.CombinedOutput(); e != nil {
				t.Fatalf("git: %s %v", out, e)
			}
		}
		git("init")
		git("remote", "add", "origin", remote)
		if e := os.MkdirAll(filepath.Join(root, ".harness"), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(root, ".harness", "governance.yaml"), []byte(livePolicy("strong")), 0644); e != nil {
			t.Fatal(e)
		}
		git("add", ".")
		git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "test: approved enrollment")
		git("update-ref", "refs/remotes/origin/main", "HEAD")
		git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
		candidate := strings.Replace(livePolicy("weak"), "schema_version: 2", "schema_version: 1", 1)
		candidate = strings.Replace(candidate, "mode: weak\n", "", 1)
		if e := os.WriteFile(filepath.Join(root, ".harness", "governance.yaml"), []byte(candidate), 0644); e != nil {
			t.Fatal(e)
		}
		git("add", ".")
		git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "test: candidate downgrade")
		p, e := LoadPolicy(root)
		if e != nil {
			t.Fatal(e)
		}
		if !useLive(context.Background(), root, p) {
			t.Fatal("candidate downgrade selected legacy observations")
		}
	}
}
func livePolicy(mode string) string {
	return fmt.Sprintf(`schema_version: 2
enabled: %t
provider: github
repository: owner/repo
mode: %s
product_authorities: ["user:admin"]
protected_kinds: [product, policy, business, contract]
approval:
  minimum: 1
  authority_self_approval: false
`, mode != "weak", mode)
}
func runLive(t *testing.T, f *liveTransport) (LiveReview, error) {
	t.Helper()
	c, e := github.New("owner/repo", f)
	if e != nil {
		t.Fatal(e)
	}
	return ReviewLive(context.Background(), c, 1, liveNow)
}
func TestLiveWeakAndUnprotectedStillRevalidate(t *testing.T) {
	for _, mode := range []string{"weak", "strong"} {
		for _, move := range []string{"head", "base", "target", "closed"} {
			t.Run(mode+"/"+move, func(t *testing.T) {
				f := &liveTransport{policy: livePolicy(mode), files: `[[{"filename":"README.md"}]]`}
				switch move {
				case "head":
					f.finalHead = liveMoved
				case "base":
					f.finalBase = liveMoved
				case "target":
					f.finalRef = "other"
				case "closed":
					f.finalState = "closed"
				}
				r, e := runLive(t, f)
				if e == nil || r.Approved {
					t.Fatalf("changed PR accepted: %+v %v", r, e)
				}
			})
		}
	}
}
func TestLiveStrongDoesNotReadCandidateWeakPolicy(t *testing.T) {
	f := &liveTransport{policy: livePolicy("strong"), files: `[[{"filename":".harness/governance.yaml","patch":"-mode: strong\n+mode: weak\n+product_authorities: [user:attacker]"}]]`, reviews: `[[{"id":1,"user":{"login":"attacker"},"state":"APPROVED","commit_id":"` + liveHead + `"}]]`}
	r, e := runLive(t, f)
	if e != nil || r.Approved || !r.PolicyChange || r.Policy.Mode != "strong" || f.policyReads != 1 {
		t.Fatalf("trusted policy bypass: %+v %v", r, e)
	}
}
func TestLiveLatestDismissalAndStaleApprovalCannotAuthorize(t *testing.T) {
	for _, reviews := range []string{`[[{"id":1,"user":{"login":"admin"},"state":"APPROVED","commit_id":"` + liveHead + `"}],[{"id":2,"user":{"login":"ADMIN"},"state":"DISMISSED","commit_id":"` + liveHead + `"}]]`, `[[{"id":1,"user":{"login":"admin"},"state":"APPROVED","commit_id":"` + liveBase + `"}]]`} {
		f := &liveTransport{policy: livePolicy("strong"), files: `[[{"filename":"contracts/api.yaml"}]]`, reviews: reviews}
		r, e := runLive(t, f)
		if e != nil || r.Approved || len(r.Approvers) != 0 {
			t.Fatalf("invalid approval counted: %+v %v", r, e)
		}
	}
}
func TestLiveDelegateScopeExpiryAndGovernanceBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, file, kinds, expiry string
		want                      bool
	}{{"valid", "contracts/api.yaml", "contract", "2026-09-10T12:00:00Z", true}, {"expired", "contracts/api.yaml", "contract", "2026-09-09T12:00:00Z", false}, {"wrong scope", "contracts/api.yaml", "product", "2026-09-10T12:00:00Z", false}, {"governance", ".harness/governance.yaml", "policy", "2026-09-10T12:00:00Z", false}} {
		t.Run(tc.name, func(t *testing.T) {
			policy := livePolicy("medium") + fmt.Sprintf("delegates:\n  - subject: user:temp\n    repository: owner/repo\n    kinds: [%s]\n    expires_at: %s\n", tc.kinds, tc.expiry)
			f := &liveTransport{policy: policy, files: fmt.Sprintf(`[[{"filename":%q}]]`, tc.file), reviews: `[[{"id":1,"user":{"login":"temp"},"state":"APPROVED","commit_id":"` + liveHead + `"}]]`}
			r, e := runLive(t, f)
			if e != nil || r.Approved != tc.want {
				t.Fatalf("delegation: %+v %v", r, e)
			}
		})
	}
}
func TestLiveMissingTrustedPolicyFailsClosed(t *testing.T) {
	f := &liveTransport{policy: livePolicy("weak"), missing: true}
	r, e := runLive(t, f)
	if e == nil || r.Approved || !strings.Contains(e.Error(), "trusted target policy") {
		t.Fatal(r, e)
	}
}
func TestLiveRenameAwayFromGovernanceRequiresAdmin(t *testing.T) {
	f := &liveTransport{policy: livePolicy("strong"), files: `[[{"filename":"backup.yaml","previous_filename":".harness/governance.yaml","status":"renamed"}]]`, reviews: `[[]]`}
	r, e := runLive(t, f)
	if e != nil || r.Approved || !r.PolicyChange {
		t.Fatal(r, e)
	}
}
