package core

import (
	"reflect"
	"testing"
)

// TestBackupPath: <path>.bak se libero, altrimenti con il timestamp.
func TestBackupPath(t *testing.T) {
	if got := BackupPath("w/.zk/config.toml", "S", nil); got != "w/.zk/config.toml.bak" {
		t.Errorf("libero: %q", got)
	}
	if got := BackupPath("w/c", "S", map[string]bool{"w/c.bak": true}); got != "w/c.bak.S" {
		t.Errorf("occupato: %q", got)
	}
}

// TestPlanUpgrade: crea solo ciò che manca, sostituisce solo i file zk diversi
// (con copia), lascia stare quelli uguali, le note e index.md.
func TestPlanUpgrade(t *testing.T) {
	zk := map[string][]byte{
		".zk/config.toml":    []byte("NUOVA"),
		".zk/templates/a.md": []byte("A"),
		".zk/templates/b.md": []byte("B"),
	}
	exists := map[string]bool{}
	for _, s := range Layout("w", "w", zk) {
		switch s := s.(type) {
		case Mkdir:
			exists[s.Path] = true
		case WriteFile:
			exists[s.Path] = true
		}
	}
	// mancano: la cartella analyses (con .gitkeep), il template b, la panoramica, l'indice
	for _, p := range []string{"w/analyses", "w/analyses/.gitkeep", "w/.zk/templates/b.md", "w/overview.md", "w/index.md"} {
		delete(exists, p)
	}
	exists["w/.zk/config.toml.bak"] = true
	have := map[string][]byte{
		"w/.zk/config.toml":    []byte("VECCHIA"),
		"w/.zk/templates/a.md": []byte("A"),
	}
	got := PlanUpgrade("w", zk, have, exists, "S")
	want := []Step{
		Mkdir{"w/analyses"},
		WriteFile{Path: "w/analyses/.gitkeep"},
		Replace{Path: "w/.zk/config.toml", Backup: "w/.zk/config.toml.bak.S", Data: []byte("NUOVA")},
		WriteFile{Path: "w/.zk/templates/b.md", Data: []byte("B")},
		WriteFile{Path: "w/overview.md", Data: OverviewPage()},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v", got, want)
	}
}
