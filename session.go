package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Decision ops recorded in the log.
const (
	OpSelect = "select"
	OpReject = "reject"
	OpMaybe  = "maybe"
	OpUndo   = "undo"
)

// Entry is one line of the decision log.
type Entry struct {
	T     time.Time `json:"t"`
	Op    string    `json:"op"`
	Path  string    `json:"path,omitempty"`  // photo path relative to the source folder
	Dest  string    `json:"dest,omitempty"`  // where the file was copied/moved, relative to the output folder
	Moved bool      `json:"moved,omitempty"` // true if the original was moved (not copied) to Dest
}

// Session is the append-only decision log stored in <output>/.photo-select/.
// Every swipe appends one JSON line and fsyncs, so progress survives crashes,
// unplugged drives and restarts. Undo appends an "undo" line.
type Session struct {
	f     *os.File
	stack []Entry        // effective decisions, oldest first
	state map[string]int // photo path -> index into stack of its latest decision
}

func stateDir(output string) string { return filepath.Join(output, ".photo-select") }

func OpenSession(output string) (*Session, error) {
	dir := stateDir(output)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "decisions.jsonl")
	s := &Session{}
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			var e Entry
			if json.Unmarshal(sc.Bytes(), &e) != nil {
				continue // a torn last line from a crash; ignore it
			}
			if e.Op == OpUndo {
				if len(s.stack) > 0 {
					s.stack = s.stack[:len(s.stack)-1]
				}
			} else {
				s.stack = append(s.stack, e)
			}
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	s.f = f
	s.rebuild()
	return s, nil
}

func (s *Session) rebuild() {
	s.state = make(map[string]int, len(s.stack))
	for i, e := range s.stack {
		s.state[e.Path] = i
	}
}

func (s *Session) write(e Entry) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := s.f.Write(append(b, '\n')); err != nil {
		return err
	}
	return s.f.Sync()
}

// Push records a decision.
func (s *Session) Push(e Entry) error {
	if e.T.IsZero() {
		e.T = time.Now()
	}
	if err := s.write(e); err != nil {
		return err
	}
	s.stack = append(s.stack, e)
	s.state[e.Path] = len(s.stack) - 1
	return nil
}

// Last returns the most recent decision, if any.
func (s *Session) Last() (Entry, bool) {
	if len(s.stack) == 0 {
		return Entry{}, false
	}
	return s.stack[len(s.stack)-1], true
}

// Undo removes the most recent decision.
func (s *Session) Undo() error {
	if len(s.stack) == 0 {
		return nil
	}
	if err := s.write(Entry{T: time.Now(), Op: OpUndo}); err != nil {
		return err
	}
	s.stack = s.stack[:len(s.stack)-1]
	s.rebuild()
	return nil
}

// Decision returns the current decision for a photo and its order in the log.
func (s *Session) Decision(path string) (Entry, int, bool) {
	i, ok := s.state[path]
	if !ok {
		return Entry{}, 0, false
	}
	return s.stack[i], i, true
}

func (s *Session) Close() error { return s.f.Close() }
