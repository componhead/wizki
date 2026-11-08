package main

import (
	"bytes"
	"context"
	"os"
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
		_ = os.WriteFile(p, []byte("# t\n"), 0o644)
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
