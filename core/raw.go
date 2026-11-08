package core

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------- raw remoto
//
// raw/ può stare su una condivisione di rete montata (NAS via NFS, SMB, sshfs,
// rclone mount...): <root>/raw diventa un link simbolico a quella cartella,
// così i path "raw/..." scritti nelle note restano validi. Nella cartella
// remota c'è un file marcatore: se manca, la condivisione non è montata (il
// punto di mount vuoto è una cartella locale qualsiasi) e wizki non scrive.

// RawMarker è il file che identifica una cartella raw configurata per wizki.
const RawMarker = ".wizki-raw"

// RawMarkerContent è il contenuto del marcatore per la wiki name.
func RawMarkerContent(name string) []byte {
	return []byte("wizki raw\nwiki: " + name + "\n")
}

// RawMarkerWiki legge il nome della wiki da un marcatore ("" se non c'è).
func RawMarkerWiki(content string) string {
	for l := range strings.SplitSeq(content, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), "wiki:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// CopyVerify copia From in To (che non deve esistere) e controlla che le
// due copie abbiano lo stesso SHA-256; From resta dov'è.
type CopyVerify struct{ From, To string }

// Symlink crea il link simbolico Link che punta a Target.
type Symlink struct{ Target, Link string }

func (CopyVerify) isStep() {}
func (Symlink) isStep()    {}

// RawState descrive <root>/raw e la cartella di destinazione, letti dal disco.
type RawState struct {
	Root   string // radice della wiki
	Target string // cartella remota (assoluta)
	// LinkTo: se <root>/raw è già un link simbolico, dove punta; "" altrimenti.
	LinkTo string
	// Local: i file (path relativi a <root>/raw) della cartella raw locale;
	// nil se non c'è.
	Local []string
	// Same / Conflict: file locali già presenti nella destinazione, con lo
	// stesso contenuto o con un contenuto diverso.
	Same, Conflict map[string]bool
	// MarkerWiki: la wiki scritta nel marcatore della destinazione ("" se manca).
	MarkerWiki string
}

// PlanRaw pianifica lo spostamento di raw/ nella cartella remota st.Target:
// copia verificata di ogni file locale (salvo quelli già presenti uguali),
// poi la cartella locale diventa raw.local-<stamp> (non si cancella nulla),
// <root>/raw diventa un link alla destinazione, e nella destinazione si scrive
// il marcatore. Se raw/ punta già lì, il piano è vuoto.
func PlanRaw(st RawState, wiki, stamp string) ([]Step, error) {
	if !filepath.IsAbs(st.Target) {
		return nil, fmt.Errorf("la cartella remota deve essere un path assoluto: %q", st.Target)
	}
	if st.MarkerWiki != "" && st.MarkerWiki != wiki {
		return nil, fmt.Errorf("%s è già la cartella raw della wiki %q: scegline un'altra", st.Target, st.MarkerWiki)
	}
	link := filepath.Join(st.Root, "raw")
	if st.LinkTo != "" {
		if filepath.Clean(st.LinkTo) == filepath.Clean(st.Target) {
			if st.MarkerWiki == "" {
				return []Step{WriteFile{Path: filepath.Join(st.Target, RawMarker), Data: RawMarkerContent(wiki)}}, nil
			}
			return nil, nil
		}
		return nil, fmt.Errorf("%s punta già a %s: per cambiarla sposta i file e il link a mano", link, st.LinkTo)
	}
	if len(st.Conflict) > 0 {
		var cs []string
		for c := range st.Conflict {
			cs = append(cs, c)
		}
		sort.Strings(cs)
		return nil, fmt.Errorf("nella destinazione ci sono file con lo stesso nome ma contenuto diverso: %s", strings.Join(cs, ", "))
	}
	var steps []Step
	files := append([]string{}, st.Local...)
	sort.Strings(files)
	dirs := map[string]bool{}
	for _, f := range files {
		if st.Same[f] {
			continue
		}
		if d := filepath.Dir(f); d != "." && !dirs[d] {
			dirs[d] = true
			steps = append(steps, Mkdir{filepath.Join(st.Target, d)})
		}
		steps = append(steps, CopyVerify{From: filepath.Join(link, f), To: filepath.Join(st.Target, f)})
	}
	if st.Local != nil {
		steps = append(steps, Move{From: link, To: link + ".local-" + stamp})
	}
	steps = append(steps, Symlink{Target: st.Target, Link: link})
	if st.MarkerWiki == "" {
		steps = append(steps, WriteFile{Path: filepath.Join(st.Target, RawMarker), Data: RawMarkerContent(wiki)})
	}
	return steps, nil
}

// ---------------------------------------------------------- cache hash

// HashEntry è una riga della cache degli hash: un file con dimensione e data
// di modifica; se non cambiano, lo SHA-256 è ancora valido.
type HashEntry struct {
	Path  string
	Size  int64
	MTime int64 // nanosecondi Unix
	Sum   string
}

// ParseHashCache legge la cache (righe "path\tsize\tmtime\tsha"); le righe
// malformate si ignorano: al peggio si ricalcola un hash.
func ParseHashCache(s string) map[string]HashEntry {
	out := map[string]HashEntry{}
	for l := range strings.SplitSeq(s, "\n") {
		f := strings.Split(l, "\t")
		if len(f) != 4 {
			continue
		}
		size, err1 := strconv.ParseInt(f[1], 10, 64)
		mt, err2 := strconv.ParseInt(f[2], 10, 64)
		if err1 != nil || err2 != nil || f[3] == "" {
			continue
		}
		out[f[0]] = HashEntry{Path: f[0], Size: size, MTime: mt, Sum: f[3]}
	}
	return out
}

// FormatHashCache scrive la cache, in ordine di path.
func FormatHashCache(es map[string]HashEntry) string {
	paths := make([]string, 0, len(es))
	for p := range es {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var b strings.Builder
	for _, p := range paths {
		e := es[p]
		fmt.Fprintf(&b, "%s\t%d\t%d\t%s\n", e.Path, e.Size, e.MTime, e.Sum)
	}
	return b.String()
}

// CachedSum restituisce lo hash in cache di path se dimensione e data di
// modifica coincidono.
func CachedSum(cache map[string]HashEntry, path string, size, mtime int64) (string, bool) {
	e, ok := cache[path]
	if !ok || e.Size != size || e.MTime != mtime {
		return "", false
	}
	return e.Sum, true
}
