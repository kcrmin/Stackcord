package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/gitx"
	"github.com/kcrmin/Stackcord/cli/internal/schema"
)

const coordinationFile = "coordination.json"

var (
	ErrCASConflict            = errors.New("Git-local compare-and-swap conflict")
	ErrNoRemote               = errors.New("Git-local remote unavailable")
	ErrAuthentication         = errors.New("Git-local authentication unavailable")
	ErrRevisionMismatch       = errors.New("Git-local expected revision mismatch")
	ErrNonFastHistory         = errors.New("Git-local coordination history is not linear")
	ErrMalformedState         = errors.New("Git-local coordination state is malformed")
	ErrPushRejected           = errors.New("Git-local coordination push was rejected")
	ErrPostconditionMismatch  = errors.New("Git-local coordination postcondition mismatch")
	coordinationIDPattern     = regexp.MustCompile(`^(claim|work)\.[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
	coordinationDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	handoffEvidencePattern    = regexp.MustCompile(`^evidence\.[0-9a-f]{24}$`)
	handoffRefPattern         = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z0-9][a-z0-9-]*)+$`)
	objectIDPattern           = regexp.MustCompile(`^[0-9a-f]{40}(?:[0-9a-f]{24})?$`)
	branchNamePattern         = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)
)

// GitLocalError classifies a coordination failure without exposing a credential-bearing remote URL.
type GitLocalError struct {
	Kind      error
	Operation string
	Cause     error
}

func (err *GitLocalError) Error() string {
	if err.Operation == "" {
		return err.Kind.Error()
	}
	return err.Kind.Error() + " during " + err.Operation
}

func (err *GitLocalError) Unwrap() error { return err.Cause }

func (err *GitLocalError) Is(target error) bool {
	if target == ErrCASConflict {
		return err.Kind == ErrRevisionMismatch
	}
	return target == err.Kind
}

// GitLocalClaim is the normalized semantic scope published through Git-local coordination.
type GitLocalClaim struct {
	ID                    string           `json:"id"`
	WorkID                string           `json:"work_id"`
	DefinitionFingerprint string           `json:"definition_fingerprint"`
	Status                string           `json:"status,omitempty"`
	Owner                 string           `json:"owner"`
	Branch                string           `json:"branch"`
	Repository            string           `json:"repository"`
	Workspace             string           `json:"workspace,omitempty"`
	Paths                 []string         `json:"paths"`
	PolicyIDs             []string         `json:"policy_ids"`
	ScenarioIDs           []string         `json:"scenario_ids"`
	ContractIDs           []string         `json:"contract_ids"`
	DBEntities            []string         `json:"db_entities"`
	MigrationSlots        []string         `json:"migration_slots"`
	UIFlows               []string         `json:"ui_flows"`
	DependencyMajors      []string         `json:"dependency_majors"`
	StableIDs             []string         `json:"stable_ids"`
	RootPointer           bool             `json:"root_pointer"`
	StartsAt              time.Time        `json:"starts_at"`
	ExpiresAt             time.Time        `json:"expires_at"`
	Handoff               *GitLocalHandoff `json:"handoff,omitempty"`
}

// ClaimRevision fingerprints one normalized live item so unrelated coordination changes do not invalidate it.
func ClaimRevision(claim GitLocalClaim) string {
	data, _ := json.Marshal(claim)
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}

// GitLocalHandoff is the exact provider-side checkpoint for a real ownership change.
type GitLocalHandoff struct {
	FromOwner  string    `json:"from_owner"`
	ToOwner    string    `json:"to_owner"`
	Workspace  string    `json:"workspace"`
	Branch     string    `json:"branch"`
	Commit     string    `json:"commit"`
	LocalOnly  bool      `json:"local_only"`
	Evidence   []string  `json:"evidence"`
	Blockers   []string  `json:"blockers"`
	NextAction string    `json:"next_action"`
	RecordedAt time.Time `json:"recorded_at"`
}

// GitLocalClaimActive reports whether a claim still reserves implementation scope.
// An empty status is a backward-compatible in-progress claim from an older client.
func GitLocalClaimActive(claim GitLocalClaim, now time.Time) bool {
	if !claim.ExpiresAt.After(now) {
		return false
	}
	switch claim.Status {
	case "", "in_progress", "review", "blocked":
		return true
	default:
		return false
	}
}

// SnapshotSet is the complete live Git-local claim state at one remote revision.
type SnapshotSet struct {
	SchemaVersion int             `json:"schema_version"`
	Claims        []GitLocalClaim `json:"claims"`
	Revision      string          `json:"-"`
}

// GitLocalStore owns one dedicated remote coordination branch.
type GitLocalStore struct {
	root   string
	remote string
	branch string
}

// NewGitLocalStore creates a store. The remote may be a configured remote name or a safe Git URL/path.
func NewGitLocalStore(root, remote, branch string) GitLocalStore {
	if branch == "" {
		branch = "coordination"
	}
	return GitLocalStore{root: root, remote: remote, branch: branch}
}

// Read fetches and validates the complete live coordination state without touching the user's worktree.
func (store GitLocalStore) Read(ctx context.Context) (SnapshotSet, error) {
	repository, remote, err := store.prepare(ctx)
	if err != nil {
		return SnapshotSet{}, err
	}
	defer os.RemoveAll(repository)
	return store.readPrepared(ctx, repository, remote)
}

// CompareAndSwap publishes next only when the remote still equals expected, then re-reads the postcondition.
func (store GitLocalStore) CompareAndSwap(ctx context.Context, expected string, next SnapshotSet) (string, error) {
	if expected != "" && !objectIDPattern.MatchString(expected) {
		return "", gitLocalFailure(ErrRevisionMismatch, "validate expected revision", nil)
	}
	next = normalizeSnapshotSet(next)
	content, err := canonicalSnapshotSet(next)
	if err != nil {
		return "", gitLocalFailure(ErrMalformedState, "validate next state", err)
	}
	repository, remote, err := store.prepare(ctx)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(repository)
	current, err := store.readPrepared(ctx, repository, remote)
	if err != nil {
		return "", err
	}
	if current.Revision != expected {
		return "", gitLocalFailure(ErrRevisionMismatch, "compare remote revision", nil)
	}

	commit, err := store.createCommit(ctx, repository, expected, content)
	if err != nil {
		return "", err
	}
	lease := "--force-with-lease=refs/heads/" + store.branch + ":" + expected
	refspec := commit + ":refs/heads/" + store.branch
	if _, invokeErr := coordinationGit(ctx, repository, nil, nil, "push", "--porcelain", "--no-verify", lease, remote, refspec); invokeErr != nil {
		kind := ErrPushRejected
		if authenticationFailure(invokeErr) {
			kind = ErrAuthentication
		} else if refreshed, refreshErr := store.Read(ctx); refreshErr == nil && refreshed.Revision != expected {
			kind = ErrRevisionMismatch
		}
		return "", gitLocalFailure(kind, "compare-and-swap push", invokeErr)
	}

	observed, err := store.Read(ctx)
	if err != nil {
		return "", gitLocalFailure(ErrPostconditionMismatch, "re-read pushed state", err)
	}
	observedContent, canonicalErr := canonicalSnapshotSet(observed)
	if canonicalErr != nil || observed.Revision != commit || !bytes.Equal(observedContent, content) {
		return "", gitLocalFailure(ErrPostconditionMismatch, "verify pushed state", canonicalErr)
	}
	return commit, nil
}

func (store GitLocalStore) prepare(ctx context.Context) (string, string, error) {
	root, err := filepath.Abs(store.root)
	if err != nil {
		return "", "", gitLocalFailure(ErrNoRemote, "resolve repository", err)
	}
	if !safeCoordinationBranch(store.branch) {
		return "", "", gitLocalFailure(ErrMalformedState, "validate coordination branch", nil)
	}
	if _, invokeErr := coordinationGit(ctx, root, nil, nil, "rev-parse", "--git-dir"); invokeErr != nil {
		return "", "", gitLocalFailure(ErrNoRemote, "inspect repository", invokeErr)
	}
	remote, err := resolveCoordinationRemote(ctx, root, store.remote)
	if err != nil {
		return "", "", err
	}
	objectFormat, invokeErr := coordinationGit(ctx, root, nil, nil, "rev-parse", "--show-object-format")
	if invokeErr != nil {
		objectFormat = []byte("sha1")
	}
	repository, err := os.MkdirTemp("", "service-coordination-*")
	if err != nil {
		return "", "", gitLocalFailure(ErrNoRemote, "create isolated repository", err)
	}
	args := []string{"init", "--bare", "--quiet"}
	if strings.TrimSpace(string(objectFormat)) == "sha256" {
		args = append(args, "--object-format=sha256")
	}
	args = append(args, ".")
	if _, invokeErr = coordinationGit(ctx, repository, nil, nil, args...); invokeErr != nil {
		_ = os.RemoveAll(repository)
		return "", "", gitLocalFailure(ErrNoRemote, "initialize isolated repository", invokeErr)
	}
	return repository, remote, nil
}

func (store GitLocalStore) readPrepared(ctx context.Context, repository, remote string) (SnapshotSet, error) {
	remoteRef := "refs/heads/" + store.branch
	output, err := coordinationGit(ctx, repository, nil, nil, "ls-remote", "--exit-code", remote, remoteRef)
	if err != nil {
		var invocation *gitInvocationError
		if errors.As(err, &invocation) && invocation.ExitCode == 2 {
			return SnapshotSet{SchemaVersion: 1, Claims: []GitLocalClaim{}}, nil
		}
		kind := ErrNoRemote
		if authenticationFailure(err) {
			kind = ErrAuthentication
		}
		return SnapshotSet{}, gitLocalFailure(kind, "read coordination revision", err)
	}
	fields := strings.Fields(string(output))
	if len(fields) != 2 || fields[1] != remoteRef || !objectIDPattern.MatchString(fields[0]) {
		return SnapshotSet{}, gitLocalFailure(ErrMalformedState, "parse coordination revision", nil)
	}
	revision := fields[0]
	trackingRef := "refs/remotes/service-coordination/source"
	refspec := "+" + remoteRef + ":" + trackingRef
	if _, err = coordinationGit(ctx, repository, nil, nil, "fetch", "--no-tags", "--no-write-fetch-head", remote, refspec); err != nil {
		kind := ErrNoRemote
		if authenticationFailure(err) {
			kind = ErrAuthentication
		}
		return SnapshotSet{}, gitLocalFailure(kind, "fetch coordination revision", err)
	}
	parents, err := coordinationGit(ctx, repository, nil, nil, "rev-list", "--parents", "-n", "1", revision)
	if err != nil {
		return SnapshotSet{}, gitLocalFailure(ErrMalformedState, "inspect coordination history", err)
	}
	if count := len(strings.Fields(string(parents))); count < 1 || count > 2 {
		return SnapshotSet{}, gitLocalFailure(ErrNonFastHistory, "inspect coordination history", nil)
	}
	tree, err := coordinationGit(ctx, repository, nil, nil, "ls-tree", "-r", "-z", "--name-only", revision)
	if err != nil || string(tree) != coordinationFile+"\x00" {
		return SnapshotSet{}, gitLocalFailure(ErrMalformedState, "inspect coordination tree", err)
	}
	content, err := coordinationGit(ctx, repository, nil, nil, "cat-file", "blob", revision+":"+coordinationFile)
	if err != nil {
		return SnapshotSet{}, gitLocalFailure(ErrMalformedState, "read coordination state", err)
	}
	state, err := decodeCanonicalSnapshotSet(content)
	if err != nil {
		return SnapshotSet{}, gitLocalFailure(ErrMalformedState, "decode coordination state", err)
	}
	state.Revision = revision
	return state, nil
}

func (store GitLocalStore) createCommit(ctx context.Context, repository, parent string, content []byte) (string, error) {
	blob, err := coordinationGit(ctx, repository, content, nil, "hash-object", "-w", "--stdin")
	if err != nil {
		return "", gitLocalFailure(ErrMalformedState, "write coordination object", err)
	}
	blobID := strings.TrimSpace(string(blob))
	if !objectIDPattern.MatchString(blobID) {
		return "", gitLocalFailure(ErrMalformedState, "parse coordination object", nil)
	}
	index := filepath.Join(repository, "coordination.index")
	indexEnv := []string{"GIT_INDEX_FILE=" + index}
	readArgs := []string{"read-tree", "--empty"}
	if parent != "" {
		readArgs = []string{"read-tree", parent}
	}
	if _, err = coordinationGit(ctx, repository, nil, indexEnv, readArgs...); err != nil {
		return "", gitLocalFailure(ErrMalformedState, "prepare coordination index", err)
	}
	if _, err = coordinationGit(ctx, repository, nil, indexEnv, "update-index", "--add", "--cacheinfo", "100644", blobID, coordinationFile); err != nil {
		return "", gitLocalFailure(ErrMalformedState, "update coordination index", err)
	}
	tree, err := coordinationGit(ctx, repository, nil, indexEnv, "write-tree")
	if err != nil {
		return "", gitLocalFailure(ErrMalformedState, "write coordination tree", err)
	}
	args := []string{"commit-tree", strings.TrimSpace(string(tree))}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	identity := []string{
		"GIT_AUTHOR_NAME=Service Coordination", "GIT_AUTHOR_EMAIL=coordination@example.invalid",
		"GIT_COMMITTER_NAME=Service Coordination", "GIT_COMMITTER_EMAIL=coordination@example.invalid",
	}
	commit, err := coordinationGit(ctx, repository, []byte("chore: update coordination state\n"), identity, args...)
	if err != nil {
		return "", gitLocalFailure(ErrMalformedState, "create coordination commit", err)
	}
	commitID := strings.TrimSpace(string(commit))
	if !objectIDPattern.MatchString(commitID) {
		return "", gitLocalFailure(ErrMalformedState, "parse coordination commit", nil)
	}
	return commitID, nil
}

func resolveCoordinationRemote(ctx context.Context, root, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\x00\r\n") {
		return "", gitLocalFailure(ErrNoRemote, "validate remote", nil)
	}
	if branchNamePattern.MatchString(value) && !strings.ContainsAny(value, "/\\:") {
		output, err := coordinationGit(ctx, root, nil, nil, "remote", "get-url", "--push", value)
		if err != nil {
			return "", gitLocalFailure(ErrNoRemote, "resolve configured remote", err)
		}
		value = strings.TrimSpace(string(output))
	}
	if !safeCoordinationRemote(root, value) {
		return "", gitLocalFailure(ErrNoRemote, "validate remote transport", nil)
	}
	if !filepath.IsAbs(value) && !strings.Contains(value, "://") && !looksLikeSCPRemote(value) {
		value = filepath.Clean(filepath.Join(root, value))
	}
	return value, nil
}

func safeCoordinationRemote(root, value string) bool {
	lower := strings.ToLower(value)
	if value == "" || strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\x00\r\n") || strings.HasPrefix(lower, "ext::") {
		return false
	}
	if filepath.IsAbs(value) {
		return true
	}
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "ssh://") || strings.HasPrefix(lower, "git://") || looksLikeSCPRemote(value) {
		return true
	}
	if strings.Contains(value, "://") {
		return false
	}
	_, err := os.Stat(filepath.Join(root, value))
	return err == nil
}

func looksLikeSCPRemote(value string) bool {
	separator := strings.IndexByte(value, ':')
	return separator > 0 && strings.Contains(value[:separator], "@") && !strings.ContainsAny(value, "\x00\r\n")
}

func safeCoordinationBranch(value string) bool {
	return branchNamePattern.MatchString(value) && !strings.Contains(value, "..") && !strings.Contains(value, "@{") && !strings.HasSuffix(value, ".") && !strings.HasSuffix(value, ".lock") && !strings.Contains(value, "//")
}

func normalizeSnapshotSet(value SnapshotSet) SnapshotSet {
	value.Revision = ""
	if value.Claims == nil {
		value.Claims = []GitLocalClaim{}
	}
	for index := range value.Claims {
		claim := &value.Claims[index]
		claim.StartsAt = claim.StartsAt.UTC()
		claim.ExpiresAt = claim.ExpiresAt.UTC()
		claim.Paths = normalizedCoordinationSet(claim.Paths)
		claim.PolicyIDs = normalizedCoordinationSet(claim.PolicyIDs)
		claim.ScenarioIDs = normalizedCoordinationSet(claim.ScenarioIDs)
		claim.ContractIDs = normalizedCoordinationSet(claim.ContractIDs)
		claim.DBEntities = normalizedCoordinationSet(claim.DBEntities)
		claim.MigrationSlots = normalizedCoordinationSet(claim.MigrationSlots)
		claim.UIFlows = normalizedCoordinationSet(claim.UIFlows)
		claim.DependencyMajors = normalizedCoordinationSet(claim.DependencyMajors)
		claim.StableIDs = normalizedCoordinationSet(claim.StableIDs)
		if claim.Handoff != nil {
			claim.Handoff.RecordedAt = claim.Handoff.RecordedAt.UTC()
			claim.Handoff.Evidence = normalizedCoordinationSet(claim.Handoff.Evidence)
			claim.Handoff.Blockers = normalizedCoordinationSet(claim.Handoff.Blockers)
		}
	}
	sort.Slice(value.Claims, func(left, right int) bool { return value.Claims[left].ID < value.Claims[right].ID })
	return value
}

func canonicalSnapshotSet(value SnapshotSet) ([]byte, error) {
	value = normalizeSnapshotSet(value)
	if err := validateSnapshotSet(value); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decodeCanonicalSnapshotSet(content []byte) (SnapshotSet, error) {
	value, err := schema.DecodeJSON[SnapshotSet](content)
	if err != nil {
		return SnapshotSet{}, err
	}
	canonical, err := canonicalSnapshotSet(value)
	if err != nil {
		return SnapshotSet{}, err
	}
	if !bytes.Equal(content, canonical) {
		return SnapshotSet{}, fmt.Errorf("coordination state is not canonical JSON")
	}
	return normalizeSnapshotSet(value), nil
}

func validateSnapshotSet(value SnapshotSet) error {
	if value.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1")
	}
	claims, work := map[string]bool{}, map[string]bool{}
	for _, claim := range value.Claims {
		if !coordinationIDPattern.MatchString(claim.ID) || !strings.HasPrefix(claim.ID, "claim.") || !coordinationIDPattern.MatchString(claim.WorkID) || !strings.HasPrefix(claim.WorkID, "work.") {
			return fmt.Errorf("claim and work IDs must be stable IDs")
		}
		if claims[claim.ID] || work[claim.WorkID] {
			return fmt.Errorf("claim and work IDs must be unique")
		}
		claims[claim.ID], work[claim.WorkID] = true, true
		if !coordinationDigestPattern.MatchString(claim.DefinitionFingerprint) || strings.TrimSpace(claim.Owner) == "" || strings.TrimSpace(claim.Repository) == "" || gitx.ValidateBranch(claim.Branch) != nil {
			return fmt.Errorf("claim identity, fingerprint, owner, repository, or branch is invalid")
		}
		switch claim.Status {
		case "", "in_progress", "blocked", "review", "integrated", "done":
		default:
			return fmt.Errorf("claim lifecycle status is invalid")
		}
		if claim.StartsAt.IsZero() || !claim.ExpiresAt.After(claim.StartsAt) {
			return fmt.Errorf("claim lease must have ordered start and expiry times")
		}
		if claim.Handoff != nil {
			handoff := claim.Handoff
			if strings.TrimSpace(handoff.FromOwner) == "" || strings.TrimSpace(handoff.ToOwner) == "" || handoff.FromOwner == handoff.ToOwner || handoff.ToOwner != claim.Owner || handoff.Branch != claim.Branch || !objectIDPattern.MatchString(handoff.Commit) || !strings.HasPrefix(handoff.Workspace, "workspace.") || strings.TrimSpace(handoff.NextAction) == "" || handoff.RecordedAt.IsZero() {
				return fmt.Errorf("claim handoff identity is invalid")
			}
			for _, id := range handoff.Evidence {
				if !handoffEvidencePattern.MatchString(id) {
					return fmt.Errorf("claim handoff evidence is invalid")
				}
			}
			for _, id := range handoff.Blockers {
				if !handoffRefPattern.MatchString(id) {
					return fmt.Errorf("claim handoff blocker is invalid")
				}
			}
		}
		for _, path := range claim.Paths {
			clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
			if path == "" || filepath.IsAbs(path) || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsAny(path, "\x00\r\n") {
				return fmt.Errorf("claim path is unsafe")
			}
		}
	}
	return nil
}

func normalizedCoordinationSet(values []string) []string {
	result, seen := []string{}, map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

type gitInvocationError struct {
	ExitCode int
	Stderr   string
	Auth     bool
}

func (err *gitInvocationError) Error() string { return "restricted Git command failed" }

func coordinationGit(ctx context.Context, directory string, stdin []byte, extraEnvironment []string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = directory
	command.Env = coordinationEnvironment(extraEnvironment...)
	command.Stdin = bytes.NewReader(stdin)
	var stdout, stderr coordinationBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		exitCode := -1
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		}
		rawError := stderr.String() + "\n" + stdout.String()
		return nil, &gitInvocationError{ExitCode: exitCode, Stderr: sanitizedGitError(rawError), Auth: containsAuthenticationFailure(rawError)}
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func coordinationEnvironment(extra ...string) []string {
	keys := []string{"PATH", "HOME", "SystemRoot", "SYSTEMROOT", "TEMP", "TMP", "TMPDIR", "USERPROFILE"}
	result := make([]string, 0, len(keys)+len(extra)+7)
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			result = append(result, key+"="+value)
		}
	}
	result = append(result, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1", "GIT_ALLOW_PROTOCOL=https:ssh:git:file", "GCM_INTERACTIVE=Never", "LC_ALL=C")
	return append(result, extra...)
}

type coordinationBuffer struct{ bytes.Buffer }

func (buffer *coordinationBuffer) Write(data []byte) (int, error) {
	original := len(data)
	remaining := (4 << 20) - buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}
		_, _ = buffer.Buffer.Write(data)
	}
	return original, nil
}

func authenticationFailure(err error) bool {
	var invocation *gitInvocationError
	return errors.As(err, &invocation) && invocation.Auth
}

func containsAuthenticationFailure(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"authentication failed", "could not read username", "permission denied (publickey)", "terminal prompts disabled", "authentication is required"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func sanitizedGitError(value string) string {
	value = strings.TrimSpace(value)
	for _, marker := range []string{"token=", "password=", "secret="} {
		if index := strings.Index(strings.ToLower(value), marker); index >= 0 {
			value = value[:index+len(marker)] + "[REDACTED]"
		}
	}
	if len(value) > 1024 {
		value = value[:1024]
	}
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func gitLocalFailure(kind error, operation string, cause error) error {
	return &GitLocalError{Kind: kind, Operation: operation, Cause: cause}
}
