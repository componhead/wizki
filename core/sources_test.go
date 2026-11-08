package core

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestSourceKind: tipo dall'estensione, maiuscole comprese; i tipi ignoti no.
func TestSourceKind(t *testing.T) {
	cases := map[string]Kind{"a.PDF": KindPDF, "b.md": KindDoc, "c.HTM": KindDoc, "d.Jpeg": KindImage, "e.csv": KindData}
	for name, want := range cases {
		if got, ok := SourceKind(name); !ok || got != want {
			t.Errorf("SourceKind(%q) = %q, %v", name, got, ok)
		}
	}
	if _, ok := SourceKind("f.docx"); ok {
		t.Error("docx non dovrebbe essere supportato")
	}
}

// TestPlanSourceDoc: originale in raw/doc, immagini in raw/assets/<stem>/ con
// nome slug (anche due con lo stesso nome), riferimenti riscritti nel testo.
func TestPlanSourceDoc(t *testing.T) {
	in := SourceInput{
		Path:    "inbox/Il Mio Articolo.md",
		Content: []byte("![a](img/Foto%20Uno.PNG \"t\")\n![[b/foto uno.png|200]]\n![c](https://x/y.png)\n"),
		Assets: map[string]string{
			"img/Foto%20Uno.PNG": "inbox/img/Foto Uno.PNG",
			"b/foto uno.png":     "inbox/b/foto uno.png",
		},
	}
	steps, err := PlanSource("w", in)
	if err != nil {
		t.Fatal(err)
	}
	text := "![a](raw/assets/il-mio-articolo/foto-uno-2.png \"t\")\n![[raw/assets/il-mio-articolo/foto-uno.png|200]]\n![c](https://x/y.png)\n"
	want := []Step{
		Mkdir{"w/raw/doc"}, Mkdir{"w/raw/text"},
		Move{From: "inbox/Il Mio Articolo.md", To: "w/raw/doc/il-mio-articolo.md"},
		Mkdir{"w/raw/assets/il-mio-articolo"},
		Move{From: "inbox/b/foto uno.png", To: "w/raw/assets/il-mio-articolo/foto-uno.png"},
		Move{From: "inbox/img/Foto Uno.PNG", To: "w/raw/assets/il-mio-articolo/foto-uno-2.png"},
		WriteFile{Path: "w/raw/text/il-mio-articolo-md.md", Data: []byte(text)},
	}
	if !reflect.DeepEqual(steps, want) {
		t.Errorf("got  %#v\nwant %#v", steps, want)
	}
}

// TestPlanSourceAltriTipi: immagine (riferimento + OCR), dati, vuoti, ignoti, PDF.
func TestPlanSourceAltriTipi(t *testing.T) {
	steps, _ := PlanSource("w", SourceInput{Path: "in/Foto.JPG", Lang: "eng"})
	if got := steps[len(steps)-1]; got != (ImageText{Image: "w/raw/assets/foto.jpg", Out: "w/raw/text/foto-jpg.md", Header: "![foto](raw/assets/foto.jpg)\n", Lang: "eng"}) {
		t.Errorf("immagine: %#v", got)
	}
	steps, _ = PlanSource("w", SourceInput{Path: "in/d.csv", Content: []byte("a,b\n")})
	if got := steps[len(steps)-1]; !reflect.DeepEqual(got, WriteFile{Path: "w/raw/text/d-csv.md", Data: []byte("a,b\n")}) {
		t.Errorf("dati: %#v", got)
	}
	if _, err := PlanSource("w", SourceInput{Path: "in/v.txt", Content: []byte(" \n")}); !errors.Is(err, ErrEmpty) {
		t.Errorf("vuoto: %v", err)
	}
	if _, err := PlanSource("w", SourceInput{Path: "in/x.docx"}); err == nil || !strings.Contains(err.Error(), "non supportato") {
		t.Errorf("ignoto: %v", err)
	}
	steps, _ = PlanSource("w", SourceInput{Path: "in/c.pdf"})
	if _, ok := steps[len(steps)-1].(Extract); !ok {
		t.Errorf("pdf: %#v", steps)
	}
}

// TestImageRefs: solo riferimenti locali, come scritti, senza doppioni.
func TestImageRefs(t *testing.T) {
	text := `![a](x.png) ![b](<con spazio.png>) ![c](y.png "t") ![[z.png|100]] <img src='w.gif'>
![r](https://h/r.png) ![d](data:image/png;base64,AA) ![e](/abs.png) ![f](x.png)`
	want := []string{"x.png", "con spazio.png", "y.png", "z.png", "w.gif"}
	if got := ImageRefs(text); !reflect.DeepEqual(got, want) {
		t.Errorf("ImageRefs = %v", got)
	}
	if n := RemoteImageCount(text); n != 1 {
		t.Errorf("remote = %d", n)
	}
}

// TestRefPath: niente query né frammento, %XX decodificati.
func TestRefPath(t *testing.T) {
	if got := RefPath("img/a%20b.png?v=2#x"); got != "img/a b.png" {
		t.Errorf("RefPath = %q", got)
	}
}

// TestRewriteRefs: cambia solo le destinazioni mappate, nelle tre sintassi.
func TestRewriteRefs(t *testing.T) {
	in := `![a](x.png "t") ![b](<y z.png>) ![[x.png|9]] <img alt="i" src="x.png"> ![n](altro.png)`
	got := RewriteRefs(in, map[string]string{"x.png": "raw/assets/s/x.png", "y z.png": "raw/assets/s/y-z.png"})
	want := `![a](raw/assets/s/x.png "t") ![b](<raw/assets/s/y-z.png>) ![[raw/assets/s/x.png|9]] <img alt="i" src="raw/assets/s/x.png"> ![n](altro.png)`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// TestHTMLToMarkdown: tiene <article>, scarta script, menu e piè di pagina,
// converte intestazioni, enfasi, link, elenchi, immagini ed entità.
func TestHTMLToMarkdown(t *testing.T) {
	in := `<html><head><title>T &amp; t</title><style>x{}</style></head><body>
<nav><a href="/">Home</a></nav><article><h2>Sezione</h2>
<p>Uno <strong>due</strong> <em>tre</em>
<a href="https://e.it/a">link</a> <a href="#n">ancora</a>.</p>
<script>alert(1)</script><ul><li>a</li><li>b</li></ul><img alt="fig" src="f.png"><br>fine&nbsp;riga</article>
<footer>piè</footer></body></html>`
	want := "# T & t\n\n## Sezione\n\nUno **due** _tre_\n[link](https://e.it/a) ancora.\n\n- a\n- b\n\n![fig](f.png)\nfine riga\n"
	if got := HTMLToMarkdown(in); got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
	if got := HTMLToMarkdown("<script>x</script>"); got != "" {
		t.Errorf("solo script: %q", got)
	}
}

// TestInboxOrder: documenti prima delle immagini, senza allegati, nascosti e tipi ignoti.
func TestInboxOrder(t *testing.T) {
	got := InboxOrder([]string{"z.png", "b.md", ".gitkeep", "a.pdf", "allegato.png", "x.docx", "c.csv"}, map[string]bool{"allegato.png": true})
	want := []string{"a.pdf", "b.md", "c.csv", "z.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

// TestCountPhrase: singolare e plurale.
func TestCountPhrase(t *testing.T) {
	if CountPhrase(1, "a", "b") != "1 a" || CountPhrase(3, "a", "b") != "3 b" {
		t.Error("CountPhrase sbagliato")
	}
}
