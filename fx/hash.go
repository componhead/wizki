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
	"wizki/i18n"
)

// ErrDuplicate: la fonte ha lo stesso contenuto di un originale già in raw/.
var ErrDuplicate error = i18n.NewErr(func(d *i18n.Dict) string { return d.ErrDuplicate })

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
// Gli hash restano in una cache locale (vedi hashCachePath): un file con la
// stessa dimensione e data di modifica non si rilegge, il che conta quando
// raw/ è su una condivisione di rete. La cache è solo un'ottimizzazione: se
// non si legge o non si scrive, si ricalcola tutto.
func HashRaw(root string) (map[string]string, error) {
	cachePath := hashCachePath(root)
	cache := map[string]core.HashEntry{}
	if cachePath != "" {
		if b, err := os.ReadFile(cachePath); err == nil {
			cache = core.ParseHashCache(string(b))
		}
	}
	fresh := map[string]core.HashEntry{}
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
			fi, err := e.Info()
			if err != nil {
				return nil, err
			}
			key := filepath.Join(d, e.Name())
			sum, ok := core.CachedSum(cache, key, fi.Size(), fi.ModTime().UnixNano())
			if !ok {
				if sum, err = HashFile(p); err != nil {
					return nil, err
				}
			}
			fresh[key] = core.HashEntry{Path: key, Size: fi.Size(), MTime: fi.ModTime().UnixNano(), Sum: sum}
			out[p] = sum
		}
	}
	if cachePath != "" {
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
			_ = WriteGenerated(cachePath, core.FormatHashCache(fresh))
		}
	}
	return out, nil
}
