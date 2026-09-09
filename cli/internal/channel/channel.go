// Package channel implements an opt-in signed Git mailbox. Messages confer no policy authority.
package channel

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const maxEvents = 1000
const maxPayload = 32768

var ErrNotConfigured = errors.New("channel is not configured")
var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,79}$`)

type RunnerConfig struct {
	ResultFormat   string   `json:"result_format,omitempty"`
	Argv           []string `json:"argv"`
	AllowedKinds   []string `json:"allowed_kinds"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}
type RunnerState struct {
	ResultFormat   string   `json:"result_format,omitempty"`
	Enabled        bool     `json:"enabled"`
	AllowedKinds   []string `json:"allowed_kinds"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}
type Config struct {
	Channel string            `json:"channel"`
	Remote  string            `json:"remote"`
	Peer    string            `json:"peer"`
	Peers   map[string]string `json:"peers"`
	Runner  RunnerConfig      `json:"runner"`
}
type diskConfig struct {
	Config
	PrivateKey string `json:"private_key"`
}
type RequestInput struct {
	To           string   `json:"to"`
	Kind         string   `json:"kind"`
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	Dependencies []string `json:"dependencies"`
	Scope        []string `json:"scope"`
}
type ResultInput struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Body      string `json:"body"`
}
type Event struct {
	ID           string   `json:"id"`
	Channel      string   `json:"channel"`
	Type         string   `json:"type"`
	Author       string   `json:"author"`
	To           string   `json:"to,omitempty"`
	Kind         string   `json:"kind,omitempty"`
	Title        string   `json:"title,omitempty"`
	Body         string   `json:"body"`
	RequestID    string   `json:"request_id,omitempty"`
	Status       string   `json:"status,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Scope        []string `json:"scope,omitempty"`
	CreatedAt    string   `json:"created_at"`
	Signature    string   `json:"signature"`
}
type RequestState struct {
	Request       Event  `json:"request"`
	Status        string `json:"status"`
	BlockedReason string `json:"blocked_reason,omitempty"`
	Result        *Event `json:"result,omitempty"`
}
type State struct {
	WorkerBlockedReason string            `json:"worker_blocked_reason,omitempty"`
	Configured          bool              `json:"configured"`
	Channel             string            `json:"channel"`
	Remote              string            `json:"remote"`
	Peer                string            `json:"peer"`
	PublicKey           string            `json:"public_key"`
	Peers               map[string]string `json:"peers"`
	Runner              RunnerState       `json:"runner"`
	Requests            []RequestState    `json:"requests"`
	Revision            string            `json:"revision"`
}
type Store struct {
	root, dir        string
	ExpectedRevision string
}

func Open(root string) (*Store, error) {
	p, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	return &Store{root: p, dir: filepath.Join(p, ".harness", "local", "channel")}, nil
}
func safePath(p string) error {
	for q := p; ; q = filepath.Dir(q) {
		info, e := os.Lstat(q)
		if e == nil && (info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("unsafe symbolic link: %s", q)
		}
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if filepath.Dir(q) == q {
			break
		}
	}
	return nil
}
func (s *Store) lock() (func(), error) {
	if e := safePath(s.dir); e != nil {
		return nil, e
	}
	if e := os.MkdirAll(s.dir, 0700); e != nil {
		return nil, e
	}
	if e := safePath(filepath.Join(s.dir, "lock")); e != nil {
		return nil, e
	}
	return fileLock(filepath.Join(s.dir, "lock"))
}
func writeJSON(p string, v any) error {
	if e := safePath(p + ".tmp"); e != nil {
		return e
	}
	if e := safePath(p); e != nil {
		return e
	}
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(p+".tmp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if e != nil {
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(p+".tmp", p)
}
func (s *Store) load() (diskConfig, error) {
	var c diskConfig
	p := filepath.Join(s.dir, "config.json")
	if e := safePath(p); e != nil {
		return c, e
	}
	i, e := os.Stat(p)
	if os.IsNotExist(e) {
		return c, ErrNotConfigured
	}
	if e != nil {
		return c, e
	}
	if i.Size() > 65536 || !i.Mode().IsRegular() || (runtime.GOOS != "windows" && i.Mode().Perm()&0077 != 0) {
		return c, errors.New("unsafe private configuration permissions")
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(b, &c)
	if e != nil {
		return c, e
	}
	key, e := base64.StdEncoding.DecodeString(c.PrivateKey)
	if e != nil || len(key) != 64 || !bytes.Equal(ed25519.NewKeyFromSeed(key[:32]), key) || base64.StdEncoding.EncodeToString(key[32:]) != c.Peers[c.Peer] {
		return c, errors.New("invalid local private key or identity pin")
	}
	if !identifier.MatchString(c.Channel) || !identifier.MatchString(c.Peer) || !validRemote(c.Remote) || len(c.Peers) > 128 {
		return c, errors.New("invalid local channel configuration")
	}
	for peer, key := range c.Peers {
		if !identifier.MatchString(peer) || !validKey(key) {
			return c, errors.New("invalid local peer pin")
		}
	}
	if e = validateRunner(c.Runner); e != nil {
		return c, e
	}
	return c, e
}
func validRemote(r string) bool {
	if len(r) > 2048 || r == "" || strings.HasPrefix(r, "-") || strings.ContainsAny(r, "\r\n\x00") || strings.Contains(r, "::") {
		return false
	}
	if strings.Contains(r, "://") {
		u, e := url.Parse(r)
		return e == nil && (u.Scheme == "https" || u.Scheme == "ssh") && u.User == nil && u.RawQuery == "" && u.Fragment == ""
	}
	return true
}
func (s *Store) Setup(c Config) (State, error) {
	if s.ExpectedRevision != "" && s.ExpectedRevision != "unconfigured" {
		return State{}, errors.New("channel configuration changed; refresh")
	}
	unlock, e := s.lock()
	if e != nil {
		return State{}, e
	}
	defer unlock()
	if _, e = s.load(); !errors.Is(e, ErrNotConfigured) {
		return State{}, errors.New("channel already configured or unsafe")
	}
	if !identifier.MatchString(c.Channel) || !identifier.MatchString(c.Peer) || !validRemote(c.Remote) {
		return State{}, errors.New("invalid channel, peer or remote")
	}
	if e = validateRunner(c.Runner); e != nil {
		return State{}, e
	}
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		return State{}, e
	}
	if len(c.Peers) > 127 {
		return State{}, errors.New("too many peers")
	}
	if c.Peers == nil {
		c.Peers = map[string]string{}
	}
	for p, k := range c.Peers {
		if !identifier.MatchString(p) || !validKey(k) {
			return State{}, errors.New("invalid trusted peer")
		}
	}
	c.Peers[c.Peer] = base64.StdEncoding.EncodeToString(pub)
	dc := diskConfig{c, base64.StdEncoding.EncodeToString(key)}
	if _, e = s.git(context.Background(), nil, "init", "--bare", filepath.Join(s.dir, "store.git")); e != nil {
		return State{}, e
	}
	if e = secureDirectory(s.dir); e != nil {
		return State{}, e
	}
	if s.ExpectedRevision != "" && s.ExpectedRevision != "unconfigured" {
		return State{}, errors.New("channel configuration changed; refresh")
	}
	if e = writeJSON(filepath.Join(s.dir, "config.json"), dc); e != nil {
		return State{}, e
	}
	return s.state(dc, nil), nil
}
func validKey(k string) bool {
	b, e := base64.StdEncoding.DecodeString(k)
	return e == nil && len(b) == ed25519.PublicKeySize
}
func (s *Store) Trust(peer, key string) (State, error) {
	u, e := s.lock()
	if e != nil {
		return State{}, e
	}
	defer u()
	c, e := s.load()
	if e != nil {
		return State{}, e
	}
	if e = s.checkRevision(c); e != nil {
		return State{}, e
	}
	if !identifier.MatchString(peer) || !validKey(key) {
		return State{}, errors.New("invalid peer key")
	}
	if old, ok := c.Peers[peer]; ok && old != key {
		return State{}, errors.New("peer already pinned to another key")
	}
	if _, exists := c.Peers[peer]; !exists && len(c.Peers) >= 128 {
		return State{}, errors.New("too many peers")
	}
	c.Peers[peer] = key
	if e = writeJSON(filepath.Join(s.dir, "config.json"), c); e != nil {
		return State{}, e
	}
	events, _, e := s.read(context.Background(), c, false)
	return s.state(c, events), e
}
func validateRunner(r RunnerConfig) error {
	if r.ResultFormat != "" && r.ResultFormat != "text" && r.ResultFormat != "json" {
		return errors.New("invalid runner result format")
	}
	total := 0
	for _, arg := range r.Argv {
		total += len(arg)
		if strings.ContainsRune(arg, 0) {
			return errors.New("invalid runner argument")
		}
	}
	if total > 16000 || len(r.AllowedKinds) > 128 {
		return errors.New("runner configuration exceeds limit")
	}
	if len(r.Argv) > 32 || r.TimeoutSeconds < 0 || r.TimeoutSeconds > 3600 {
		return errors.New("invalid runner limits")
	}
	if len(r.Argv) > 0 && (!filepath.IsAbs(r.Argv[0]) || len(r.AllowedKinds) == 0) {
		return errors.New("runner requires absolute executable and allowed kinds")
	}
	for _, k := range r.AllowedKinds {
		if !identifier.MatchString(k) {
			return errors.New("invalid runner kind")
		}
	}
	return nil
}
func (s *Store) ConfigureRunner(r RunnerConfig) (State, error) {
	u, e := s.lock()
	if e != nil {
		return State{}, e
	}
	defer u()
	c, e := s.load()
	if e != nil {
		return State{}, e
	}
	if e = s.checkRevision(c); e != nil {
		return State{}, e
	}
	if e = validateRunner(r); e != nil {
		return State{}, e
	}
	c.Runner = r
	if e = writeJSON(filepath.Join(s.dir, "config.json"), c); e != nil {
		return State{}, e
	}
	events, _, e := s.read(context.Background(), c, false)
	return s.state(c, events), e
}
func (s *Store) git(ctx context.Context, in []byte, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	base := []string{"-c", "core.hooksPath=" + filepath.Join(s.dir, "disabled-hooks"), "-c", "protocol.allow=never", "-c", "protocol.ext.allow=never", "-c", "protocol.https.allow=always", "-c", "protocol.ssh.allow=always", "-c", "protocol.file.allow=always", "--git-dir=" + filepath.Join(s.dir, "store.git")}
	cmd := exec.CommandContext(ctx, "git", append(base, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_AUTHOR_NAME=Channel", "GIT_AUTHOR_EMAIL=channel@localhost", "GIT_COMMITTER_NAME=Channel", "GIT_COMMITTER_EMAIL=channel@localhost")
	cmd.Stdin = bytes.NewReader(in)
	var out limitedBuffer
	if len(args) > 1 && args[0] == "cat-file" && args[1] == "--batch" {
		out.limit = 40 * 1024 * 1024
	}
	cmd.Stdout = &out
	cmd.Stderr = &out
	e := cmd.Run()
	if e != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], e, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}
func branch(c diskConfig) string {
	h := sha256.Sum256([]byte(c.Channel))
	return "refs/heads/stackcord-channel-" + hex.EncodeToString(h[:12])
}
func (s *Store) read(ctx context.Context, c diskConfig, sync bool) ([]Event, string, error) {
	pinPath := filepath.Join(s.dir, "head.json")
	var pin string
	if b, e := os.ReadFile(pinPath); e == nil {
		if e = json.Unmarshal(b, &pin); e != nil {
			return nil, "", e
		}
	}
	head := pin
	if sync {
		out, e := s.git(ctx, nil, "ls-remote", "--heads", c.Remote, branch(c))
		if e != nil {
			return nil, "", e
		}
		if out == "" {
			if pin != "" {
				return nil, "", errors.New("channel history deleted")
			}
			return nil, "", nil
		}
		head = strings.Fields(out)[0]
		if _, e = s.git(ctx, nil, "fetch", "--no-tags", c.Remote, branch(c)); e != nil {
			return nil, "", e
		}
		f, e := s.git(ctx, nil, "rev-parse", "FETCH_HEAD")
		if e != nil {
			return nil, "", e
		}
		head = f
		if pin != "" {
			if _, e = s.git(ctx, nil, "merge-base", "--is-ancestor", pin, head); e != nil {
				return nil, "", errors.New("channel history rollback or rewrite")
			}
		}
	}
	if head == "" {
		return nil, "", nil
	}
	events, e := s.readEvents(ctx, c, head)
	if e != nil {
		return nil, "", e
	}
	if sync && head != pin {
		if e = writeJSON(pinPath, head); e != nil {
			return nil, "", e
		}
	}
	return events, head, nil
}
func signingBytes(e Event) []byte { e.Signature = ""; b, _ := json.Marshal(e); return b }
func validate(c diskConfig, events []Event, e Event, signature bool) error {
	if !identifier.MatchString(e.ID) || e.Channel != c.Channel || !identifier.MatchString(e.Author) || len(signingBytes(e)) > maxPayload-256 {
		return errors.New("invalid event identity or size")
	}
	pub, err := base64.StdEncoding.DecodeString(c.Peers[e.Author])
	if err != nil || len(pub) != 32 {
		return errors.New("untrusted author")
	}
	if signature {
		sig, err := base64.StdEncoding.DecodeString(e.Signature)
		if err != nil || !ed25519.Verify(pub, signingBytes(e), sig) {
			return errors.New("invalid event signature")
		}
	}
	reqs := map[string]Event{}
	results := map[string]string{}
	for _, old := range events {
		if old.ID == e.ID {
			return errors.New("replayed event")
		}
		if old.Type == "request" {
			reqs[old.ID] = old
		} else {
			results[old.RequestID] = old.Status
		}
	}
	if _, err = time.Parse(time.RFC3339Nano, e.CreatedAt); err != nil {
		return errors.New("invalid timestamp")
	}
	switch e.Type {
	case "request":
		if !identifier.MatchString(e.Kind) || !validKey(c.Peers[e.To]) || strings.TrimSpace(e.Title) == "" || e.RequestID != "" || e.Status != "" || len(e.Dependencies) > 32 || len(e.Scope) > 32 {
			return errors.New("invalid request")
		}
		seen := map[string]bool{}
		for _, d := range e.Dependencies {
			if _, ok := reqs[d]; !ok || seen[d] || d == e.ID {
				return errors.New("invalid dependency")
			}
			seen[d] = true
		}
	case "result":
		r, ok := reqs[e.RequestID]
		if e.Status == "success" {
			for _, d := range r.Dependencies {
				if results[d] != "success" {
					return errors.New("request dependencies are not successful")
				}
			}
		}
		if !ok || r.To != e.Author || results[e.RequestID] != "" || (e.Status != "success" && e.Status != "failed") || e.To != "" || e.Kind != "" || e.Title != "" || len(e.Dependencies) > 0 || len(e.Scope) > 0 {
			return errors.New("invalid or duplicate result")
		}
	default:
		return errors.New("unknown event type")
	}
	return nil
}
func (s *Store) state(c diskConfig, events []Event) State {
	st := State{Configured: true, Channel: c.Channel, Remote: c.Remote, Peer: c.Peer, PublicKey: c.Peers[c.Peer], Peers: c.Peers, Runner: RunnerState{ResultFormat: c.Runner.ResultFormat, Enabled: len(c.Runner.Argv) > 0, AllowedKinds: c.Runner.AllowedKinds, TimeoutSeconds: c.Runner.TimeoutSeconds}, Requests: []RequestState{}}
	st.Revision = s.revision(c)
	st.WorkerBlockedReason = s.quarantineReason()
	results := map[string]Event{}
	for _, e := range events {
		if e.Type == "result" {
			results[e.RequestID] = e
		}
	}
	for _, e := range events {
		if e.Type != "request" {
			continue
		}
		r := RequestState{Request: e, Status: "ready"}
		if res, ok := results[e.ID]; ok {
			r.Status = res.Status
			r.Result = &res
		} else {
			for _, d := range e.Dependencies {
				res, ok := results[d]
				if !ok || res.Status != "success" {
					r.Status = "blocked"
					r.BlockedReason = "dependency " + d + " has no successful result"
					break
				}
			}
			if _, err := os.Stat(filepath.Join(s.dir, "receipt-"+e.ID+".json")); err == nil {
				r.Status = "interrupted"
				r.BlockedReason = "durable execution receipt exists; explicit retry required"
				s.receiptState(&r)

			}
		}
		if r.Status == "ready" && r.Request.To == c.Peer && st.WorkerBlockedReason != "" {
			r.Status = "blocked"
			r.BlockedReason = st.WorkerBlockedReason
		}
		st.Requests = append(st.Requests, r)
	}
	return st
}
func (s *Store) State(ctx context.Context, sync bool) (State, error) {
	if _, e := s.load(); errors.Is(e, ErrNotConfigured) {
		return State{Requests: []RequestState{}, Revision: "unconfigured"}, nil
	} else if e != nil {
		return State{}, e
	}
	u, e := s.lock()
	if e != nil {
		return State{}, e
	}
	defer u()
	c, e := s.load()
	if errors.Is(e, ErrNotConfigured) {
		return State{Requests: []RequestState{}, Revision: "unconfigured"}, nil
	}
	if e != nil {
		return State{}, e
	}
	events, _, e := s.read(ctx, c, sync)
	if e != nil {
		return State{}, e
	}
	return s.state(c, events), nil
}
func (s *Store) append(ctx context.Context, c diskConfig, ev Event) (Event, error) {
	key, e := base64.StdEncoding.DecodeString(c.PrivateKey)
	if e != nil || len(key) != 64 {
		return Event{}, errors.New("invalid local private key")
	}
	ev.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, signingBytes(ev)))
	for attempt := 0; attempt < 5; attempt++ {
		events, head, e := s.read(ctx, c, true)
		if e != nil {
			return Event{}, e
		}
		for _, old := range events {
			if old.ID == ev.ID {
				a, _ := json.Marshal(old)
				b, _ := json.Marshal(ev)
				if bytes.Equal(a, b) {
					return old, nil
				}
			}
		}
		if len(events) >= maxEvents {
			return Event{}, errors.New("channel history limit exceeded")
		}
		if e = validate(c, events, ev, true); e != nil {
			return Event{}, e
		}
		b, _ := json.Marshal(ev)
		blob, e := s.git(ctx, b, "hash-object", "-w", "--stdin")
		if e != nil {
			return Event{}, e
		}
		tree, e := s.git(ctx, []byte("100644 blob "+blob+"\tevent.json\n"), "mktree")
		if e != nil {
			return Event{}, e
		}
		args := []string{"commit-tree", tree, "-m", "channel event"}
		if head != "" {
			args = append(args, "-p", head)
		}
		commit, e := s.git(ctx, nil, args...)
		if e != nil {
			return Event{}, e
		}
		if _, e = s.git(ctx, nil, "push", c.Remote, commit+":"+branch(c)); e == nil {
			if e = writeJSON(filepath.Join(s.dir, "head.json"), commit); e != nil {
				return Event{}, e
			}
			return ev, nil
		}
	}
	return Event{}, errors.New("concurrent channel append failed after retries")
}
func newEvent(c diskConfig, kind string) Event {
	b := make([]byte, 16)
	rand.Read(b)
	return Event{ID: hex.EncodeToString(b), Channel: c.Channel, Author: c.Peer, Type: kind, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}
func (s *Store) Send(ctx context.Context, r RequestInput) (Event, error) {
	u, e := s.lock()
	if e != nil {
		return Event{}, e
	}
	defer u()
	c, e := s.load()
	if e != nil {
		return Event{}, e
	}
	if e = s.checkRevision(c); e != nil {
		return Event{}, e
	}
	ev := newEvent(c, "request")
	ev.To = r.To
	ev.Kind = r.Kind
	ev.Title = r.Title
	ev.Body = r.Body
	ev.Dependencies = r.Dependencies
	ev.Scope = r.Scope
	return s.append(ctx, c, ev)
}
func (s *Store) Respond(ctx context.Context, r ResultInput) (Event, error) {
	u, e := s.lock()
	if e != nil {
		return Event{}, e
	}
	defer u()
	c, e := s.load()
	if e != nil {
		return Event{}, e
	}
	if e = s.checkRevision(c); e != nil {
		return Event{}, e
	}
	ev := newEvent(c, "result")
	ev.RequestID = r.RequestID
	ev.Status = r.Status
	ev.Body = r.Body
	return s.append(ctx, c, ev)
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	limit := b.limit
	if limit == 0 {
		limit = 1024 * 1024
	}
	if b.Len() < limit {
		left := limit - b.Len()
		if len(p) > left {
			p = p[:left]
		}
		b.Buffer.Write(p)
	}
	return n, nil
}

func (s *Store) revision(c diskConfig) string {
	b, _ := json.Marshal(c)
	h := sha256.New()
	h.Write(b)
	head, _ := os.ReadFile(filepath.Join(s.dir, "head.json"))
	h.Write(head)
	files, _ := filepath.Glob(filepath.Join(s.dir, "receipt-*.json"))
	for _, p := range files {
		b, _ := os.ReadFile(p)
		h.Write([]byte(filepath.Base(p)))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Store) checkRevision(c diskConfig) error {
	if s.ExpectedRevision != "" && s.ExpectedRevision != s.revision(c) {
		return errors.New("channel configuration or state changed; refresh before applying")
	}
	return nil
}

func (s *Store) workerLock() (func(), error) {
	if e := safePath(s.dir); e != nil {
		return nil, e
	}
	if e := os.MkdirAll(s.dir, 0700); e != nil {
		return nil, e
	}
	p := filepath.Join(s.dir, "worker.lock")
	if e := safePath(p); e != nil {
		return nil, e
	}
	return fileLock(p)
}
