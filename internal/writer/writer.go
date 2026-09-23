// Package writer is the only code in Constelate that changes files. Every
// change is a Plan: a list of operations the user sees as a diff, confirms,
// and that is backed up before it runs. Skill folders are never deleted;
// "uninstall" moves them to ~/.config/constelate/disabled/.
package writer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OpKind is what an operation does.
type OpKind int

const (
	OpWrite   OpKind = iota // write New to Path (Old is the current content, nil when creating)
	OpMoveDir               // rename directory From to To; To must not exist
)

type Op struct {
	Kind OpKind
	// OpWrite
	Path string
	Old  []byte
	New  []byte
	// OpMoveDir
	From string
	To   string
	// Why is one line for the confirmation screen.
	Why string
}

// Plan is a set of operations applied together, or not at all.
type Plan struct {
	Title string
	Ops   []Op
}

// Guard says which directory trees a plan may touch. Anything else is refused
// before any backup or write happens.
type Guard struct {
	Allowed []string // absolute directory prefixes
}

// Result reports what Apply did.
type Result struct {
	DryRun    bool
	BackupDir string   // "" for dry runs or plans that needed no backup
	Done      []string // human lines, one per op
}

var ErrRefused = errors.New("path outside the allowed roots")

// Check refuses a plan that reaches outside the guard or would overwrite a
// move target. Call it before showing the diff so the user never confirms
// something that cannot run.
func (g Guard) Check(p Plan) error {
	for _, op := range p.Ops {
		paths := []string{op.Path}
		if op.Kind == OpMoveDir {
			paths = []string{op.From, op.To}
		}
		for _, x := range paths {
			if x == "" || !g.allowed(x) {
				return fmt.Errorf("%w: %s", ErrRefused, x)
			}
		}
		if op.Kind == OpMoveDir {
			if _, err := os.Stat(op.To); err == nil {
				return fmt.Errorf("refusing to overwrite existing %s", op.To)
			}
			if fi, err := os.Stat(op.From); err != nil || !fi.IsDir() {
				return fmt.Errorf("move source is not a directory: %s", op.From)
			}
		}
	}
	return nil
}

// Allows reports whether p lies inside one of the guarded roots.
func (g Guard) Allows(p string) bool { return g.allowed(p) }

func (g Guard) allowed(p string) bool {
	p = filepath.Clean(p)
	for _, root := range g.Allowed {
		root = filepath.Clean(root)
		if p == root || strings.HasPrefix(p, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// Apply runs the plan. With dryRun it only describes what would happen.
// Backups go under backupRoot/<timestamp>/ with a manifest.json so a move can
// be reversed by hand.
func (g Guard) Apply(p Plan, dryRun bool, backupRoot string) (Result, error) {
	if err := g.Check(p); err != nil {
		return Result{}, err
	}
	res := Result{DryRun: dryRun}
	if dryRun {
		for _, op := range p.Ops {
			res.Done = append(res.Done, "would "+describe(op))
		}
		return res, nil
	}

	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		return res, fmt.Errorf("backup root: %w", err)
	}
	// MkdirTemp keeps two applies in the same second from sharing a directory.
	bdir, err := os.MkdirTemp(backupRoot, time.Now().Format("20060102-150405")+"-*")
	if err != nil {
		return res, fmt.Errorf("backup dir: %w", err)
	}
	manifest := map[string]any{"title": p.Title, "at": time.Now().Format(time.RFC3339), "ops": []map[string]string{}}

	// Back up first, all ops, before changing anything.
	for _, op := range p.Ops {
		entry := map[string]string{"kind": kindName(op.Kind)}
		switch op.Kind {
		case OpWrite:
			entry["path"] = op.Path
			if op.Old != nil {
				dst := filepath.Join(bdir, flatten(op.Path))
				if err := writeFile(dst, op.Old); err != nil {
					return res, fmt.Errorf("backup %s: %w", op.Path, err)
				}
				entry["backup"] = dst
			}
		case OpMoveDir:
			entry["from"], entry["to"] = op.From, op.To
		}
		manifest["ops"] = append(manifest["ops"].([]map[string]string), entry)
	}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	if err := writeFile(filepath.Join(bdir, "manifest.json"), mb); err != nil {
		return res, fmt.Errorf("backup manifest: %w", err)
	}
	res.BackupDir = bdir

	for _, op := range p.Ops {
		switch op.Kind {
		case OpWrite:
			if err := writeFile(op.Path, op.New); err != nil {
				return res, fmt.Errorf("write %s: %w", op.Path, err)
			}
		case OpMoveDir:
			if err := os.MkdirAll(filepath.Dir(op.To), 0o755); err != nil {
				return res, err
			}
			if err := os.Rename(op.From, op.To); err != nil {
				return res, fmt.Errorf("move %s: %w (cross-device moves are not attempted; nothing was deleted)", op.From, err)
			}
		}
		res.Done = append(res.Done, describe(op))
	}
	return res, nil
}

func describe(op Op) string {
	switch op.Kind {
	case OpWrite:
		if op.Old == nil {
			return "create " + op.Path
		}
		return "write " + op.Path
	case OpMoveDir:
		return "move " + op.From + " → " + op.To
	}
	return "?"
}

func kindName(k OpKind) string {
	if k == OpMoveDir {
		return "move"
	}
	return "write"
}

func flatten(p string) string {
	return strings.NewReplacer(string(filepath.Separator), "__", ":", "_").Replace(strings.TrimLeft(p, string(filepath.Separator)))
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// write-then-rename so a crash never leaves a half-written settings file
	tmp, err := os.CreateTemp(filepath.Dir(path), ".constelate-*")
	if err != nil {
		return err
	}
	if _, err := io.Copy(tmp, strings.NewReader(string(b))); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ReadCurrent returns a file's bytes, or nil if it does not exist, for Op.Old.
func ReadCurrent(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}
