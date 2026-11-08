package core

import (
	"bytes"
	"path/filepath"
)

// Replace sostituisce un file esistente con Data, dopo averne salvato una
// copia in Backup.
type Replace struct {
	Path, Backup string
	Data         []byte
}

func (Replace) isStep() {}

// BackupPath sceglie dove salvare la copia di path: <path>.bak, oppure
// <path>.bak.<stamp> se <path>.bak esiste già (taken).
func BackupPath(path, stamp string, taken map[string]bool) string {
	if b := path + ".bak"; !taken[b] {
		return b
	}
	return path + ".bak." + stamp
}

// PlanUpgrade porta la wiki con radice root alla versione attuale del layout:
//   - crea ciò che manca (cartelle, .gitkeep, config e template zk, log.md,
//     overview.md, .gitignore);
//   - sostituisce, con copia di sicurezza, i file della configurazione zk
//     (zkFiles: path relativo alla radice -> contenuto) che esistono ma sono
//     diversi; have contiene il contenuto attuale di quelli che esistono;
//   - non tocca nient'altro che esista già: note, log.md, overview.md.
//
// index.md non è nel piano: è generato, lo si rigenera dopo l'aggiornamento.
// exists dice quali path del layout esistono già.
func PlanUpgrade(root string, zkFiles, have map[string][]byte, exists map[string]bool, stamp string) []Step {
	zkPath := map[string][]byte{}
	for rel, data := range zkFiles {
		zkPath[filepath.Join(root, rel)] = data
	}
	var steps []Step
	for _, s := range Layout(root, filepath.Base(root), zkFiles) {
		switch s := s.(type) {
		case Mkdir:
			if !exists[s.Path] {
				steps = append(steps, s)
			}
		case WriteFile:
			if s.Path == filepath.Join(root, IndexPath) {
				continue
			}
			want, managed := zkPath[s.Path]
			cur, present := have[s.Path]
			switch {
			case managed && present && !bytes.Equal(cur, want):
				steps = append(steps, Replace{Path: s.Path, Backup: BackupPath(s.Path, stamp, exists), Data: want})
			case !exists[s.Path] && !present:
				steps = append(steps, s)
			}
		}
	}
	return steps
}
