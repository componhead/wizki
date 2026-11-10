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
//   - crea ciò che manca (cartelle, .gitkeep, config e template zk, log.md
//     intestato a logName, overview.md, .gitignore);
//   - sostituisce, con copia di sicurezza, i file della configurazione zk
//     (zkFiles: path relativo alla radice -> contenuto) che esistono ma sono
//     diversi; have contiene il contenuto attuale di quelli che esistono;
//   - non tocca nient'altro che esista già: note, log.md, overview.md.
//
// index.md non è nel piano: è generato, lo si rigenera dopo l'aggiornamento.
// exists dice quali path del layout esistono già.
func PlanUpgrade(root, logName string, zkFiles, have map[string][]byte, exists map[string]bool, stamp string) []Step {
	return planManaged(Layout(root, logName, zkFiles), root, zkFiles, have, exists, stamp, filepath.Join(root, IndexPath))
}

// planManaged filtra il piano di creazione layout (radice root): tiene le
// cartelle e i file che mancano e trasforma in Replace i file gestiti
// (managed: path relativo a root -> contenuto) che esistono ma sono diversi.
// skip è un path da lasciare fuori ("" per nessuno).
func planManaged(layout []Step, root string, managed, have map[string][]byte, exists map[string]bool, stamp, skip string) []Step {
	want := map[string][]byte{}
	for rel, data := range managed {
		want[filepath.Join(root, rel)] = data
	}
	var steps []Step
	for _, s := range layout {
		switch s := s.(type) {
		case Mkdir:
			if !exists[s.Path] {
				steps = append(steps, s)
			}
		case WriteFile:
			if s.Path == skip {
				continue
			}
			data, isManaged := want[s.Path]
			cur, present := have[s.Path]
			switch {
			case isManaged && present && !bytes.Equal(cur, data):
				steps = append(steps, Replace{Path: s.Path, Backup: BackupPath(s.Path, stamp, exists), Data: data})
			case !exists[s.Path] && !present:
				steps = append(steps, s)
			}
		}
	}
	return steps
}
