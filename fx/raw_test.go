package fx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"wizki/core"
)

// TestCopyVerify: copia identica con la stessa data di modifica; l'originale
// resta; una destinazione esistente non si sovrascrive.
func TestCopyVerify(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	_ = os.WriteFile(from, []byte("contenuto"), 0o644)
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	_ = os.Chtimes(from, when, when)
	if err := copyVerify(from, to); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(to); string(b) != "contenuto" {
		t.Errorf("copia = %q", b)
	}
	if fi, _ := os.Stat(to); !fi.ModTime().Equal(when) {
		t.Errorf("data = %v", fi.ModTime())
	}
	if _, err := os.Stat(from); err != nil {
		t.Error("l'originale deve restare")
	}
	if err := copyVerify(from, to); !errors.Is(err, ErrExists) {
		t.Errorf("seconda copia: %v", err)
	}
}

// TestCheckRaw: raw locale o assente va bene; un link a una cartella senza
// marcatore o inesistente no.
func TestCheckRaw(t *testing.T) {
	root := t.TempDir()
	if err := CheckRaw(root); err != nil {
		t.Errorf("raw assente: %v", err)
	}
	nas := t.TempDir()
	_ = os.Symlink(nas, filepath.Join(root, "raw"))
	if err := CheckRaw(root); !errors.Is(err, ErrRawUnreachable) {
		t.Errorf("senza marcatore: %v", err)
	}
	_ = os.WriteFile(filepath.Join(nas, core.RawMarker), core.RawMarkerContent("w"), 0o644)
	if err := CheckRaw(root); err != nil {
		t.Errorf("con marcatore: %v", err)
	}
	_ = os.RemoveAll(nas)
	if err := CheckRaw(root); !errors.Is(err, ErrRawUnreachable) {
		t.Errorf("cartella sparita: %v", err)
	}
}

// TestHashRawUsaLaCache: un file con dimensione e data invariate non si
// rilegge (qui: la cache "mente" e l'hash restituito è quello in cache).
func TestHashRawUsaLaCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	root := t.TempDir()
	p := filepath.Join(root, "raw", "pdf", "a.pdf")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte("x"), 0o644)
	first, err := HashRaw(root)
	if err != nil || first[p] == "" {
		t.Fatalf("primo giro: %v %v", first, err)
	}
	fi, _ := os.Stat(p)
	cache := map[string]core.HashEntry{filepath.Join("raw", "pdf", "a.pdf"): {
		Path: filepath.Join("raw", "pdf", "a.pdf"), Size: fi.Size(), MTime: fi.ModTime().UnixNano(), Sum: "dallacache",
	}}
	_ = os.WriteFile(hashCachePath(root), []byte(core.FormatHashCache(cache)), 0o644)
	if second, _ := HashRaw(root); second[p] != "dallacache" {
		t.Errorf("la cache non è stata usata: %v", second)
	}
	later := fi.ModTime().Add(time.Second)
	_ = os.Chtimes(p, later, later)
	if third, _ := HashRaw(root); third[p] != first[p] {
		t.Errorf("file cambiato: atteso ricalcolo, got %v", third)
	}
}
