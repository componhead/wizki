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
	got := PlanUpgrade("w", "w", zk, have, exists, "S")
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

// TestPlanProjectUpgrade: crea i file del progetto che mancano (con le loro
// cartelle) e sostituisce, con copia, quelli diversi.
func TestPlanProjectUpgrade(t *testing.T) {
	files := map[string][]byte{
		"README.md":             []byte("NUOVO"),
		"CLAUDE.md":             []byte("C"),
		".claude/commands/i.md": []byte("I"),
	}
	exists := map[string]bool{"p": true, "p/README.md": true, "p/CLAUDE.md": true}
	have := map[string][]byte{"p/README.md": []byte("VECCHIO"), "p/CLAUDE.md": []byte("C")}
	got := PlanProjectUpgrade("p", files, have, exists, "S")
	want := []Step{
		Mkdir{"p/.claude"},
		Mkdir{"p/.claude/commands"},
		WriteFile{Path: "p/.claude/commands/i.md", Data: []byte("I")},
		Replace{Path: "p/README.md", Backup: "p/README.md.bak", Data: []byte("NUOVO")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v", got, want)
	}
}

// TestPlanMigrate: tutto va in library/, tranne .git, library stessa e i file
// del progetto (per la loro voce di primo livello), in ordine di nome.
func TestPlanMigrate(t *testing.T) {
	project := map[string][]byte{"README.md": nil, ".claude/commands/i.md": nil}
	got := PlanMigrate("w", []string{"topics", ".git", ".zk", "README.md", ".claude", "log.md", ".gitignore"}, project)
	want := []Step{
		Mkdir{"w/library"},
		Move{From: "w/.gitignore", To: "w/library/.gitignore"},
		Move{From: "w/.zk", To: "w/library/.zk"},
		Move{From: "w/log.md", To: "w/library/log.md"},
		Move{From: "w/topics", To: "w/library/topics"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v", got, want)
	}
}
