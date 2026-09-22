package coordination

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Store owns one local database. Open it once in the runtime and use HTTP from
// other processes. Never put the live database on a shared network filesystem.
type Store struct {
	db  *bolt.DB
	now func() time.Time
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: database path required", ErrInvalid)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(path); err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf("%w: database must be a regular file", ErrInvalid)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		meta, e := tx.CreateBucketIfNotExists([]byte("meta"))
		if e != nil {
			return e
		}
		version := meta.Get([]byte("schema"))
		if version != nil && !bytes.Equal(version, []byte("1")) {
			return fmt.Errorf("unsupported database schema")
		}
		if e = meta.Put([]byte("schema"), []byte("1")); e != nil {
			return e
		}
		_, e = tx.CreateBucketIfNotExists([]byte("projects"))
		return e
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, now: time.Now}, nil
}
func (s *Store) Close() error { return s.db.Close() }

// Backup writes a transactionally consistent database. Restore with the runtime
// stopped; preserve a backup of the previous database before replacing it.
func (s *Store) Backup(w io.Writer) error {
	return s.db.View(func(tx *bolt.Tx) error { _, err := tx.WriteTo(w); return err })
}

type receipt struct {
	Hash   [32]byte `json:"hash"`
	Result Task     `json:"result"`
}

func project(tx *bolt.Tx, id string) *bolt.Bucket {
	return tx.Bucket([]byte("projects")).Bucket([]byte(id))
}
func readTask(p *bolt.Bucket, id string) (Task, error) {
	var task Task
	if p == nil {
		return task, ErrNotFound
	}
	raw := p.Bucket([]byte("tasks")).Get([]byte(id))
	if raw == nil {
		return task, ErrNotFound
	}
	return task, json.Unmarshal(raw, &task)
}
func put(b *bolt.Bucket, key []byte, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return b.Put(key, raw)
}
func sequence(n uint64) []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, n); return b }

func (s *Store) Apply(m Mutation) (Task, error) {
	var result Task
	raw, err := json.Marshal(m)
	if err != nil {
		return result, err
	}
	if err = validateMutation(m, raw); err != nil {
		return result, err
	}
	hash := sha256.Sum256(raw)
	err = s.db.Update(func(tx *bolt.Tx) error {
		p, e := tx.Bucket([]byte("projects")).CreateBucketIfNotExists([]byte(m.Project))
		if e != nil {
			return e
		}
		for _, name := range []string{"tasks", "events", "receipts"} {
			if _, e = p.CreateBucketIfNotExists([]byte(name)); e != nil {
				return e
			}
		}
		receipts := p.Bucket([]byte("receipts"))
		if previous := receipts.Get([]byte(m.ID)); previous != nil {
			var r receipt
			if e = json.Unmarshal(previous, &r); e != nil {
				return e
			}
			if r.Hash != hash {
				return fmt.Errorf("%w: mutation ID was already used", ErrConflict)
			}
			result = r.Result
			return nil
		}
		task, e := readTask(p, m.Task)
		if m.Kind == "create" {
			if e == nil {
				return fmt.Errorf("%w: task already exists", ErrConflict)
			}
			if e != ErrNotFound {
				return e
			}
			for _, id := range m.Definition.Dependencies {
				if _, e = readTask(p, id); e != nil {
					return fmt.Errorf("%w: dependency %s does not exist", ErrInvalid, id)
				}
			}
			task = Task{Project: m.Project, ID: m.Task, Definition: *m.Definition, State: "ready"}
		} else {
			if e != nil {
				return e
			}
			if e = transition(p, &task, m, s.now().UTC()); e != nil {
				return e
			}
		}
		task.Revision++
		task.UpdatedAt = s.now().UTC()
		if m.Checkpoint != nil {
			task.Checkpoint = m.Checkpoint
		}
		if m.Usage != nil {
			if task.Usage == nil {
				task.Usage = &Usage{}
			}
			const max = int64(1<<63 - 1)
			if task.Usage.InputTokens > max-m.Usage.InputTokens || task.Usage.OutputTokens > max-m.Usage.OutputTokens {
				return fmt.Errorf("%w: token count overflow", ErrInvalid)
			}
			task.Usage.InputTokens += m.Usage.InputTokens
			task.Usage.OutputTokens += m.Usage.OutputTokens
		}
		data, e := json.Marshal(task)
		if e != nil {
			return e
		}
		if len(data) > MaxMutationBytes {
			return fmt.Errorf("%w: task exceeds storage limit", ErrInvalid)
		}
		if e = p.Bucket([]byte("tasks")).Put([]byte(task.ID), data); e != nil {
			return e
		}
		events := p.Bucket([]byte("events"))
		n, e := events.NextSequence()
		if e != nil {
			return e
		}
		if e = put(events, sequence(n), Event{Sequence: n, Task: task.ID, Kind: m.Kind, State: task.State, Revision: task.Revision, Owner: task.Owner, Epoch: task.Epoch, At: task.UpdatedAt}); e != nil {
			return e
		}
		if e = put(receipts, []byte(m.ID), receipt{Hash: hash, Result: task}); e != nil {
			return e
		}
		result = task
		return nil
	})
	return result, err
}
