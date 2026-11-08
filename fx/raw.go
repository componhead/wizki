package fx

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"wizki/core"
)

// ErrRawUnreachable: raw/ è un link a una cartella remota che ora non si
// raggiunge (condivisione non montata, rete assente).
var ErrRawUnreachable = errors.New("raw remoto non raggiungibile")

// moveFile sposta un file. Tra dischi diversi (per esempio dalla inbox locale
// a raw/ su una condivisione di rete) rename non funziona: copia, verifica lo
// SHA-256 della copia e solo allora cancella l'originale.
func moveFile(from, to string) error {
	err := os.Rename(from, to)
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	if err := copyVerify(from, to); err != nil {
		return err
	}
	return os.Remove(from)
}

// copyVerify copia from in to (che non deve esistere), forza la scrittura sul
// disco, rilegge la copia e la confronta con l'originale. Se non coincidono,
// cancella la copia. Conserva la data di modifica dell'originale.
func copyVerify(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	fi, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fi.Mode().Perm())
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("%w: %s", ErrExists, to)
	}
	if err != nil {
		return err
	}
	h := sha256.New()
	_, cerr := io.Copy(dst, io.TeeReader(src, h))
	if err := errors.Join(cerr, dst.Sync(), dst.Close()); err != nil {
		_ = os.Remove(to)
		return fmt.Errorf("copia di %s: %w", from, err)
	}
	want := hex.EncodeToString(h.Sum(nil))
	got, err := HashFile(to)
	if err != nil || got != want {
		_ = os.Remove(to)
		if err == nil {
			err = errors.New("contenuto diverso dall'originale")
		}
		return fmt.Errorf("verifica della copia %s: %w", to, err)
	}
	_ = os.Chtimes(to, fi.ModTime(), fi.ModTime())
	return nil
}

// CheckRaw controlla, prima di scrivere in raw/, che una raw remota sia
// raggiungibile: se <root>/raw è un link, la destinazione deve esistere ed
// avere il marcatore. Una raw locale (o non ancora creata) va sempre bene.
func CheckRaw(root string) error {
	link := filepath.Join(root, "raw")
	fi, err := os.Lstat(link)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && fi.Mode()&os.ModeSymlink == 0) {
		return nil
	}
	if err != nil {
		return err
	}
	target, _ := os.Readlink(link)
	if st, err := os.Stat(link); err != nil || !st.IsDir() {
		return fmt.Errorf("%w: %s punta a %s, che non si raggiunge: la condivisione è montata?", ErrRawUnreachable, link, target)
	}
	if _, err := os.Stat(filepath.Join(link, core.RawMarker)); err != nil {
		return fmt.Errorf("%w: in %s manca %s: la condivisione è montata? (se è una cartella nuova, configurala con wizki raw)",
			ErrRawUnreachable, target, core.RawMarker)
	}
	return nil
}

// ReadRawState legge lo stato di <root>/raw e della cartella remota target,
// per core.PlanRaw. Confronta per contenuto i file presenti in entrambe.
func ReadRawState(root, target string) (core.RawState, error) {
	st := core.RawState{Root: root, Target: target, Same: map[string]bool{}, Conflict: map[string]bool{}}
	tfi, err := os.Stat(target)
	if err != nil {
		return st, fmt.Errorf("la cartella remota %s non si raggiunge: %w", target, err)
	}
	if !tfi.IsDir() {
		return st, fmt.Errorf("%s non è una cartella", target)
	}
	probe, err := os.CreateTemp(target, ".wizki-prova-*")
	if err != nil {
		return st, fmt.Errorf("non posso scrivere in %s: %w", target, err)
	}
	probe.Close()
	_ = os.Remove(probe.Name())
	if b, err := os.ReadFile(filepath.Join(target, core.RawMarker)); err == nil {
		st.MarkerWiki = core.RawMarkerWiki(string(b))
	}

	link := filepath.Join(root, "raw")
	fi, err := os.Lstat(link)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return st, nil
	case err != nil:
		return st, err
	case fi.Mode()&os.ModeSymlink != 0:
		st.LinkTo, err = os.Readlink(link)
		return st, err
	case !fi.IsDir():
		return st, fmt.Errorf("%s non è una cartella", link)
	}
	st.Local = []string{}
	err = filepath.WalkDir(link, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(link, p)
		if err != nil {
			return err
		}
		st.Local = append(st.Local, rel)
		dst := filepath.Join(target, rel)
		if _, err := os.Stat(dst); err != nil {
			return nil
		}
		a, err := HashFile(p)
		if err != nil {
			return err
		}
		b, err := HashFile(dst)
		if err != nil {
			return err
		}
		if a == b {
			st.Same[rel] = true
		} else {
			st.Conflict[rel] = true
		}
		return nil
	})
	return st, err
}

// hashCachePath: la cache degli hash di una cartella raw, nella cache
// dell'utente (locale anche quando raw è in rete). "" se non c'è una cache.
func hashCachePath(root string) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	raw, err := filepath.EvalSymlinks(filepath.Join(root, "raw"))
	if err != nil {
		return ""
	}
	if abs, err := filepath.Abs(raw); err == nil {
		raw = abs
	}
	sum := sha256.Sum256([]byte(raw))
	return filepath.Join(dir, "wizki", "raw-"+hex.EncodeToString(sum[:])[:16]+".tsv")
}
