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

// ErrDuplicate: il PDF ha lo stesso contenuto di uno già presente in raw/pdf/.
var ErrDuplicate = errors.New("PDF già acquisito")

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

// HashPDFs calcola lo SHA-256 di ogni *.pdf in dir (path -> hash).
// Una directory inesistente equivale a nessun PDF.
func HashPDFs(dir string) (map[string]string, error) {
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !core.IsPDF(e.Name()) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		sum, err := HashFile(p)
		if err != nil {
			return nil, err
		}
		out[p] = sum
	}
	return out, nil
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
