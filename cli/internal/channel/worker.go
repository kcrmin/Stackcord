package channel

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type receipt struct {
	RequestID         string `json:"request_id"`
	Started           bool   `json:"started"`
	InterruptedReason string `json:"interrupted_reason,omitempty"`
	Result            *Event `json:"result,omitempty"`
}

func (s *Store) WorkOnce(ctx context.Context) (State, error) {
	wu, e := s.workerLock()
	if e != nil {
		return State{}, e
	}
	defer wu()
	u, e := s.lock()
	if e != nil {
		return State{}, e
	}
	defer func() {
		if u != nil {
			u()
		}
	}()
	c, e := s.load()
	if e != nil {
		return State{}, e
	}
	if e = s.checkRevision(c); e != nil {
		return State{}, e
	}
	if len(c.Runner.Argv) == 0 {
		return State{}, errors.New("local runner is not configured")
	}
	events, _, e := s.read(ctx, c, true)
	if e != nil {
		return State{}, e
	}
	st := s.state(c, events)
	for _, r := range st.Requests {
		if r.Request.To != c.Peer || r.Result != nil {
			continue
		}
		p := filepath.Join(s.dir, "receipt-"+r.Request.ID+".json")
		var rec receipt
		if data, err := os.ReadFile(p); err == nil {
			if err = json.Unmarshal(data, &rec); err != nil {
				return State{}, err
			}
			if rec.Result != nil {
				if _, err = s.append(ctx, c, *rec.Result); err != nil {
					return State{}, err
				}
				events, _, err = s.read(ctx, c, false)
				return s.state(c, events), err
			}
			continue
		}
		if r.Status != "ready" {
			continue
		}
		allowed := false
		for _, kind := range c.Runner.AllowedKinds {
			if kind == r.Request.Kind {
				allowed = true
			}
		}
		if !allowed {
			continue
		}
		activeRelease, err := fileLock(filepath.Join(s.dir, "active-"+r.Request.ID+".lock"))
		if err != nil {
			return State{}, err
		}
		activeHeld := true
		activeUnlock := func() {
			if activeHeld {
				activeHeld = false
				activeRelease()
			}
		}
		defer activeUnlock()
		rec = receipt{RequestID: r.Request.ID, Started: true}
		if e = writeJSON(p, rec); e != nil {
			return State{}, e
		}
		u()
		u = nil
		resultFormat := c.Runner.ResultFormat
		timeout := c.Runner.TimeoutSeconds
		if timeout == 0 {
			timeout = 300
		}
		runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		cmd := exec.CommandContext(runCtx, c.Runner.Argv[0], c.Runner.Argv[1:]...)
		cmd.Dir = s.root
		input, _ := json.Marshal(r.Request)
		cmd.Stdin = strings.NewReader(string(input))
		var output limitedBuffer
		cmd.Stdout = &output
		cmd.Stderr = nil
		cmd.WaitDelay = 2 * time.Second
		err = cmd.Run()
		interrupted := runCtx.Err() != nil || errors.Is(err, exec.ErrWaitDelay)
		cancel()
		u, e = s.lock()
		if e != nil {
			return State{}, e
		}
		c, e = s.load()
		if e != nil {
			return State{}, e
		}
		if interrupted {
			rec.InterruptedReason = "runner timeout or cancellation may have left child processes; stop remaining processes before explicit retry"
			if e = writeJSON(p, rec); e != nil {
				return State{}, e
			}
			activeUnlock()
			return s.state(c, events), nil
		}
		result := newEvent(c, "result")
		result.RequestID = r.Request.ID
		result.Status = "success"
		if err != nil {
			result.Status = "failed"
		}
		body := output.String()
		if len(body) > 4000 {
			body = body[:4000] + "\n[output truncated]"
		}
		if err != nil {
			body += "\nrunner failed or timed out"
		}
		if resultFormat == "json" {
			status, responseBody, parseErr := parseResult(output.String())
			if parseErr != nil {
				result.Status = "failed"
				body = "runner returned an invalid structured result"
			} else {
				body = responseBody
				if err == nil {
					result.Status = status
				}
			}
		}
		result.Body = body
		rec.Result = &result
		if e = writeJSON(p, rec); e != nil {
			return State{}, e
		}
		if _, e = s.append(ctx, c, result); e != nil {
			return State{}, e
		}
		events, _, e = s.read(ctx, c, false)
		return s.state(c, events), e
	}
	return st, nil
}
func (s *Store) Retry(ctx context.Context, id string) (State, error) {
	if !identifier.MatchString(id) {
		return State{}, errors.New("invalid request identifier")
	}
	wu, e := s.workerLock()
	if e != nil {
		return State{}, e
	}
	defer wu()
	u, e := s.lock()
	if e != nil {
		return State{}, e
	}
	defer func() {
		if u != nil {
			u()
		}
	}()
	c, e := s.load()
	if e != nil {
		return State{}, e
	}
	if e = s.checkRevision(c); e != nil {
		return State{}, e
	}
	events, _, e := s.read(ctx, c, true)
	if e != nil {
		return State{}, e
	}
	st := s.state(c, events)
	for _, r := range st.Requests {
		if r.Request.ID == id && r.Request.To == c.Peer {
			p := filepath.Join(s.dir, "receipt-"+r.Request.ID+".json")
			data, e := os.ReadFile(p)
			if e != nil {
				return State{}, e
			}
			var rec receipt
			if e = json.Unmarshal(data, &rec); e != nil {
				return State{}, e
			}
			if rec.Result != nil {
				return State{}, errors.New("completed result awaits publication; run worker to publish without rerunning")
			}
			if e = os.Remove(p); e != nil {
				return State{}, e
			}
			return s.state(c, events), nil
		}
	}
	return State{}, errors.New("no interrupted local request to retry")
}

func (s *Store) receiptState(r *RequestState) {
	p := filepath.Join(s.dir, "receipt-"+r.Request.ID+".json")
	data, e := os.ReadFile(p)
	if e != nil {
		return
	}
	var rec receipt
	if json.Unmarshal(data, &rec) != nil {
		return
	}
	if rec.Result != nil {
		r.Status = "awaiting_publication"
		r.BlockedReason = "completed local result awaits publication; run worker to publish"
		return
	}
	if rec.InterruptedReason != "" {
		r.BlockedReason = rec.InterruptedReason
	} else {
		r.BlockedReason = "interrupted execution may have surviving processes; stop them before explicit retry"
	}
	active := filepath.Join(s.dir, "active-"+r.Request.ID+".lock")
	if _, e = os.Stat(active); e == nil {
		u, e := fileLock(active)
		if e != nil {
			r.Status = "running"
			r.BlockedReason = "foreground worker is executing this request"
		} else {
			u()
		}
	}
}

func parseResult(raw string) (string, string, error) {
	dec := json.NewDecoder(strings.NewReader(raw))
	token, e := dec.Token()
	if e != nil || token != json.Delim('{') {
		return "", "", errors.New("expected result object")
	}
	fields := map[string]string{}
	for dec.More() {
		token, e = dec.Token()
		if e != nil {
			return "", "", e
		}
		key, ok := token.(string)
		if !ok || (key != "status" && key != "body") {
			return "", "", errors.New("unknown result field")
		}
		if _, exists := fields[key]; exists {
			return "", "", errors.New("duplicate result field")
		}
		var value any
		if e = dec.Decode(&value); e != nil {
			return "", "", e
		}
		text, ok := value.(string)
		if !ok {
			return "", "", errors.New("result values must be strings")
		}
		fields[key] = text
	}
	if _, e = dec.Token(); e != nil {
		return "", "", e
	}
	var trailing any
	if e = dec.Decode(&trailing); e != io.EOF {
		return "", "", errors.New("trailing result content")
	}
	if len(fields) != 2 || (fields["status"] != "success" && fields["status"] != "failed") || len(fields["body"]) > 4000 {
		return "", "", errors.New("invalid result status or body")
	}
	return fields["status"], fields["body"], nil
}

// A started receipt without a completed local result is a quarantine, including
// after process crashes. Publishing other already-completed receipts is safe;
// starting any new executable is not safe until explicit cleanup acknowledgement.
func (s *Store) quarantineReason() string {
	paths, e := filepath.Glob(filepath.Join(s.dir, "receipt-*.json"))
	if e != nil {
		return "cannot inspect execution receipts; automatic execution is quarantined"
	}
	for _, p := range paths {
		data, e := os.ReadFile(p)
		if e != nil {
			return "cannot read execution receipt; automatic execution is quarantined"
		}
		var rec receipt
		if json.Unmarshal(data, &rec) != nil {
			return "invalid execution receipt; automatic execution is quarantined"
		}
		if rec.Started && rec.Result == nil {
			return "unfinished execution " + rec.RequestID + ": wait while running; after interruption stop remaining processes, then explicitly retry to acknowledge cleanup"
		}
	}
	return ""
}
