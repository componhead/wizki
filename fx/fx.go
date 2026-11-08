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
	case core.ImageText:
		return execImageText(ctx, s)
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
	case core.Replace:
		return replaceWithBackup(s)
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

// Undo annulla un passo già eseguito, quando l'acquisizione fallisce: la fonte
// (e le sue immagini) devono tornare intatte nella inbox. Le cartelle create
// restano.
func Undo(s core.Step) error {
	switch s := s.(type) {
	case core.Move:
		return os.Rename(s.To, s.From)
	case core.Extract:
		return os.Remove(s.Out)
	case core.OCR:
		return os.Remove(s.Out)
	case core.ImageText:
		return os.Remove(s.Out)
	case core.WriteFile:
		// solo dopo il controllo preliminare di extract: il file prima non c'era
		return os.Remove(s.Path)
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

// ReadPages legge le pagine della wiki: la panoramica, se c'è, e i .md di
// sources/, topics/ ed entities/, con path relativi alla radice e in ordine.
func ReadPages(root string) ([]core.Page, error) {
	rels := []string{}
	if _, err := os.Stat(filepath.Join(root, core.OverviewPath)); err == nil {
		rels = append(rels, core.OverviewPath)
	}
	for _, d := range core.PageDirs {
		files, err := filepath.Glob(filepath.Join(root, d, "*.md"))
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			rels = append(rels, filepath.Join(d, filepath.Base(f)))
		}
	}
	pages := make([]core.Page, 0, len(rels))
	for _, rel := range rels {
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return nil, err
		}
		pages = append(pages, core.ParsePage(rel, string(b)))
	}
	return pages, nil
}

// ReadIfExists legge un file; exists è false se il file non c'è.
func ReadIfExists(path string) (content string, exists bool, err error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}

// WriteGenerated sovrascrive un file generato (es. index.md) passando da un
// file temporaneo, così un errore a metà non lascia il file troncato.
func WriteGenerated(path, content string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	_, werr := tmp.WriteString(content)
	if err := errors.Join(werr, tmp.Close(), os.Chmod(tmp.Name(), 0o644)); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// replaceWithBackup copia il file attuale in s.Backup (che non deve esistere),
// poi lo sostituisce con s.Data.
func replaceWithBackup(s core.Replace) error {
	cur, err := os.ReadFile(s.Path)
	if err != nil {
		return err
	}
	if err := Exec(context.Background(), core.WriteFile{Path: s.Backup, Data: cur}); err != nil {
		return fmt.Errorf("copia di sicurezza: %w", err)
	}
	return WriteGenerated(s.Path, string(s.Data))
}

// Exists: il path esiste (file o cartella).
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ResolveAssets cerca, per ogni riferimento a un'immagine, il file locale
// relativo alla cartella della fonte src. Accetta solo immagini esistenti
// dentro quella cartella (o sotto): non sposta mai file da altre parti.
// found: riferimento -> path; missing: i riferimenti senza file.
func ResolveAssets(src string, refs []string) (found map[string]string, missing []string) {
	found = map[string]string{}
	base := filepath.Dir(src)
	for _, r := range refs {
		p := filepath.Join(base, core.RefPath(r))
		rel, err := filepath.Rel(base, p)
		kind, _ := core.SourceKind(p)
		fi, serr := os.Stat(p)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
			kind != core.KindImage || serr != nil || !fi.Mode().IsRegular() {
			missing = append(missing, r)
			continue
		}
		found[r] = p
	}
	return found, missing
}

// InboxSources elenca le fonti acquisibili in dir (solo primo livello),
// nell'ordine di core.InboxOrder: le immagini allegate a un documento della
// inbox non compaiono, perché si acquisiscono con il documento.
func InboxSources(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	attached := map[string]bool{}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		names = append(names, e.Name())
		if k, _ := core.SourceKind(e.Name()); k != core.KindDoc {
			continue
		}
		p := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		found, _ := ResolveAssets(p, core.ImageRefs(core.DocText(p, b)))
		for _, a := range found {
			if filepath.Dir(a) == filepath.Clean(dir) {
				attached[filepath.Base(a)] = true
			}
		}
	}
	return core.InboxOrder(names, attached), nil
}
