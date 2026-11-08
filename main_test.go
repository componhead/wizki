package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBin installa in PATH un eseguibile finto.
func fakeBin(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func do(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(context.Background(), args, &out, &errb)
	return code, out.String(), errb.String()
}

// extractOut separa l'output di extract: path del testo e id del PDF.
func extractOut(t *testing.T, out string) (path, id string) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("attese 2 righe (path, id), out = %q", out)
	}
	return lines[0], lines[1]
}

func TestLintConFakeZk(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"topics/scuola.md", "topics/economia.md"} {
		p := filepath.Join(root, f)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte("---\nsummary: s\n---\n# t\n"), 0o644)
	}
	fakeBin(t, "zk", `case "$*" in
  *--orphan*) printf 'sources/orfana.md\0' ;;
  *"--tag scuola"*) printf 'sources/a.md\0sources/b.md\0' ;;
  *"--link-to topics/scuola.md"*) printf 'sources/a.md\0topics/economia.md\0' ;;
  *"--tag economia"*) printf 'sources/b.md\0' ;;
  *"--link-to topics/economia.md"*) printf 'sources/b.md\0' ;;
esac
`)
	code, out, _ := do(t, "-root", root, "lint")
	if code != 1 {
		t.Errorf("exit = %d, atteso 1\n%s", code, out)
	}
	for _, want := range []string{
		"sources/b.md ha il tag ma non linka [[topics/scuola]]",
		"sources/orfana.md: nessun link in entrata",
		"economia: una sola fonte",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("manca %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "topics/economia.md") {
		t.Errorf("i link da topics/ vanno ignorati:\n%s", out)
	}
}

func TestExtractConFakePdftotext(t *testing.T) {
	root := t.TempDir()
	pdf := filepath.Join(t.TempDir(), "Circolare 2026.pdf")
	_ = os.WriteFile(pdf, []byte("%PDF-finto"), 0o644)
	fakeBin(t, "pdftotext", `printf 'testo estratto\n' > "$3"`)

	code, out, errb := do(t, "-root", root, "extract", pdf)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	path, id := extractOut(t, out)
	if want := filepath.Join(root, "raw", "text", "circolare-2026.md"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	// sha256("%PDF-finto")
	if want := "39a364a0"; id != want {
		t.Errorf("id = %q, want %q", id, want)
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "pdf", "circolare-2026.pdf")); err != nil {
		t.Error("PDF non spostato in raw/pdf")
	}
	if _, err := os.Stat(pdf); err == nil {
		t.Error("l'originale dovrebbe essere stato spostato")
	}
}

// scanFixture: pdf finto + pdftotext che non estrae nulla.
func scanFixture(t *testing.T) (root, pdf string) {
	t.Helper()
	root = t.TempDir()
	pdf = filepath.Join(t.TempDir(), "scan.pdf")
	_ = os.WriteFile(pdf, []byte("%PDF-finto"), 0o644)
	fakeBin(t, "pdftotext", `: > "$3"`)
	return
}

func assertScanIntatta(t *testing.T, root, pdf string) {
	t.Helper()
	if _, err := os.Stat(pdf); err != nil {
		t.Error("la scansione doveva restare nella posizione originale")
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "pdf", "scan.pdf")); err == nil {
		t.Error("raw/pdf non deve contenere la scansione")
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "text", "scan.md")); err == nil {
		t.Error("raw/text non deve contenere un file parziale")
	}
}

func TestExtractNoOCRRestituisceCodice3(t *testing.T) {
	root, pdf := scanFixture(t)
	if code, _, errb := do(t, "-root", root, "-no-ocr", "extract", pdf); code != 3 || !strings.Contains(errb, "nessun testo") {
		t.Errorf("code=%d err=%q", code, errb)
	}
	assertScanIntatta(t, root, pdf)
}

// OCR finto: pdftoppm crea 10 pagine (per provare l'ordine numerico), tesseract
// risponde con il nome della pagina.
const fakePdftoppm = `prefix="$5"; for n in 1 2 10; do : > "${prefix}-$n.png"; done`
const fakeTesseract = `if [ "$1" = "--list-langs" ]; then printf 'List of available languages (2):\neng\nita\n'; exit 0; fi
printf 'testo di %s lingua %s\n' "$(basename "$1" .png)" "$4"`

func TestExtractOCRFallback(t *testing.T) {
	root, pdf := scanFixture(t)
	fakeBin(t, "pdftoppm", fakePdftoppm)
	fakeBin(t, "tesseract", fakeTesseract)

	code, out, errb := do(t, "-root", root, "extract", pdf)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	if !strings.Contains(errb, "OCR") {
		t.Errorf("atteso avviso OCR su stderr: %q", errb)
	}
	path, _ := extractOut(t, out)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "testo di p-1 lingua ita\n\ntesto di p-2 lingua ita\n\ntesto di p-10 lingua ita\n"
	if string(b) != want {
		t.Errorf("testo = %q, want %q", b, want)
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "pdf", "scan.pdf")); err != nil {
		t.Error("PDF non in raw/pdf")
	}
}

func TestExtractOCRLinguaMancante(t *testing.T) {
	root, pdf := scanFixture(t)
	fakeBin(t, "pdftoppm", fakePdftoppm)
	fakeBin(t, "tesseract", fakeTesseract)
	code, _, errb := do(t, "-root", root, "-ocr-lang", "ita+deu", "extract", pdf)
	if code != 4 || !strings.Contains(errb, "deu") {
		t.Errorf("code=%d err=%q", code, errb)
	}
	assertScanIntatta(t, root, pdf)
}

func TestExtractOCRSenzaTestoRestituisceCodice3(t *testing.T) {
	root, pdf := scanFixture(t)
	fakeBin(t, "pdftoppm", fakePdftoppm)
	fakeBin(t, "tesseract", `if [ "$1" = "--list-langs" ]; then printf 'List of available languages (1):\nita\n'; exit 0; fi
printf '  \n'`)
	if code, _, errb := do(t, "-root", root, "extract", pdf); code != 3 {
		t.Errorf("code=%d err=%q", code, errb)
	}
	assertScanIntatta(t, root, pdf)
}

func TestExtractOCRErroreDiPdftoppmFaRollback(t *testing.T) {
	root, pdf := scanFixture(t)
	fakeBin(t, "pdftoppm", `echo boom >&2; exit 1`)
	fakeBin(t, "tesseract", fakeTesseract)
	if code, _, errb := do(t, "-root", root, "extract", pdf); code != 1 || !strings.Contains(errb, "boom") {
		t.Errorf("code=%d err=%q", code, errb)
	}
	assertScanIntatta(t, root, pdf)
}

func TestExtractNonSovrascrive(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "raw", "pdf"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "raw", "pdf", "a.pdf"), []byte("vecchio"), 0o644)
	pdf := filepath.Join(t.TempDir(), "a.pdf")
	_ = os.WriteFile(pdf, []byte("nuovo"), 0o644)
	fakeBin(t, "pdftotext", `printf 'x' > "$3"`)
	if code, _, errb := do(t, "-root", root, "extract", pdf); code != 1 || !strings.Contains(errb, "immutabile") {
		t.Errorf("code=%d err=%q", code, errb)
	}
	b, _ := os.ReadFile(filepath.Join(root, "raw", "pdf", "a.pdf"))
	if string(b) != "vecchio" {
		t.Error("raw/ è stato modificato")
	}
}

// Stesso contenuto, nome diverso: exit 5, il PDF resta dov'è e raw/ non cambia.
func TestExtractDuplicatoConNomeDiverso(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "raw", "pdf"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "raw", "pdf", "circolare.pdf"), []byte("%PDF-finto"), 0o644)
	pdf := filepath.Join(t.TempDir(), "Copia di circolare.PDF")
	_ = os.WriteFile(pdf, []byte("%PDF-finto"), 0o644)
	fakeBin(t, "pdftotext", `printf 'x' > "$3"`)

	code, out, errb := do(t, "-root", root, "extract", pdf)
	if code != 5 || !strings.Contains(errb, "circolare.pdf") || !strings.Contains(errb, "39a364a0") {
		t.Errorf("code=%d out=%q err=%q", code, out, errb)
	}
	if _, err := os.Stat(pdf); err != nil {
		t.Error("il duplicato doveva restare nella posizione originale")
	}
	entries, _ := os.ReadDir(filepath.Join(root, "raw", "pdf"))
	if len(entries) != 1 {
		t.Errorf("raw/pdf modificato: %d file", len(entries))
	}
	if _, err := os.Stat(filepath.Join(root, "raw", "text")); err == nil {
		t.Error("raw/text non doveva essere creato")
	}
}

// Contenuto diverso: nessun falso positivo anche se raw/pdf non è vuoto.
func TestExtractContenutoDiversoNonEDuplicato(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "raw", "pdf"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "raw", "pdf", "altro.pdf"), []byte("%PDF-altro"), 0o644)
	pdf := filepath.Join(t.TempDir(), "nuovo.pdf")
	_ = os.WriteFile(pdf, []byte("%PDF-finto"), 0o644)
	fakeBin(t, "pdftotext", `printf 'x' > "$3"`)
	if code, _, errb := do(t, "-root", root, "extract", pdf); code != 0 {
		t.Errorf("code=%d err=%q", code, errb)
	}
}

// Lint: id duplicati e id mancanti nelle note-fonte.
func TestLintIdFonti(t *testing.T) {
	root := t.TempDir()
	notes := map[string]string{
		"sources/39a364a0-circolare.md":       "---\nid: 39a364a0\ntitle: Circolare\n---\n",
		"sources/39a364a0-copia-circolare.md": "---\nid: 39a364a0\ntitle: Copia\n---\n",
		"sources/senza-id.md":                 "---\ntitle: Senza id\n---\n",
		"sources/ok.md":                       "---\nid: deadbeef\n---\n",
	}
	for f, c := range notes {
		p := filepath.Join(root, f)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte(c), 0o644)
	}
	fakeBin(t, "zk", ":")
	code, out, _ := do(t, "-root", root, "lint")
	if code != 1 {
		t.Errorf("exit = %d, atteso 1\n%s", code, out)
	}
	for _, want := range []string{
		"id 39a364a0: stesso PDF in più note: sources/39a364a0-circolare.md, sources/39a364a0-copia-circolare.md",
		"sources/senza-id.md: id mancante nel frontmatter",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("manca %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "deadbeef") {
		t.Errorf("id unico segnalato per errore:\n%s", out)
	}
}

func TestLogAppendOnly(t *testing.T) {
	root := t.TempDir()
	for _, title := range []string{"Primo", "Secondo"} {
		if code, _, errb := do(t, "-root", root, "log", "ingest", title); code != 0 {
			t.Fatalf("exit %d: %s", code, errb)
		}
	}
	b, _ := os.ReadFile(filepath.Join(root, "log.md"))
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "ingest | Primo") || !strings.HasSuffix(lines[1], "ingest | Secondo") {
		t.Errorf("log = %q", b)
	}
}

func TestInitCreaLaWiki(t *testing.T) {
	t.Chdir(t.TempDir())
	code, out, errb := do(t, "init", "mia")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	for _, p := range []string{".", "inbox", "raw", "raw/pdf", "raw/doc", "raw/assets", "raw/data", "raw/text", "sources", "topics", "entities", "analyses", ".zk", ".zk/templates", ".git"} {
		if fi, err := os.Stat(filepath.Join("mia", p)); err != nil || !fi.IsDir() {
			t.Errorf("manca la cartella %s", p)
		}
		if p == ".git" {
			continue
		}
		if _, err := os.Stat(filepath.Join("mia", p, ".gitkeep")); err != nil {
			t.Errorf("manca %s/.gitkeep", p)
		}
	}
	cfg, _ := os.ReadFile(filepath.Join("mia", ".zk", "config.toml"))
	want, _ := zkConfig.ReadFile("library/.zk/config.toml")
	if len(cfg) == 0 || !bytes.Equal(cfg, want) {
		t.Error("config.toml diverso da quello incorporato")
	}
	for _, p := range []string{".zk/templates/source.md", ".zk/templates/topic.md", ".zk/templates/entity.md", ".zk/templates/analysis.md", "overview.md", "index.md"} {
		if _, err := os.Stat(filepath.Join("mia", p)); err != nil {
			t.Errorf("manca %s", p)
		}
	}
	if b, _ := os.ReadFile(filepath.Join("mia", "log.md")); string(b) != "# Log: mia\n\n" {
		t.Errorf("log.md = %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join("mia", ".gitignore")); !strings.Contains(string(b), ".zk/notebook.db") {
		t.Errorf(".gitignore = %q", b)
	}
	head, _ := os.ReadFile(filepath.Join("mia", ".git", "HEAD"))
	if strings.TrimSpace(string(head)) != "ref: refs/heads/master" {
		t.Errorf("HEAD = %q", head)
	}
	if !strings.Contains(out, `wiki "mia" pronta`) {
		t.Errorf("out = %s", out)
	}
}

func TestInitNonSovrascrive(t *testing.T) {
	t.Chdir(t.TempDir())
	if code, _, errb := do(t, "init", "mia"); code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	log := filepath.Join("mia", "log.md")
	if err := os.WriteFile(log, []byte("mio log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := do(t, "init", "mia")
	if code != 0 {
		t.Fatalf("seconda init: exit %d: %s", code, errb)
	}
	if b, _ := os.ReadFile(log); string(b) != "mio log\n" {
		t.Errorf("log.md sovrascritto: %q", b)
	}
	for _, want := range []string{"già presente: mia/log.md", "già presente: mia/.git"} {
		if !strings.Contains(out, want) {
			t.Errorf("manca %q in:\n%s", want, out)
		}
	}
}

func TestInitNomeNonValido(t *testing.T) {
	t.Chdir(t.TempDir())
	if code, _, _ := do(t, "init", "a/b"); code != 1 {
		t.Errorf("exit = %d, atteso 1", code)
	}
	if code, _, _ := do(t, "init"); code != 1 {
		t.Errorf("senza nome: exit = %d, atteso 1", code)
	}
}

// TestIndexPoiLintSenzaWarn: `wizki index` scrive un indice che il lint
// considera aggiornato; la panoramica senza link in entrata non è orfana.
func TestIndexPoiLintSenzaWarn(t *testing.T) {
	root := t.TempDir()
	notes := map[string]string{
		"overview.md":          "---\ntitle: Panoramica\nsummary: tesi\n---\n[[topics/scuola|scuola]]\n",
		"topics/scuola.md":     "---\ntitle: scuola\nsummary: la scuola\n---\n",
		"entities/mim.md":      "---\ntitle: MIM\ntipo: ente\nsummary: ministero\n---\n",
		"sources/1-a.md":       "---\nid: 11111111\ntitle: A\nsummary: a\ndate: 2026-11-08\npublished: 2026-10\n---\n#scuola [[topics/scuola|scuola]] [[entities/mim|MIM]]\n",
		"sources/ignorato.txt": "non è una pagina",
	}
	for f, c := range notes {
		p := filepath.Join(root, f)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte(c), 0o644)
	}
	fakeBin(t, "zk", `case "$*" in
  *--orphan*) printf 'overview.md\0' ;;
  *"--tag scuola"*) printf 'sources/1-a.md\0' ;;
  *"--link-to topics/scuola.md"*) printf 'sources/1-a.md\0overview.md\0' ;;
esac
`)
	code, out, errb := do(t, "-root", root, "index")
	if code != 0 || strings.TrimSpace(out) != filepath.Join(root, "index.md") {
		t.Fatalf("index: exit %d, out %q, err %q", code, out, errb)
	}
	idx, _ := os.ReadFile(filepath.Join(root, "index.md"))
	for _, want := range []string{
		"- [[topics/scuola|scuola]] — la scuola · 1 fonte",
		"- [[entities/mim|MIM]] — ministero · 1 fonte",
		"- [[sources/1-a|A]] — a · pubbl. 2026-10 · acq. 2026-11-08",
	} {
		if !strings.Contains(string(idx), want) {
			t.Errorf("manca %q in index.md:\n%s", want, idx)
		}
	}
	if strings.Contains(string(idx), "ignorato") {
		t.Errorf("file non .md nell'indice:\n%s", idx)
	}
	code, out, _ = do(t, "-root", root, "lint")
	if code != 0 || strings.Contains(out, "[warn]") {
		t.Errorf("lint dopo index: exit %d\n%s", code, out)
	}
	_ = os.WriteFile(filepath.Join(root, "topics", "scuola.md"), []byte("---\ntitle: scuola\nsummary: cambiato\n---\n"), 0o644)
	if code, out, _ = do(t, "-root", root, "lint"); code != 1 || !strings.Contains(out, "index.md: non aggiornato") {
		t.Errorf("lint dopo una modifica: exit %d\n%s", code, out)
	}
}

// TestUpgrade: senza -apply non cambia nulla; con -apply aggiorna la config
// (con .bak), crea ciò che manca, rigenera index.md e non tocca le note;
// rilanciato, non ha più niente da fare.
func TestUpgrade(t *testing.T) {
	t.Chdir(t.TempDir())
	if code, _, errb := do(t, "init", "w"); code != 0 {
		t.Fatalf("init: %s", errb)
	}
	cfg := filepath.Join("w", ".zk", "config.toml")
	_ = os.WriteFile(cfg, []byte("vecchia\n"), 0o644)
	_ = os.RemoveAll(filepath.Join("w", "entities"))
	_ = os.Remove(filepath.Join("w", "index.md"))
	note := filepath.Join("w", "topics", "mia.md")
	_ = os.WriteFile(note, []byte("---\ntitle: mia\n---\n"), 0o644)

	code, out, errb := do(t, "-root", "w", "upgrade")
	if code != 0 || !strings.Contains(out, "prova: nessuna modifica") ||
		!strings.Contains(out, "aggiorna: w/.zk/config.toml (copia del vecchio in w/.zk/config.toml.bak)") ||
		!strings.Contains(out, "crea: w/entities/") || !strings.Contains(out, "crea: w/index.md") {
		t.Fatalf("prova: exit %d\n%s%s", code, out, errb)
	}
	if b, _ := os.ReadFile(cfg); string(b) != "vecchia\n" {
		t.Error("la prova ha modificato la config")
	}

	if code, out, errb = do(t, "-root", "w", "upgrade", "-apply"); code != 0 {
		t.Fatalf("apply: exit %d\n%s%s", code, out, errb)
	}
	want, _ := zkConfig.ReadFile("library/.zk/config.toml")
	if b, _ := os.ReadFile(cfg); !bytes.Equal(b, want) {
		t.Error("config non aggiornata")
	}
	if b, _ := os.ReadFile(cfg + ".bak"); string(b) != "vecchia\n" {
		t.Errorf("copia .bak = %q", b)
	}
	for _, p := range []string{"entities/.gitkeep", "index.md"} {
		if _, err := os.Stat(filepath.Join("w", p)); err != nil {
			t.Errorf("manca %s", p)
		}
	}
	if idx, _ := os.ReadFile(filepath.Join("w", "index.md")); !strings.Contains(string(idx), "[[topics/mia|mia]]") {
		t.Errorf("index.md non rigenerato:\n%s", idx)
	}
	if b, _ := os.ReadFile(note); string(b) != "---\ntitle: mia\n---\n" {
		t.Errorf("nota modificata: %q", b)
	}

	if code, out, _ = do(t, "-root", "w", "upgrade"); code != 0 || !strings.Contains(out, "w è già aggiornata") {
		t.Errorf("secondo giro: exit %d\n%s", code, out)
	}
}

// TestUpgradeNonWiki: una cartella senza .zk/config.toml non viene toccata.
func TestUpgradeNonWiki(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := do(t, "-root", dir, "upgrade", "-apply")
	if code != 1 || !strings.Contains(errb, "non sembra una wiki") {
		t.Errorf("exit %d: %s", code, errb)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("creati file in una cartella che non è una wiki: %v", entries)
	}
}

// TestExtractDocConImmagini: un .md della inbox va in raw/doc con le sue
// immagini locali; il testo punta ai nuovi path; l'id è lo SHA del .md.
func TestExtractDocConImmagini(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "inbox")
	_ = os.MkdirAll(filepath.Join(in, "img"), 0o755)
	_ = os.WriteFile(filepath.Join(in, "img", "Foto 1.png"), []byte("PNG"), 0o644)
	md := filepath.Join(in, "Nota.md")
	_ = os.WriteFile(md, []byte("![f](img/Foto%201.png)\n![r](https://h/r.png)\n"), 0o644)

	code, out, errb := do(t, "-root", root, "extract", md)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	path, _ := extractOut(t, out)
	if want := filepath.Join(root, "raw", "text", "nota-md.md"); path != want {
		t.Errorf("path = %q", path)
	}
	if b, _ := os.ReadFile(path); string(b) != "![f](raw/assets/nota/foto-1.png)\n![r](https://h/r.png)\n" {
		t.Errorf("testo = %q", b)
	}
	for _, p := range []string{"raw/doc/nota.md", "raw/assets/nota/foto-1.png"} {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Errorf("manca %s", p)
		}
	}
	if !strings.Contains(errb, "1 immagine locale") || !strings.Contains(errb, "1 immagine remota non scaricata") {
		t.Errorf("stderr = %q", errb)
	}
}

// TestExtractDestinazioneEsistenteNonToccaNulla: se il testo esiste già in
// raw/text, niente viene spostato (né la fonte né le sue immagini).
func TestExtractDestinazioneEsistenteNonToccaNulla(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "inbox")
	_ = os.MkdirAll(in, 0o755)
	_ = os.WriteFile(filepath.Join(in, "a.png"), []byte("PNG"), 0o644)
	md := filepath.Join(in, "x.md")
	_ = os.WriteFile(md, []byte("![](a.png)\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "raw", "text"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "raw", "text", "x-md.md"), []byte("vecchio"), 0o644)

	code, _, errb := do(t, "-root", root, "extract", md)
	if code != 1 || !strings.Contains(errb, "esiste già") {
		t.Errorf("exit %d: %s", code, errb)
	}
	for _, p := range []string{md, filepath.Join(in, "a.png")} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s spostato", p)
		}
	}
}

// TestExtractImmagineConOCR: l'immagine va in raw/assets, il testo ha il
// riferimento e l'OCR; senza OCR solo il riferimento, con un avviso.
func TestExtractImmagineConOCR(t *testing.T) {
	root := t.TempDir()
	img := filepath.Join(t.TempDir(), "Cartello.PNG")
	_ = os.WriteFile(img, []byte("PNG-1"), 0o644)
	fakeBin(t, "tesseract", `case "$1" in
  --list-langs) printf 'List of available languages (1):\nita\n' ;;
  *) printf 'CIAO\n' ;;
esac
`)
	code, out, errb := do(t, "-root", root, "extract", img)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errb)
	}
	path, _ := extractOut(t, out)
	if b, _ := os.ReadFile(path); string(b) != "![cartello](raw/assets/cartello.png)\n\n## Testo (OCR)\n\nCIAO\n" {
		t.Errorf("testo = %q", b)
	}

	img2 := filepath.Join(t.TempDir(), "altra.png")
	_ = os.WriteFile(img2, []byte("PNG-2"), 0o644)
	code, out, errb = do(t, "-root", root, "-ocr-lang", "klingon", "extract", img2)
	path, _ = extractOut(t, out)
	if b, _ := os.ReadFile(path); code != 0 || string(b) != "![altra](raw/assets/altra.png)\n" || !strings.Contains(errb, "niente OCR") {
		t.Errorf("senza OCR: exit %d, testo %q, err %q", code, b, errb)
	}
}

// TestExtractDuplicatoTraTipi: lo stesso contenuto già acquisito con un altro
// nome (in raw/doc) è un duplicato anche se cambia l'estensione del tipo doc.
func TestExtractDuplicatoTraTipi(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(t.TempDir(), "a.md")
	_ = os.WriteFile(a, []byte("uguale\n"), 0o644)
	if code, _, errb := do(t, "-root", root, "extract", a); code != 0 {
		t.Fatalf("prima: %s", errb)
	}
	b := filepath.Join(t.TempDir(), "b.txt")
	_ = os.WriteFile(b, []byte("uguale\n"), 0o644)
	if code, _, errb := do(t, "-root", root, "extract", b); code != 5 || !strings.Contains(errb, "raw/doc/a.md") {
		t.Errorf("exit %d: %s", code, errb)
	}
}

// TestExtractVuotoETipoIgnoto: testo vuoto -> 3, tipo non supportato -> 1; la fonte resta.
func TestExtractVuotoETipoIgnoto(t *testing.T) {
	root := t.TempDir()
	v := filepath.Join(t.TempDir(), "v.txt")
	_ = os.WriteFile(v, []byte("\n"), 0o644)
	if code, _, _ := do(t, "-root", root, "extract", v); code != 3 {
		t.Errorf("vuoto: exit %d", code)
	}
	x := filepath.Join(t.TempDir(), "x.docx")
	_ = os.WriteFile(x, []byte("d"), 0o644)
	if code, _, errb := do(t, "-root", root, "extract", x); code != 1 || !strings.Contains(errb, "non supportato") {
		t.Errorf("ignoto: exit %d %s", code, errb)
	}
	for _, p := range []string{v, x} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s spostato", p)
		}
	}
}

// TestInbox: elenco ordinato, senza le immagini allegate ai documenti.
func TestInbox(t *testing.T) {
	root := t.TempDir()
	in := filepath.Join(root, "inbox")
	_ = os.MkdirAll(in, 0o755)
	files := map[string]string{
		"z.png": "x", "allegato.png": "x", "a.pdf": "x", ".gitkeep": "",
		"p.html": `<p>x</p><img src="allegato.png">`, "n.docx": "x",
	}
	for f, c := range files {
		_ = os.WriteFile(filepath.Join(in, f), []byte(c), 0o644)
	}
	code, out, errb := do(t, "-root", root, "inbox")
	if code != 0 || out != "a.pdf\np.html\nz.png\n" {
		t.Errorf("exit %d, out %q, err %q", code, out, errb)
	}
}

// TestMain: nei test nessuno risponde alle domande, qualunque sia lo stdin.
func TestMain(m *testing.M) {
	interactive = func() bool { return false }
	os.Exit(m.Run())
}

// onlyBins sostituisce il PATH con una cartella che contiene solo questi
// eseguibili finti (nome -> script sh).
func onlyBins(t *testing.T, bins map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, script := range bins {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	return dir
}

// TestDepsPropostaSenzaTerminale: elenca ciò che manca e stampa il comando
// di installazione, senza lanciarlo; exit 1.
func TestDepsPropostaSenzaTerminale(t *testing.T) {
	onlyBins(t, map[string]string{
		"git":       "",
		"pacman":    "",
		"tesseract": `printf 'List of available languages (1):\neng\n'`,
	})
	code, out, _ := do(t, "deps")
	for _, want := range []string{
		"✗ pdftotext", "✓ tesseract ", "✗ tesseract-ita", "✗ zk",
		"sudo pacman -S --needed poppler tesseract-data-ita zk",
		"Lancialo in un terminale",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("manca %q in:\n%s", want, out)
		}
	}
	if code != 1 {
		t.Errorf("exit %d", code)
	}
}

// TestDepsInstallaSeConfermato: con un terminale e la risposta "s" lancia il
// comando proposto, poi ricontrolla.
func TestDepsInstallaSeConfermato(t *testing.T) {
	log := filepath.Join(t.TempDir(), "sudo.log")
	onlyBins(t, map[string]string{
		"git": "", "pdftotext": "", "pdftoppm": "", "zk": "", "brew": "",
		"tesseract": `printf 'List of available languages (1):\neng\n'`,
	})
	interactive = func() bool { return true }
	stdin = strings.NewReader("s\n")
	t.Cleanup(func() { interactive = func() bool { return false }; stdin = os.Stdin })
	// brew non usa sudo: il finto brew registra gli argomenti
	_ = os.WriteFile(filepath.Join(filepath.Dir(lookPathOrFail(t, "brew")), "brew"), []byte("#!/bin/sh\necho \"$@\" > "+log+"\n"), 0o755)

	_, out, _ := do(t, "deps")
	if b, _ := os.ReadFile(log); strings.TrimSpace(string(b)) != "install tesseract-lang" {
		t.Errorf("comando lanciato = %q\n%s", b, out)
	}
	if strings.Count(out, "Dipendenze:") != 2 {
		t.Errorf("atteso un secondo controllo dopo l'installazione:\n%s", out)
	}
}

func lookPathOrFail(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestInitSenzaGit: senza git, init si ferma prima di creare qualsiasi cosa.
func TestInitSenzaGit(t *testing.T) {
	t.Chdir(t.TempDir())
	onlyBins(t, map[string]string{"zk": ""})
	code, _, errb := do(t, "init", "w")
	if code != 1 || !strings.Contains(errb, "manca git") {
		t.Errorf("exit %d: %s", code, errb)
	}
	if _, err := os.Stat("w"); err == nil {
		t.Error("init ha creato la cartella senza git")
	}
}
