package core

import (
	"fmt"
	"sort"
	"strings"
)

// Dep è un programma (o una lingua di Tesseract) di cui wizki ha bisogno.
type Dep struct {
	Name    string // nome mostrato: il comando, o "tesseract-<lingua>"
	Purpose string // a cosa serve, in parole semplici
	Lang    string // per le lingue di Tesseract: il codice (es. "ita"); "" per i comandi
	// Required: senza, `init` non può funzionare (gli altri limitano solo
	// alcune funzioni: OCR, lint, ...).
	Required bool
}

// Managers sono i gestori di pacchetti riconosciuti, nell'ordine in cui si
// cercano.
var Managers = []string{"pacman", "apt", "dnf", "brew"}

// pkgOf: per gestore, il pacchetto che fornisce ogni comando ("" = non è nei
// pacchetti di quel gestore).
var pkgOf = map[string]map[string]string{
	"pacman": {"git": "git", "pdftotext": "poppler", "pdftoppm": "poppler", "tesseract": "tesseract", "zk": "zk"},
	"apt":    {"git": "git", "pdftotext": "poppler-utils", "pdftoppm": "poppler-utils", "tesseract": "tesseract-ocr", "zk": ""},
	"dnf":    {"git": "git", "pdftotext": "poppler-utils", "pdftoppm": "poppler-utils", "tesseract": "tesseract", "zk": ""},
	"brew":   {"git": "git", "pdftotext": "poppler", "pdftoppm": "poppler", "tesseract": "tesseract", "zk": "zk"},
}

// langPkg: il pacchetto con una lingua di Tesseract, per gestore.
func langPkg(manager, lang string) string {
	switch manager {
	case "pacman":
		return "tesseract-data-" + lang
	case "apt":
		return "tesseract-ocr-" + lang
	case "dnf":
		return "tesseract-langpack-" + lang
	case "brew":
		return "tesseract-lang" // tutte le lingue in un pacchetto
	}
	return ""
}

// installPrefix: il comando di installazione di ogni gestore.
var installPrefix = map[string][]string{
	"pacman": {"sudo", "pacman", "-S", "--needed"},
	"apt":    {"sudo", "apt", "install"},
	"dnf":    {"sudo", "dnf", "install"},
	"brew":   {"brew", "install"},
}

// ZkInstallURL: dove trovare zk quando il gestore non lo ha.
const ZkInstallURL = "https://github.com/zk-org/zk#install"

// Deps elenca le dipendenze di wizki, comprese le lingue OCR richieste
// (lang, es. "ita" o "ita+eng").
func Deps(lang string) []Dep {
	deps := []Dep{
		{Name: "git", Purpose: "storia della wiki (serve a init)", Required: true},
		{Name: "pdftotext", Purpose: "testo dei PDF"},
		{Name: "pdftoppm", Purpose: "OCR dei PDF scansionati"},
		{Name: "tesseract", Purpose: "OCR di scansioni e immagini"},
	}
	for _, l := range strings.Split(lang, "+") {
		if l = strings.TrimSpace(l); l != "" {
			deps = append(deps, Dep{Name: "tesseract-" + l, Purpose: "lingua " + l + " per l'OCR", Lang: l})
		}
	}
	return append(deps, Dep{Name: "zk", Purpose: "note, tag e link (serve a lint)"})
}

// DepReport: una riga per dipendenza, ✓ se c'è e ✗ se manca.
func DepReport(deps []Dep, missing map[string]bool) string {
	var b strings.Builder
	width := 0
	for _, d := range deps {
		width = max(width, len(d.Name))
	}
	for _, d := range deps {
		mark := "✓"
		if missing[d.Name] {
			mark = "✗"
		}
		fmt.Fprintf(&b, "  %s %-*s  %s\n", mark, width, d.Name, d.Purpose)
	}
	return b.String()
}

// InstallCommand restituisce il comando che installa le dipendenze mancanti
// con il gestore dato (pacchetti in ordine, senza doppioni) e quelle che quel
// gestore non ha. Con un gestore sconosciuto cmd è nil e tutte sono "senza
// pacchetto".
func InstallCommand(manager string, missing []Dep) (cmd []string, unpackaged []Dep) {
	pkgs, known := pkgOf[manager]
	seen := map[string]bool{}
	var names []string
	for _, d := range missing {
		p := ""
		switch {
		case !known:
		case d.Lang != "":
			p = langPkg(manager, d.Lang)
		default:
			p = pkgs[d.Name]
		}
		if p == "" {
			unpackaged = append(unpackaged, d)
			continue
		}
		if !seen[p] {
			seen[p] = true
			names = append(names, p)
		}
	}
	if len(names) == 0 {
		return nil, unpackaged
	}
	sort.Strings(names)
	return append(append([]string{}, installPrefix[manager]...), names...), unpackaged
}

// YesAnswer: la risposta a una domanda sì/no è un sì (s, si, sì, y, yes).
func YesAnswer(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "s", "si", "sì", "y", "yes":
		return true
	}
	return false
}
