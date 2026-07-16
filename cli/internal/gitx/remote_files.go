package gitx

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
)

// RemoteFile is one file observed in an already-fetched remote-tracking ref.
type RemoteFile struct {
	Ref  string
	Path string
	Data []byte
}

// ReadRemoteFiles reads a constrained directory from fetched remote refs without checkout or fetch.
func ReadRemoteFiles(ctx context.Context, root, directory, suffix string) ([]RemoteFile, error) {
	directory = path.Clean(strings.ReplaceAll(directory, "\\", "/"))
	if directory == "." || directory == ".." || strings.HasPrefix(directory, "../") || strings.HasPrefix(directory, "/") || strings.ContainsAny(directory, "\x00\r\n") {
		return nil, fmt.Errorf("safe relative remote directory is required")
	}
	if suffix == "" || strings.ContainsAny(suffix, "/\\\x00\r\n") {
		return nil, fmt.Errorf("safe file suffix is required")
	}
	git := runner{}
	refsOutput, err := git.read(ctx, root, "for-each-ref", "--format=%(refname)", "refs/remotes")
	if err != nil {
		return nil, err
	}
	refs := strings.Fields(refsOutput)
	var files []RemoteFile
	seen := map[string]bool{}
	for _, ref := range refs {
		if !strings.HasPrefix(ref, "refs/remotes/") || strings.HasSuffix(ref, "/HEAD") || strings.ContainsAny(ref, "\x00\r\n") {
			continue
		}
		pathsOutput, err := git.read(ctx, root, "ls-tree", "-r", "-z", "--name-only", ref, "--", directory)
		if err != nil {
			return nil, err
		}
		for _, filePath := range strings.Split(pathsOutput, "\x00") {
			if filePath == "" {
				continue
			}
			if strings.ContainsAny(filePath, "\r\n\\") || !strings.HasPrefix(filePath, directory+"/") || !strings.HasSuffix(filePath, suffix) {
				continue
			}
			key := ref + "\x00" + filePath
			if seen[key] {
				continue
			}
			seen[key] = true
			content, err := git.read(ctx, root, "cat-file", "blob", ref+":"+filePath)
			if err != nil {
				return nil, err
			}
			files = append(files, RemoteFile{Ref: ref, Path: filePath, Data: []byte(content)})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Ref == files[j].Ref {
			return files[i].Path < files[j].Path
		}
		return files[i].Ref < files[j].Ref
	})
	return files, nil
}
