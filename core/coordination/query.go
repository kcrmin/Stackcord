package coordination

import (
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

func (s *Store) Projects() ([]string, error) {
	ids := []string{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("projects")).ForEach(func(k, v []byte) error {
			if len(ids) >= 1000 {
				return fmt.Errorf("%w: too many projects for one runtime", ErrInvalid)
			}
			ids = append(ids, string(k))
			return nil
		})
	})
	return ids, err
}
func (s *Store) Tasks(id string) ([]Task, error) {
	if !identifier.MatchString(id) {
		return nil, ErrInvalid
	}
	tasks := []Task{}
	total := 0
	err := s.db.View(func(tx *bolt.Tx) error {
		p := project(tx, id)
		if p == nil {
			return nil
		}
		return p.Bucket([]byte("tasks")).ForEach(func(k, v []byte) error {
			total += len(v)
			if len(tasks) >= 1000 || total > 4<<20 {
				return fmt.Errorf("%w: task listing too large; use task context or events", ErrInvalid)
			}
			var t Task
			if e := json.Unmarshal(v, &t); e != nil {
				return e
			}
			tasks = append(tasks, t)
			return nil
		})
	})
	return tasks, err
}

// Context returns the latest checkpoint and direct dependency results, not a
// transcript. A caller must explicitly resolve referenced artifacts as needed.
func (s *Store) Context(id, task string, maxBytes int) (Context, error) {
	var result Context
	if !identifier.MatchString(id) || !identifier.MatchString(task) || maxBytes < 1 || maxBytes > MaxContextBytes {
		return result, ErrInvalid
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		p := project(tx, id)
		t, e := readTask(p, task)
		if e != nil {
			return e
		}
		result.Task = t
		result.Dependencies = []Dependency{}
		for _, id := range t.Definition.Dependencies {
			dep, e := readTask(p, id)
			if e != nil {
				return e
			}
			d := Dependency{ID: id, State: dep.State, Revision: dep.Revision}
			if dep.Checkpoint != nil {
				d.Artifacts = dep.Checkpoint.Artifacts
			}
			result.Dependencies = append(result.Dependencies, d)
		}
		raw, e := json.Marshal(result)
		if e != nil {
			return e
		}
		if len(raw) > maxBytes {
			return fmt.Errorf("%w: need %d bytes, have %d", ErrBudget, len(raw), maxBytes)
		}
		return nil
	})
	if err != nil {
		return Context{}, err
	}
	return result, nil
}

// Events uses a project-local cursor. Advance only after processing the page.
// Nothing acknowledges or deletes events; a caller can replay the same cursor.
func (s *Store) Events(id string, after uint64, limit int) (Page, error) {
	page := Page{Events: []Event{}, Next: after}
	if !identifier.MatchString(id) || limit < 1 || limit > 100 {
		return page, ErrInvalid
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		p := project(tx, id)
		if p == nil {
			if after != 0 {
				return ErrInvalid
			}
			return nil
		}
		b := p.Bucket([]byte("events"))
		if after > b.Sequence() {
			return fmt.Errorf("%w: cursor is ahead of this database", ErrInvalid)
		}
		c := b.Cursor()
		for k, v := c.Seek(sequence(after + 1)); k != nil; k, v = c.Next() {
			if len(page.Events) == limit {
				page.More = true
				break
			}
			var ev Event
			if e := json.Unmarshal(v, &ev); e != nil {
				return e
			}
			page.Events = append(page.Events, ev)
			page.Next = ev.Sequence
		}
		return nil
	})
	return page, err
}
