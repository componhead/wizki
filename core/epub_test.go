package core

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// makeEpub costruisce in memoria uno zip con i file dati (path -> contenuto).
func makeEpub(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const testContainer = `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

// testEpub3: un EPUB 3 con indice nav, due capitoli (in ordine di spine
// inverso rispetto al manifest), un'immagine usata due volte, un link interno
// e una nota.
func testEpub3() map[string]string {
	return map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": testContainer,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Storia della scuola</dc:title>
    <dc:creator>Maria Rossi</dc:creator>
    <dc:creator>Luca Bianchi</dc:creator>
    <dc:date>2019-05-01</dc:date>
    <dc:language>it</dc:language>
    <dc:publisher>Editore Esempio</dc:publisher>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="c2" href="text/cap%202.xhtml" media-type="application/xhtml+xml"/>
    <item id="c1" href="text/cap1.xhtml" media-type="application/xhtml+xml"/>
    <item id="img" href="img/mappa.png" media-type="image/png"/>
  </manifest>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`,
		"OEBPS/nav.xhtml": `<html><body><nav epub:type="toc"><ol>
  <li><a href="text/cap1.xhtml">Le origini</a></li>
  <li><a href="text/cap%202.xhtml#s1">Il Novecento</a></li>
</ol></nav></body></html>`,
		"OEBPS/text/cap1.xhtml": `<html><head><title>Storia della scuola</title></head><body>
<p>Le prime scuole nacquero nei conventi. Vedi <a href="cap%202.xhtml#s1">il capitolo dopo</a>.</p>
<img src="../img/mappa.png" alt="Mappa"/></body></html>`,
		"OEBPS/text/cap 2.xhtml": `<html><head><title>Storia della scuola</title></head><body>
<h2>Il secolo breve</h2><p>Nel 1923 la riforma Gentile.<a href="https://example.org">fonte</a></p>
<img src="../img/mappa.png" alt="Di nuovo"/><img src="../img/manca.png" alt="Manca"/></body></html>`,
		"OEBPS/img/mappa.png": "PNG",
	}
}

// TestParseEpub3: metadati, ordine dello spine, titoli dall'indice nav,
// immagini risolte e deduplicate, link interni ridotti a testo.
func TestParseEpub3(t *testing.T) {
	b, err := ParseEpub(makeEpub(t, testEpub3()))
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "Storia della scuola" || b.Date != "2019-05-01" || b.Language != "it" || b.Publisher != "Editore Esempio" {
		t.Errorf("metadati: %+v", b)
	}
	if strings.Join(b.Authors, ";") != "Maria Rossi;Luca Bianchi" {
		t.Errorf("autori: %v", b.Authors)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("capitoli: %d", len(b.Chapters))
	}
	c1, c2 := b.Chapters[0], b.Chapters[1]
	if !strings.HasPrefix(c1, "# Le origini\n") {
		t.Errorf("cap1 senza titolo dall'indice: %q", c1)
	}
	if strings.Contains(c1, "](cap") || !strings.Contains(c1, "Vedi il capitolo dopo.") {
		t.Errorf("link interno non ridotto a testo: %q", c1)
	}
	if !strings.Contains(c1, "![Mappa](OEBPS/img/mappa.png)") {
		t.Errorf("immagine non risolta: %q", c1)
	}
	if !strings.HasPrefix(c2, "## Il secolo breve") {
		t.Errorf("cap2: il titolo del capitolo c'è già, non va aggiunto: %q", c2)
	}
	if strings.Contains(c1+c2, "# Storia della scuola") {
		t.Error("il <title> dei capitoli (titolo del libro) non deve diventare un titolo")
	}
	if !strings.Contains(c2, "[fonte](https://example.org)") {
		t.Errorf("gli URL restano: %q", c2)
	}
	if strings.Contains(c2, "manca.png") {
		t.Errorf("immagine mancante: niente riferimento rotto: %q", c2)
	}
	if len(b.ImageOrder) != 1 || string(b.Images["OEBPS/img/mappa.png"]) != "PNG" {
		t.Errorf("immagini: %v", b.ImageOrder)
	}
}

// TestParseEpub2NCX: senza nav, i titoli dei capitoli vengono dall'NCX.
func TestParseEpub2NCX(t *testing.T) {
	f := map[string]string{
		"META-INF/container.xml": testContainer,
		"OEBPS/content.opf": `<package><metadata><title>Libro</title></metadata>
  <manifest><item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  <item id="a" href="a.html" media-type="application/xhtml+xml"/></manifest>
  <spine toc="ncx"><itemref idref="a"/></spine></package>`,
		"OEBPS/toc.ncx": `<ncx><navMap><navPoint><navLabel><text>Parte prima</text></navLabel>
  <content src="a.html#p"/></navPoint></navMap></ncx>`,
		"OEBPS/a.html": `<html><body><p>Testo della parte prima.</p></body></html>`,
	}
	b, err := ParseEpub(makeEpub(t, f))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 1 || !strings.HasPrefix(b.Chapters[0], "# Parte prima\n") {
		t.Errorf("capitoli: %q", b.Chapters)
	}
}

// TestParseEpubDRM: capitoli cifrati -> ErrEpubDRM; i font offuscati
// (anche loro in encryption.xml) invece non contano.
func TestParseEpubDRM(t *testing.T) {
	enc := func(uri string) string {
		return `<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container" xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData><enc:CipherData><enc:CipherReference URI="` + uri + `"/></enc:CipherData></enc:EncryptedData></encryption>`
	}
	f := testEpub3()
	f["META-INF/encryption.xml"] = enc("OEBPS/text/cap1.xhtml")
	if _, err := ParseEpub(makeEpub(t, f)); !errors.Is(err, ErrEpubDRM) {
		t.Errorf("capitolo cifrato: err = %v", err)
	}
	f["META-INF/encryption.xml"] = enc("OEBPS/fonts/a.otf")
	if _, err := ParseEpub(makeEpub(t, f)); err != nil {
		t.Errorf("font offuscato: err = %v", err)
	}
}

// TestParseEpubErrori: zip non valido, pacchetto mancante, nessun testo.
func TestParseEpubErrori(t *testing.T) {
	if _, err := ParseEpub([]byte("non uno zip")); !errors.Is(err, ErrEpubInvalid) {
		t.Errorf("non zip: %v", err)
	}
	if _, err := ParseEpub(makeEpub(t, map[string]string{"mimetype": "application/epub+zip"})); !errors.Is(err, ErrEpubInvalid) {
		t.Errorf("senza container: %v", err)
	}
	f := testEpub3()
	f["OEBPS/text/cap1.xhtml"] = `<html><body> </body></html>`
	f["OEBPS/text/cap 2.xhtml"] = `<html><body><p></p></body></html>`
	if _, err := ParseEpub(makeEpub(t, f)); !errors.Is(err, ErrEmpty) {
		t.Errorf("senza testo: %v", err)
	}
}

// TestPlanSourceEpub: originale in raw/epub/, immagini in raw/assets/<stem>/,
// testo con metadati e riferimenti riscritti.
func TestPlanSourceEpub(t *testing.T) {
	steps, err := PlanSource("w", SourceInput{Path: "inbox/Storia Scuola.EPUB", Content: makeEpub(t, testEpub3())})
	if err != nil {
		t.Fatal(err)
	}
	want := []Step{
		Mkdir{"w/raw/epub"}, Mkdir{"w/raw/text"},
		Move{From: "inbox/Storia Scuola.EPUB", To: "w/raw/epub/storia-scuola.epub"},
		Mkdir{"w/raw/assets/storia-scuola"},
		WriteFile{Path: "w/raw/assets/storia-scuola/mappa.png", Data: []byte("PNG")},
	}
	if len(steps) != len(want)+1 {
		t.Fatalf("passi: %#v", steps)
	}
	for i, s := range want {
		if a, b := stepString(s), stepString(steps[i]); a != b {
			t.Errorf("passo %d: got %s, want %s", i, b, a)
		}
	}
	text, ok := steps[len(steps)-1].(WriteFile)
	if !ok || text.Path != "w/raw/text/storia-scuola-epub.md" {
		t.Fatalf("ultimo passo: %#v", steps[len(steps)-1])
	}
	body := string(text.Data)
	for _, s := range []string{
		"# Storia della scuola\n\n",
		"- Autore: Maria Rossi; Luca Bianchi\n- Data: 2019-05-01\n- Editore: Editore Esempio\n- Lingua: it\n",
		"![Mappa](raw/assets/storia-scuola/mappa.png)",
		"![Di nuovo](raw/assets/storia-scuola/mappa.png)",
	} {
		if !strings.Contains(body, s) {
			t.Errorf("manca %q in:\n%s", s, body)
		}
	}
}

// stepString rende confrontabile un passo (WriteFile ha []byte).
func stepString(s Step) string {
	if w, ok := s.(WriteFile); ok {
		return "write " + w.Path + " " + string(w.Data)
	}
	return fmt.Sprintf("%#v", s)
}
