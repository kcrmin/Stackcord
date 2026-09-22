package runtimecmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/kcrmin/Stackcord/cli/internal/console"
	"github.com/kcrmin/Stackcord/core/coordination"
	"github.com/kcrmin/Stackcord/core/httpapi"
	"github.com/spf13/cobra"
)

func newServe(c *connection) *cobra.Command {
	var db string
	var port int
	var ui bool
	cmd := &cobra.Command{Use: "serve", Short: "Run the durable HTTP coordinator; --ui adds the web console", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if port < 0 || port > 65535 {
			return fmt.Errorf("port must be 0-65535")
		}
		token, e := c.token()
		if e != nil {
			return e
		}
		if len(token) < 32 {
			return fmt.Errorf("server credential must contain at least 32 characters")
		}
		return runServer(cmd, db, port, token, ui, nil)
	}}
	cmd.Flags().StringVar(&db, "db", "", "durable database outside Git (required; back it up)")
	cmd.Flags().IntVar(&port, "port", 7331, "loopback port; 0 selects a free port")
	cmd.Flags().BoolVar(&ui, "ui", false, "serve the optional web console")
	_ = cmd.MarkFlagRequired("db")
	return cmd
}

func runServer(cmd *cobra.Command, db string, port int, token string, ui bool, local *localOptions) error {
	listener, e := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if e != nil {
		return e
	}
	defer listener.Close()
	store, e := coordination.Open(db)
	if e != nil {
		return e
	}
	defer store.Close()
	var handler http.Handler = httpapi.New(store, token, listener.Addr().String())
	if ui {
		if local != nil {
			handler = console.NewLocal(handler, listener.Addr().String(), token, local.code, local.file)
		} else {
			handler = console.New(handler, listener.Addr().String())
		}
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		<-ctx.Done()
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if e := server.Shutdown(shutdown); e != nil {
			_ = server.Close()
		}
	}()
	fmt.Fprintf(cmd.OutOrStdout(), "http://%s\nRuntime database: %s\nWeb console: %t. API credential is never printed. Stop with Ctrl+C.\n", listener.Addr(), db, ui)
	if local != nil {
		url := "http://" + listener.Addr().String() + "/#connect=" + local.code
		fmt.Fprintf(cmd.OutOrStdout(), "Open Stackcord (one-use link, valid 5 minutes):\n%s\n", url)
		if !local.noOpen {
			if err := openBrowser(url); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Open the link above in your browser.")
			}
		}
	}
	e = server.Serve(listener)
	cancel()
	<-stopped
	if e != nil && e != http.ErrServerClosed {
		return e
	}
	return nil
}
