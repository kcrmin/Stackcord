package channel

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// readEvents uses three bounded Git processes regardless of history length. The
// metadata pass rejects oversized objects before their content is requested.
func (s *Store) readEvents(ctx context.Context, c diskConfig, head string) ([]Event, error) {
	out, e := s.git(ctx, nil, "rev-list", "--reverse", "--parents", "--max-count=1001", head)
	if e != nil {
		return nil, e
	}
	lines := strings.Split(out, "\n")
	if len(lines) > maxEvents {
		return nil, errors.New("channel history limit exceeded")
	}
	var specs strings.Builder
	previous := ""
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 || len(fields) > 2 || (previous == "" && len(fields) != 1) || (previous != "" && (len(fields) != 2 || fields[1] != previous)) {
			return nil, errors.New("channel requires linear complete history")
		}
		previous = fields[0]
		specs.WriteString(previous + "^{tree}\n" + previous + ":event.json\n")
	}
	check, e := s.git(ctx, []byte(specs.String()), "cat-file", "--batch-check")
	if e != nil {
		return nil, e
	}
	meta := strings.Split(check, "\n")
	if len(meta) != 2*len(lines) {
		return nil, errors.New("invalid channel object metadata")
	}
	sizes := make([]int, len(meta))
	oids := make([]string, len(meta))
	for i, line := range meta {
		f := strings.Fields(line)
		if len(f) != 3 {
			return nil, errors.New("missing channel object")
		}
		n, e := strconv.Atoi(f[2])
		if e != nil || n < 0 || n > maxPayload {
			return nil, errors.New("event exceeds size limit")
		}
		expected := "blob"
		if i%2 == 0 {
			expected = "tree"
			if n != 38 {
				return nil, errors.New("invalid channel tree")
			}
		}
		if f[1] != expected {
			return nil, errors.New("invalid channel object type")
		}
		sizes[i] = n
		oids[i] = f[0]
	}
	raw, e := s.git(ctx, []byte(specs.String()), "cat-file", "--batch")
	if e != nil {
		return nil, e
	}
	data := []byte(raw)
	objects := make([][]byte, len(meta))
	for i, line := range meta {
		newline := bytes.IndexByte(data, '\n')
		if newline < 0 || string(data[:newline]) != line {
			return nil, errors.New("invalid channel batch header")
		}
		data = data[newline+1:]
		if len(data) < sizes[i] {
			return nil, errors.New("truncated channel object")
		}
		objects[i] = data[:sizes[i]]
		data = data[sizes[i]:]
		if len(data) > 0 {
			if data[0] != '\n' {
				return nil, errors.New("invalid channel batch boundary")
			}
			data = data[1:]
		}
	}
	if len(data) != 0 {
		return nil, errors.New("trailing channel object data")
	}
	events := make([]Event, 0, len(lines))
	for i := 0; i < len(objects); i += 2 {
		oid, e := hex.DecodeString(oids[i+1])
		if e != nil || !bytes.Equal(objects[i], append([]byte("100644 event.json\x00"), oid...)) {
			return nil, errors.New("invalid channel tree")
		}
		var ev Event
		if e = json.Unmarshal(objects[i+1], &ev); e != nil {
			return nil, e
		}
		canonical, _ := json.Marshal(ev)
		if !bytes.Equal(canonical, objects[i+1]) {
			return nil, errors.New("noncanonical or trailing event data")
		}
		if e = validate(c, events, ev, true); e != nil {
			return nil, e
		}
		events = append(events, ev)
	}
	return events, nil
}
