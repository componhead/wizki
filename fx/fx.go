// Package fx raccoglie gli effetti collaterali: filesystem, pdftotext, zk, git.
// È l'unico package che parla col mondo esterno.
package fx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"wizki/core"
)

// ErrNoText: il PDF non ha livello di testo (probabile scansione).
var ErrNoText = errors.New("nessun testo estratto dal PDF")

// ErrExists: il file (o il repository git) da creare c'è già; non lo tocco.
var ErrExists = errors.New("esiste già")

// Exec esegue un singolo passo del piano.
func Exec(ctx context.Context, s core.Step) error {
	switch s := s.(type) {
	case core.Mkdir:
		return os.MkdirAll(s.Path, 0o755)
	case core.Move:
		if _, err := os.Stat(s.To); err == nil {
			return fmt.Errorf("%s esiste già: raw/ è immutabile, non sovrascrivo", s.To)
		}
		return os.Rename(s.From, s.To)
	case core.Extract:
		if out, err := exec.CommandContext(ctx, "pdftotext", "-layout", s.PDF, s.Out).CombinedOutput(); err != nil {
			return fmt.Errorf("pdftotext: %w: %s", err, strings.TrimSpace(string(out)))
		}
		b, err := os.ReadFile(s.Out)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(b)) == "" {
			return ErrNoText
		}
		return nil
	case core.OCR:
		return execOCR(ctx, s)
	case core.WriteFile:
		f, err := os.OpenFile(s.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s", ErrExists, s.Path)
		}
		if err != nil {
			return err
		}
		_, err = f.Write(s.Data)
		return errors.Join(err, f.Close())
	case core.GitInit:
		if _, err := os.Stat(filepath.Join(s.Dir, ".git")); err == nil {
			return fmt.Errorf("%w: %s", ErrExists, filepath.Join(s.Dir, ".git"))
		}
		cmd := exec.CommandContext(ctx, "git", "init", "-q", "-b", "master")
		cmd.Dir = s.Dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git init: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("passo sconosciuto %T", s)
	}
}

// Undo annulla un passo già eseguito (usato quando l'estrazione fallisce per
// mancanza di testo: la scansione deve restare intatta nella inbox).
func Undo(s core.Step) error {
	switch s := s.(type) {
	case core.Move:
		return os.Rename(s.To, s.From)
	case core.Extract:
		return os.Remove(s.Out)
	case core.OCR:
		return os.Remove(s.Out)
	}
	return nil
}

// AppendLog aggiunge una riga a log.md (append-only).
func AppendLog(path, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func Today() time.Time { return time.Now() }

// TopicNames: gli stem dei file in <root>/topics/*.md.
func TopicNames(root string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(root, "topics", "*.md"))
	if err != nil {
		return nil, err
	}
	return core.Map(files, func(p string) string {
		return strings.TrimSuffix(filepath.Base(p), ".md")
	}), nil
}

// ZkList lancia `zk list` nel notebook e restituisce i path (output NUL-separato).
func ZkList(ctx context.Context, root string, args ...string) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	full := append([]string{"list", "--quiet", "--format", "path", "--delimiter0"}, args...)
	cmd := exec.CommandContext(ctx, "zk", full...)
	cmd.Dir = abs
	cmd.Env = append(os.Environ(), "ZK_NOTEBOOK_DIR="+abs)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("zk %s: %w: %s", strings.Join(full, " "), err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("zk: %w", err)
	}
	parts := strings.Split(string(out), "\x00")
	return core.Map(
		core.Filter(parts, func(s string) bool { return strings.TrimSpace(s) != "" }),
		filepath.Clean,
	), nil
}
