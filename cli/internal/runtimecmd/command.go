// Package runtimecmd adapts the public core to a model-free CLI.
package runtimecmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kcrmin/Stackcord/core/coordination"
	"github.com/kcrmin/Stackcord/core/httpapi"
	"github.com/spf13/cobra"
)

type connection struct{ endpoint, tokenFile string }

func (c *connection) token() (string, error) {
	token := os.Getenv("STACKCORD_TOKEN")
	file := c.tokenFile
	if file == "" && token == "" {
		file = defaultLocalCredential(c)
	}
	if file != "" {
		f, e := os.Open(file)
		if e != nil {
			return "", e
		}
		defer f.Close()
		b, e := io.ReadAll(io.LimitReader(f, 4097))
		if e != nil {
			return "", e
		}
		if len(b) > 4096 {
			return "", fmt.Errorf("credential file too large")
		}
		token = strings.TrimSpace(string(b))
	}
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return "", fmt.Errorf("set STACKCORD_TOKEN or --token-file to a private runtime credential")
	}
	return token, nil
}
func (c *connection) client() (*httpapi.Client, error) {
	token, e := c.token()
	if e != nil {
		return nil, e
	}
	return httpapi.NewClient(c.endpoint, token)
}
func New(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "stackcord", Short: "Coordinate AI work across sessions with durable, bounded context", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().Bool("json", true, "runtime commands always return JSON")
	Register(root)
	return root
}

// Register allows the full distribution to attach the optional harness without
// coupling the core module or this adapter to any Git workflow.
func Register(root *cobra.Command) {
	c := &connection{}
	root.PersistentFlags().StringVar(&c.endpoint, "endpoint", "http://127.0.0.1:7331", "runtime URL (HTTPS or loopback HTTP)")
	root.PersistentFlags().StringVar(&c.tokenFile, "token-file", "", "private bearer credential file; otherwise STACKCORD_TOKEN")
	root.AddCommand(newStart(), newServe(c))
	emit := func(cmd *cobra.Command, v any, e error) error {
		if e != nil {
			return e
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(v)
	}
	tasks := &cobra.Command{Use: "task", Short: "Create, claim, checkpoint, pause or finish durable work"}
	var input, project string
	apply := &cobra.Command{Use: "apply", Short: "Apply one idempotent JSON mutation", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		var reader io.Reader = cmd.InOrStdin()
		if input != "-" {
			f, e := os.Open(input)
			if e != nil {
				return e
			}
			defer f.Close()
			reader = f
		}
		raw, e := io.ReadAll(io.LimitReader(reader, coordination.MaxMutationBytes+1))
		if e != nil {
			return e
		}
		if len(raw) > coordination.MaxMutationBytes {
			return fmt.Errorf("mutation exceeds 64 KiB")
		}
		var m coordination.Mutation
		d := json.NewDecoder(strings.NewReader(string(raw)))
		d.DisallowUnknownFields()
		if e = d.Decode(&m); e != nil {
			return e
		}
		if e = d.Decode(new(any)); e != io.EOF {
			return fmt.Errorf("exactly one mutation required")
		}
		client, e := c.client()
		if e != nil {
			return e
		}
		v, e := client.Apply(cmd.Context(), m)
		return emit(cmd, v, e)
	}}
	apply.Flags().StringVar(&input, "input", "-", "mutation JSON file, or - for stdin")
	list := &cobra.Command{Use: "list", Short: "List current tasks in one project", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client, e := c.client()
		if e != nil {
			return e
		}
		v, e := client.Tasks(cmd.Context(), project)
		return emit(cmd, v, e)
	}}
	list.Flags().StringVar(&project, "project", "", "project ID")
	_ = list.MarkFlagRequired("project")
	tasks.AddCommand(apply, list)
	root.AddCommand(tasks)
	var resumeProject, task string
	var budget int
	resume := &cobra.Command{Use: "resume", Short: "Read current task context within an explicit byte budget", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client, e := c.client()
		if e != nil {
			return e
		}
		v, e := client.Context(cmd.Context(), resumeProject, task, budget)
		return emit(cmd, v, e)
	}}
	resume.Flags().StringVar(&resumeProject, "project", "", "project ID")
	resume.Flags().StringVar(&task, "task", "", "task ID")
	resume.Flags().IntVar(&budget, "max-bytes", coordination.DefaultContextBytes, "maximum JSON context bytes; never silently truncates")
	_ = resume.MarkFlagRequired("project")
	_ = resume.MarkFlagRequired("task")
	root.AddCommand(resume)
	var eventProject string
	var after uint64
	var limit int
	events := &cobra.Command{Use: "events", Short: "Read compact changes since a project cursor", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client, e := c.client()
		if e != nil {
			return e
		}
		v, e := client.Events(cmd.Context(), eventProject, after, limit)
		return emit(cmd, v, e)
	}}
	events.Flags().StringVar(&eventProject, "project", "", "project ID")
	events.Flags().Uint64Var(&after, "after", 0, "last processed event cursor")
	events.Flags().IntVar(&limit, "limit", 100, "page size (1-100)")
	_ = events.MarkFlagRequired("project")
	root.AddCommand(events)
	root.AddCommand(&cobra.Command{Use: "projects", Short: "List projects preserved by the runtime", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client, e := c.client()
		if e != nil {
			return e
		}
		v, e := client.Projects(cmd.Context())
		return emit(cmd, v, e)
	}})
	var output string
	backup := &cobra.Command{Use: "backup", Short: "Save a consistent runtime database without overwriting files", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		client, e := c.client()
		if e != nil {
			return e
		}
		f, e := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		ok := false
		defer func() {
			f.Close()
			if !ok {
				_ = os.Remove(output)
			}
		}()
		if e = client.Backup(cmd.Context(), f); e != nil {
			return e
		}
		if e = f.Sync(); e != nil {
			return e
		}
		if e = f.Close(); e != nil {
			return e
		}
		ok = true
		return emit(cmd, map[string]string{"backup": output}, nil)
	}}
	backup.Flags().StringVar(&output, "output", "", "new backup file path")
	_ = backup.MarkFlagRequired("output")
	root.AddCommand(backup)
}
