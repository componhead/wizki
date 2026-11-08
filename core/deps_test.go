package core

import (
	"reflect"
	"strings"
	"testing"
)

// TestDeps: i comandi più una voce per ogni lingua OCR richiesta; solo git è necessario.
func TestDeps(t *testing.T) {
	var names []string
	for _, d := range Deps("ita+eng") {
		names = append(names, d.Name)
		if d.Required != (d.Name == "git") {
			t.Errorf("%s: Required = %v", d.Name, d.Required)
		}
	}
	want := []string{"git", "pdftotext", "pdftoppm", "tesseract", "tesseract-ita", "tesseract-eng", "zk"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("Deps = %v", names)
	}
}

// TestInstallCommand: pacchetti giusti per gestore, senza doppioni (poppler
// una volta sola), e zk segnalato dove il gestore non lo ha.
func TestInstallCommand(t *testing.T) {
	missing := Filter(Deps("ita"), func(d Dep) bool { return d.Name != "git" && d.Name != "tesseract" })
	cases := map[string]string{
		"pacman": "sudo pacman -S --needed poppler tesseract-data-ita zk",
		"apt":    "sudo apt install poppler-utils tesseract-ocr-ita",
		"dnf":    "sudo dnf install poppler-utils tesseract-langpack-ita",
		"brew":   "brew install poppler tesseract-lang zk",
	}
	for m, want := range cases {
		cmd, un := InstallCommand(m, missing)
		if got := strings.Join(cmd, " "); got != want {
			t.Errorf("%s: %q, want %q", m, got, want)
		}
		hasZk := len(un) == 1 && un[0].Name == "zk"
		if (m == "apt" || m == "dnf") != hasZk {
			t.Errorf("%s: senza pacchetto = %v", m, un)
		}
	}
	if cmd, un := InstallCommand("", missing); cmd != nil || len(un) != len(missing) {
		t.Errorf("gestore sconosciuto: %v %v", cmd, un)
	}
}

// TestDepReport: segno ✓/✗ e colonne allineate.
func TestDepReport(t *testing.T) {
	got := DepReport([]Dep{{Name: "git", Purpose: "a"}, {Name: "zk", Purpose: "b"}}, map[string]bool{"zk": true})
	if got != "  ✓ git  a\n  ✗ zk   b\n" {
		t.Errorf("got %q", got)
	}
}

// TestYesAnswer: le risposte affermative, in italiano e in inglese.
func TestYesAnswer(t *testing.T) {
	for _, y := range []string{"s", "Sì\n", " yes ", "Y"} {
		if !YesAnswer(y) {
			t.Errorf("%q dovrebbe essere sì", y)
		}
	}
	for _, n := range []string{"", "n", "no", "forse"} {
		if YesAnswer(n) {
			t.Errorf("%q dovrebbe essere no", n)
		}
	}
}
