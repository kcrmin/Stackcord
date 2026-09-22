package runtimecmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func localDirectory() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Stackcord"), nil
}
func prepareLocal(dir string) (string, string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", err
	}
	file := filepath.Join(dir, "token")
	if info, err := os.Lstat(file); err == nil {
		if !info.Mode().IsRegular() {
			return "", "", fmt.Errorf("credential must be a regular private file: %s", file)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
			return "", "", fmt.Errorf("credential must be private (chmod 600): %s", file)
		}
	} else if !os.IsNotExist(err) {
		return "", "", err
	}
	f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err == nil {
		var raw [32]byte
		if _, err = rand.Read(raw[:]); err == nil {
			_, err = f.WriteString(hex.EncodeToString(raw[:]) + "\n")
		}
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return "", "", err
		}
	} else if !os.IsExist(err) {
		return "", "", err
	}
	token, err := (&connection{tokenFile: file}).token()
	if err == nil && len(token) < 32 {
		err = fmt.Errorf("existing credential is too short; retain it and choose another --data-dir")
	}
	return token, file, err
}
func newStart() *cobra.Command {
	var dir string
	var port int
	var noOpen bool
	cmd := &cobra.Command{Use: "start", Short: "Open Stackcord with saved local data; no setup flags needed", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if port < 0 || port > 65535 {
			return fmt.Errorf("port must be 0-65535")
		}
		if dir == "" {
			var err error
			dir, err = localDirectory()
			if err != nil {
				return err
			}
		}
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return err
		}
		token, file, err := prepareLocal(dir)
		if err != nil {
			return err
		}
		var raw [32]byte
		if _, err = rand.Read(raw[:]); err != nil {
			return err
		}
		return runServer(cmd, filepath.Join(dir, "state.db"), port, token, true, &localOptions{code: hex.EncodeToString(raw[:]), file: file, noOpen: noOpen})
	}}
	cmd.Flags().StringVar(&dir, "data-dir", "", "private data directory; defaults to the user configuration directory/Stackcord")
	cmd.Flags().IntVar(&port, "port", 7331, "local port; 0 selects a free port")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "print the one-use browser link without opening a browser")
	return cmd
}

type localOptions struct {
	code, file string
	noOpen     bool
}

func openBrowser(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	// No shell interpolation; the argument contains only a short-lived launch code.
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }()
	return nil
}
func defaultLocalCredential(c *connection) string {
	if c.endpoint != "http://127.0.0.1:7331" {
		return ""
	}
	dir, err := localDirectory()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "token")
}
