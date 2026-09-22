package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kcrmin/Stackcord/core/coordination"
)

type Client struct {
	base, token string
	http        *http.Client
}

// NewClient accepts HTTPS, or loopback HTTP (including an SSH tunnel). It never
// follows redirects, to avoid forwarding credentials to another endpoint.
func NewClient(endpoint, token string) (*Client, error) {
	u, e := url.Parse(endpoint)
	if e != nil {
		return nil, e
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Host == "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("invalid runtime endpoint")
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return nil, fmt.Errorf("use HTTPS or loopback HTTP")
		}
	}
	if strings.TrimSpace(token) == "" || strings.ContainsAny(token, "\r\n") {
		return nil, fmt.Errorf("runtime credential required")
	}
	return &Client{base: strings.TrimRight(endpoint, "/"), token: token, http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) request(ctx context.Context, path string, input any) (*http.Response, error) {
	var body io.Reader
	method := "GET"
	if input != nil {
		raw, e := json.Marshal(input)
		if e != nil {
			return nil, e
		}
		if len(raw) > coordination.MaxMutationBytes {
			return nil, coordination.ErrInvalid
		}
		body = bytes.NewReader(raw)
		method = "POST"
	}
	req, e := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if e != nil {
		return nil, e
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, e := c.http.Do(req)
	if e != nil {
		return nil, e
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		var problem struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&problem)
		if problem.Error == "" {
			problem.Error = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("runtime HTTP %d: %s", resp.StatusCode, problem.Error)
	}
	return resp, nil
}
func (c *Client) json(ctx context.Context, path string, input, result any) error {
	resp, e := c.request(ctx, path, input)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if e != nil {
		return e
	}
	if len(raw) > 8<<20 {
		return fmt.Errorf("runtime response exceeds 8 MiB")
	}
	return json.Unmarshal(raw, result)
}
func query(project string) url.Values { return url.Values{"project": {project}} }
func (c *Client) Apply(ctx context.Context, m coordination.Mutation) (coordination.Task, error) {
	var task coordination.Task
	e := c.json(ctx, "/v1/apply", m, &task)
	return task, e
}
func (c *Client) Projects(ctx context.Context) ([]string, error) {
	var ids []string
	e := c.json(ctx, "/v1/projects", nil, &ids)
	return ids, e
}
func (c *Client) Tasks(ctx context.Context, project string) ([]coordination.Task, error) {
	var tasks []coordination.Task
	e := c.json(ctx, "/v1/tasks?"+query(project).Encode(), nil, &tasks)
	return tasks, e
}
func (c *Client) Context(ctx context.Context, project, task string, maxBytes int) (coordination.Context, error) {
	var result coordination.Context
	q := query(project)
	q.Set("task", task)
	q.Set("max_bytes", strconv.Itoa(maxBytes))
	e := c.json(ctx, "/v1/context?"+q.Encode(), nil, &result)
	return result, e
}
func (c *Client) Events(ctx context.Context, project string, after uint64, limit int) (coordination.Page, error) {
	var page coordination.Page
	q := query(project)
	q.Set("after", strconv.FormatUint(after, 10))
	q.Set("limit", strconv.Itoa(limit))
	e := c.json(ctx, "/v1/events?"+q.Encode(), nil, &page)
	return page, e
}
func (c *Client) Backup(ctx context.Context, w io.Writer) error {
	resp, e := c.request(ctx, "/v1/backup", nil)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	_, e = io.Copy(w, resp.Body)
	return e
}
