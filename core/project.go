package core

import (
	"path/filepath"
	"sort"
)

// LibraryDir è la cartella della wiki dentro un progetto creato da `wizki init`:
//
//	<nome>/              progetto: README.md, CLAUDE.md, .claude/commands/, .git
//	<nome>/library/      la wiki: inbox/, raw/, sources/, topics/, … , .zk/
//
// Così lo schema (CLAUDE.md) e i comandi di Claude Code, che parlano di
// library/, valgono uguali in ogni progetto.
const LibraryDir = "library"

// ProjectLayout pianifica i file gestiti di un progetto con radice dir: per
// ciascuno (path relativo a dir -> contenuto), in ordine di path, le cartelle
// che lo contengono e il file. Nessun .gitkeep: le cartelle hanno già un file.
func ProjectLayout(dir string, files map[string][]byte) []Step {
	steps := []Step{Mkdir{dir}}
	seen := map[string]bool{".": true}
	var mkdir func(rel string)
	mkdir = func(rel string) {
		if seen[rel] {
			return
		}
		seen[rel] = true
		mkdir(filepath.Dir(rel))
		steps = append(steps, Mkdir{filepath.Join(dir, rel)})
	}
	for _, rel := range sortedKeys(files) {
		mkdir(filepath.Dir(rel))
		steps = append(steps, WriteFile{Path: filepath.Join(dir, rel), Data: files[rel]})
	}
	return steps
}

// PlanProjectUpgrade porta i file gestiti del progetto dir alla versione
// attuale: crea quelli che mancano e sostituisce, con copia di sicurezza,
// quelli diversi. have ed exists come in PlanUpgrade.
func PlanProjectUpgrade(dir string, files, have map[string][]byte, exists map[string]bool, stamp string) []Step {
	return planManaged(ProjectLayout(dir, files), dir, files, have, exists, stamp, "")
}

// PlanMigrate porta una wiki con il layout precedente (wiki e repository git
// nella stessa cartella root) al layout a progetto: crea root/library e ci
// sposta tutto, tranne il repository git e i file del progetto (projectFiles,
// path relativi a root: resta al suo posto la loro voce di primo livello).
// entries sono i nomi presenti in root.
func PlanMigrate(root string, entries []string, projectFiles map[string][]byte) []Step {
	keep := map[string]bool{".git": true, LibraryDir: true}
	for rel := range projectFiles {
		keep[topLevel(rel)] = true
	}
	lib := filepath.Join(root, LibraryDir)
	steps := []Step{Mkdir{lib}}
	names := append([]string(nil), entries...)
	sort.Strings(names)
	for _, n := range names {
		if !keep[n] {
			steps = append(steps, Move{From: filepath.Join(root, n), To: filepath.Join(lib, n)})
		}
	}
	return steps
}

// topLevel restituisce il primo elemento di un path relativo: ".claude/commands/x.md" -> ".claude".
func topLevel(rel string) string {
	for {
		d := filepath.Dir(rel)
		if d == "." {
			return rel
		}
		rel = d
	}
}

// sortedKeys restituisce le chiavi di m in ordine, per piani deterministici.
func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
