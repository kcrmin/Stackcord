package command

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/kcrmin/Stackcord/cli/internal/controlcenter"
	"github.com/kcrmin/Stackcord/cli/internal/dashboard"
	"github.com/kcrmin/Stackcord/cli/internal/domain"
	"github.com/spf13/cobra"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func newDashboardCommand() *cobra.Command {
	var root string
	var port int
	cmd := &cobra.Command{Use: "dashboard", Short: "Open the optional local settings and approvals UI", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		root, err := controlcenter.ResolveRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		info, err := os.Stat(root)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("project root must be a directory")
		}
		if port < 0 || port > 65535 {
			return fmt.Errorf("invalid port")
		}
		listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return err
		}
		defer listener.Close()
		bytes := make([]byte, 32)
		if _, err = rand.Read(bytes); err != nil {
			return err
		}
		token := hex.EncodeToString(bytes)
		backend := &controlcenter.Backend{Root: root}
		server := &http.Server{Handler: dashboard.New(backend, token, listener.Addr().String()), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 60 * time.Second}
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer cancel()
		go func() {
			<-ctx.Done()
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdown)
		}()
		fmt.Fprintf(cmd.OutOrStdout(), "http://%s/#token=%s\nDashboard stays available while this command is running. Press Ctrl+C to stop.\n", listener.Addr(), token)
		if err = server.Serve(listener); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}}
	cmd.Flags().StringVar(&root, "root", ".", "project directory")
	cmd.Flags().IntVar(&port, "port", 0, "loopback port; 0 selects an available port")
	return cmd
}
func newSetupCommand(version string, jsonOutput *bool) *cobra.Command {
	var root, ui string
	var apply bool
	cmd := &cobra.Command{Use: "setup", Short: "Inspect or save the optional UI preference", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		root, err := controlcenter.ResolveRoot(cmd.Context(), root)
		if err != nil {
			return err
		}
		s, rev, err := controlcenter.ReadSettings(root)
		if err != nil {
			return err
		}
		summary := "UI preference: " + s.UIPreference + ". Choose enable, disable, or ask; the plugin works with every choice."
		if ui != "" {
			choices := map[string]string{"enable": "enabled", "disable": "declined", "ask": "unset"}
			v, ok := choices[ui]
			if !ok {
				return fmt.Errorf("ui must be enable, disable, or ask")
			}
			s.UIPreference = v
			if apply {
				_, err = controlcenter.SavePersonal(cmd.Context(), root, rev, s)
				if err != nil {
					return err
				}
				summary = "UI preference saved: " + v
			} else {
				summary = "UI preference preview: " + v + "; pass --apply to save."
			}
		}
		result := domain.Result{SchemaVersion: "1.0", ToolVersion: version, Command: "setup", OperationID: "setup", Status: domain.StatusPassed, Summary: summary, Facts: []domain.Item{{Code: "setup.ui", Message: s.UIPreference}}}
		return writeResult(cmd, *jsonOutput, result)
	}}
	cmd.Flags().StringVar(&root, "root", ".", "project directory")
	cmd.Flags().StringVar(&ui, "ui", "", "enable|disable|ask")
	cmd.Flags().BoolVar(&apply, "apply", false, "save this explicit preference")
	return cmd
}
