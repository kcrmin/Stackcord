package governance

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type lifecycleTransport func(context.Context, ...string) ([]byte, error)

func (f lifecycleTransport) Run(ctx context.Context, args ...string) ([]byte, error) {
	return f(ctx, args...)
}
func localCandidate(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	localGit(t, root, "init")
	localGit(t, root, "remote", "add", "origin", "https://github.com/owner/repo.git")
	localGit(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "test: candidate")
	return root, localGit(t, root, "rev-parse", "HEAD")
}
func localGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	b, e := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if e != nil {
		t.Fatal(string(b), e)
	}
	return strings.TrimSpace(string(b))
}

func TestLifecycleRejectsDifferentReviewedCandidate(t *testing.T) {
	root, head := localCandidate(t)
	f := &liveTransport{policy: livePolicy("strong"), files: `[[{"filename":"contracts/api.yaml"}]]`, reviews: `[[{"id":1,"user":{"login":"admin"},"state":"APPROVED","commit_id":"` + liveHead + `"}]]`}
	tr := lifecycleTransport(func(ctx context.Context, args ...string) ([]byte, error) {
		if args[3] == "repos/owner/repo/pulls?state=open&per_page=100" {
			return json.Marshal([][]map[string]any{{{"number": 1, "head": map[string]string{"sha": head}}}})
		}
		return f.Run(ctx, args...)
	})
	r := checkGitHubWithTransport(context.Background(), root, liveNow, tr)
	if r.Status == Approved {
		t.Fatal("new remote head approved for old local candidate", r)
	}
}

func TestLifecycleRejectsLocalMutationDuringReview(t *testing.T) {
	for _, mutation := range []string{"dirty", "head"} {
		t.Run(mutation, func(t *testing.T) {
			root, head := localCandidate(t)
			f := &liveTransport{policy: livePolicy("strong"), files: `[[{"filename":"contracts/api.yaml"}]]`, reviews: `[[{"id":1,"user":{"login":"admin"},"state":"APPROVED","commit_id":"` + liveHead + `"}]]`}
			tr := lifecycleTransport(func(ctx context.Context, args ...string) ([]byte, error) {
				switch args[3] {
				case "repos/owner/repo/pulls?state=open&per_page=100":
					return json.Marshal([][]map[string]any{{{"number": 1, "head": map[string]string{"sha": head}}}})
				case "repos/owner/repo/pulls/1/reviews?per_page=100":
					if mutation == "head" {
						localGit(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "test: concurrent commit")
					} else {
						if e := os.MkdirAll(filepath.Join(root, "contracts"), 0755); e != nil {
							t.Fatal(e)
						}
						if e := os.WriteFile(filepath.Join(root, "contracts", "api.yaml"), []byte("changed"), 0644); e != nil {
							t.Fatal(e)
						}
					}
				}
				b, e := f.Run(ctx, args...)
				return []byte(strings.ReplaceAll(string(b), liveHead, head)), e
			})
			r := checkGitHubWithTransport(context.Background(), root, liveNow, tr)
			if r.Status == Approved {
				t.Fatal("local mutation was approved", r)
			}
		})
	}
}

func TestDisabledUnselectedGitHubProjectRemainsOffline(t *testing.T) {
	root, _ := localCandidate(t)
	if useLive(context.Background(), root, Policy{SchemaVersion: 1, Enabled: false}) {
		t.Fatal("ordinary disabled GitHub project requires live authentication")
	}
}

func TestLifecycleDefaultCandidateRechecksTarget(t *testing.T) {
	root, head := localCandidate(t)
	f := &liveTransport{policy: livePolicy("strong")}
	calls := 0
	tr := lifecycleTransport(func(ctx context.Context, args ...string) ([]byte, error) {
		if args[3] == "repos/owner/repo/commits/main" {
			calls++
			sha := head
			if calls > 1 {
				sha = liveMoved
			}
			return json.Marshal(map[string]string{"sha": sha})
		}
		argsCopy := append([]string(nil), args...)
		argsCopy[3] = strings.ReplaceAll(argsCopy[3], head, liveBase)
		return f.Run(ctx, argsCopy...)
	})
	r := checkGitHubWithTransport(context.Background(), root, liveNow, tr)
	if r.Status == Approved {
		t.Fatal("moved default candidate approved", r)
	}
}
