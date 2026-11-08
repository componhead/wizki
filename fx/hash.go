package fx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"wizki/core"
)

// ErrDuplicate: la fonte ha lo stesso contenuto di un originale già in raw/.
var ErrDuplicate = errors.New("fonte già acquisita")

// HashFile restituisce lo SHA-256 esadecimale del contenuto del file.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SourceIDs legge il campo id: del frontmatter di ogni nota in <root>/sources/
// (path relativo al notebook -> id, "" se manca).
func SourceIDs(root string) (map[string]string, error) {
	files, err := filepath.Glob(filepath.Join(root, "sources", "*.md"))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		out[filepath.Join("sources", filepath.Base(p))] = core.FrontmatterField(string(b), "id")
	}
	return out, nil
}

// HashRaw calcola lo SHA-256 degli originali di ogni tipo (path -> hash): i
// file al primo livello delle cartelle core.RawDirs sotto root.
func HashRaw(root string) (map[string]string, error) {
	out := map[string]string{}
	for _, d := range core.RawDirs() {
		dir := filepath.Join(root, d)
		entries, err := os.ReadDir(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if _, ok := core.SourceKind(e.Name()); e.IsDir() || !ok {
				continue
			}
			p := filepath.Join(dir, e.Name())
			sum, err := HashFile(p)
			if err != nil {
				return nil, err
			}
			out[p] = sum
		}
	}
	return out, nil
}
