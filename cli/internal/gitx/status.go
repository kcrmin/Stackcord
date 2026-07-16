package gitx

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// State is a read-only snapshot of actual local and remote-tracking Git state.
type State struct {
	Root       string      `json:"root"`
	Branch     string      `json:"branch"`
	Head       string      `json:"head"`
	Upstream   string      `json:"upstream,omitempty"`
	Dirty      bool        `json:"dirty"`
	Detached   bool        `json:"detached"`
	Ahead      int         `json:"ahead"`
	Behind     int         `json:"behind"`
	Diverged   bool        `json:"diverged"`
	Submodules []Submodule `json:"submodules"`
}

// Inspect reports actual Git state without fetching or changing the repository.
func Inspect(ctx context.Context, root string) (State, error) {
	git := runner{}
	top, err := git.read(ctx, root, "rev-parse", "--show-toplevel")
	if err != nil {
		return State{}, err
	}
	top, err = filepath.Abs(top)
	if err != nil {
		return State{}, err
	}
	head, err := git.read(ctx, top, "rev-parse", "HEAD")
	if err != nil {
		return State{}, err
	}
	branch, err := git.read(ctx, top, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return State{}, err
	}
	status, err := git.read(ctx, top, "status", "--porcelain=v2", "--untracked-files=all")
	if err != nil {
		return State{}, err
	}
	state := State{Root: top, Branch: branch, Head: head, Dirty: status != "", Detached: branch == "HEAD", Submodules: []Submodule{}}
	if upstream, upstreamErr := git.read(ctx, top, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); upstreamErr == nil {
		state.Upstream = upstream
		counts, countErr := git.read(ctx, top, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
		if countErr != nil {
			return State{}, countErr
		}
		parts := strings.Fields(counts)
		if len(parts) != 2 {
			return State{}, fmt.Errorf("unexpected Git divergence count %q", counts)
		}
		state.Ahead, _ = strconv.Atoi(parts[0])
		state.Behind, _ = strconv.Atoi(parts[1])
		state.Diverged = state.Ahead > 0 && state.Behind > 0
	}
	state.Submodules, err = inspectSubmodules(ctx, git, top)
	return state, err
}
