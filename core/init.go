package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"wizki/i18n"
)

// WriteFile crea un file nuovo con il contenuto dato; non sovrascrive mai.
type WriteFile struct {
	Path string
	Data []byte
}

func (WriteFile) isStep() {}

// GitInit inizializza un repository git (branch master) in Dir, se non c'è già.
type GitInit struct{ Dir string }

func (GitInit) isStep() {}

// WikiGitignore esclude dal repository della wiki l'indice di zk, che si rigenera.
var WikiGitignore = []byte("/.zk/notebook.db\n")

// WikiDirs sono le cartelle di una wiki, relative alla sua radice.
var WikiDirs = []string{
	"inbox",
	filepath.Join("raw", "pdf"),
	filepath.Join("raw", "doc"),
	filepath.Join("raw", "assets"),
	filepath.Join("raw", "data"),
	filepath.Join("raw", "text"),
	"sources",
	"topics",
	"entities",
	"analyses",
}

// ValidWikiName: il nome di una wiki è anche il nome della sua cartella,
// quindi non può essere vuoto, contenere separatori o essere "." / "..".
func ValidWikiName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return errors.New(i18n.T().WikiNameEmpty)
	case name == "." || name == "..":
		return fmt.Errorf(i18n.T().WikiNameInvalid, name)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf(i18n.T().WikiNameSeparator, name)
	case strings.HasPrefix(name, "-"):
		return fmt.Errorf(i18n.T().WikiNameDash, name)
	}
	return nil
}

// LogHeader è l'intestazione di log.md di una wiki appena creata.
func LogHeader(name string) []byte {
	return []byte("# Log: " + name + "\n\n")
}

// PlanInit pianifica la creazione della wiki <name>/: il layout di Layout e
// infine il repository git.
func PlanInit(name string, zkFiles map[string][]byte) ([]Step, error) {
	if err := ValidWikiName(name); err != nil {
		return nil, err
	}
	return append(Layout(name, name, zkFiles), GitInit{Dir: name}), nil
}

// Layout pianifica i file e le cartelle di una wiki con radice root: le
// cartelle di WikiDirs, i file della configurazione zk (path relativo alla
// radice -> contenuto), log.md (intestato a logName), la panoramica, un
// index.md iniziale e .gitignore. Non valida root: lo usa anche `upgrade`
// su wiki esistenti, la cui radice può essere un path qualsiasi.
// Ogni cartella creata, radice e cartelle intermedie comprese, riceve un
// .gitkeep, una volta sola.
// I file vengono in ordine di path, così il piano è deterministico.
func Layout(root, logName string, zkFiles map[string][]byte) []Step {
	var steps []Step
	seen := map[string]bool{}
	// mkdir crea rel (relativa alla radice) e i suoi genitori, ciascuno con un
	// .gitkeep: git non traccia le cartelle vuote, così sopravvivono a un clone.
	var mkdir func(rel string)
	mkdir = func(rel string) {
		if seen[rel] {
			return
		}
		seen[rel] = true
		if rel != "." {
			mkdir(filepath.Dir(rel))
		}
		dir := filepath.Join(root, rel)
		steps = append(steps, Mkdir{dir}, WriteFile{Path: filepath.Join(dir, ".gitkeep")})
	}
	mkdir(".")
	for _, d := range WikiDirs {
		mkdir(d)
	}
	rels := make([]string, 0, len(zkFiles))
	for rel := range zkFiles {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		mkdir(filepath.Dir(rel))
		steps = append(steps, WriteFile{Path: filepath.Join(root, rel), Data: zkFiles[rel]})
	}
	return append(steps,
		WriteFile{Path: filepath.Join(root, "log.md"), Data: LogHeader(logName)},
		WriteFile{Path: filepath.Join(root, OverviewPath), Data: OverviewPage()},
		WriteFile{Path: filepath.Join(root, IndexPath), Data: []byte(BuildIndex([]Page{ParsePage(OverviewPath, string(OverviewPage()))}))},
		WriteFile{Path: filepath.Join(root, ".gitignore"), Data: WikiGitignore},
	)
}

// ConfigLanguage legge il campo language della sezione [note] di una
// .zk/config.toml: è la lingua della wiki. "" se manca.
func ConfigLanguage(config string) string {
	section := ""
	for l := range strings.SplitSeq(strings.ReplaceAll(config, "\r\n", "\n"), "\n") {
		l = strings.TrimSpace(l)
		if i := strings.Index(l, "#"); i >= 0 && !strings.Contains(l[:i], `"`) {
			l = strings.TrimSpace(l[:i])
		}
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			section = strings.TrimSpace(strings.Trim(l, "[]"))
			continue
		}
		k, v, ok := strings.Cut(l, "=")
		if ok && section == "note" && strings.TrimSpace(k) == "language" {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}
