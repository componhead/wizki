package core

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
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
	filepath.Join("raw", "text"),
	"sources",
	"topics",
}

// ValidWikiName: il nome di una wiki è anche il nome della sua cartella,
// quindi non può essere vuoto, contenere separatori o essere "." / "..".
func ValidWikiName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("nome della wiki vuoto")
	case name == "." || name == "..":
		return fmt.Errorf("nome della wiki %q non valido", name)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("nome della wiki %q non valido: niente separatori di path", name)
	case strings.HasPrefix(name, "-"):
		return fmt.Errorf("nome della wiki %q non valido: non può iniziare con '-'", name)
	}
	return nil
}

// LogHeader è l'intestazione di log.md di una wiki appena creata.
func LogHeader(name string) []byte {
	return []byte("# Log: " + name + "\n\n")
}

// PlanInit pianifica la creazione della wiki <name>/: le cartelle di WikiDirs
// (ognuna con un .gitkeep),
// i file della configurazione zk (path relativo alla radice -> contenuto),
// log.md, .gitignore e infine il repository git.
// I file vengono in ordine di path, così il piano è deterministico.
func PlanInit(name string, zkFiles map[string][]byte) ([]Step, error) {
	if err := ValidWikiName(name); err != nil {
		return nil, err
	}
	steps := []Step{Mkdir{name}}
	// .gitkeep: git non traccia le cartelle vuote, così sopravvivono a un clone
	for _, d := range WikiDirs {
		dir := filepath.Join(name, d)
		steps = append(steps, Mkdir{dir}, WriteFile{Path: filepath.Join(dir, ".gitkeep")})
	}
	rels := make([]string, 0, len(zkFiles))
	for rel := range zkFiles {
		rels = append(rels, rel)
	}
	sort.Strings(rels)
	for _, rel := range rels {
		p := filepath.Join(name, rel)
		steps = append(steps, Mkdir{filepath.Dir(p)}, WriteFile{Path: p, Data: zkFiles[rel]})
	}
	steps = append(steps,
		WriteFile{Path: filepath.Join(name, "log.md"), Data: LogHeader(name)},
		WriteFile{Path: filepath.Join(name, ".gitignore"), Data: WikiGitignore},
		GitInit{Dir: name},
	)
	return steps, nil
}
