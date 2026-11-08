package core

import (
	"reflect"
	"strings"
	"testing"

	"wizki/i18n"
)

// TestWikiLinks: destinazioni normalizzate (senza .md né titolo), distinte, in ordine.
func TestWikiLinks(t *testing.T) {
	got := WikiLinks("vedi [[topics/scuola|scuola]], [[entities/mim.md|MIM]] e [[topics/scuola#sintesi|ancora]]; non [link](x)")
	want := []string{"topics/scuola", "entities/mim"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WikiLinks = %v, want %v", got, want)
	}
}

// TestParsePage: campi dal frontmatter, titolo di ripiego dal primo "# " o dal nome file.
func TestParsePage(t *testing.T) {
	p := ParsePage("entities/mim.md", "---\ntitle: Ministero\ntipo: ente\nsummary: Il ministero: scuola e merito\n---\n# Altro\n[[topics/scuola|scuola]]\n")
	want := Page{Path: "entities/mim.md", Title: "Ministero", Tipo: "ente", Summary: "Il ministero: scuola e merito", Links: []string{"topics/scuola"}}
	if !reflect.DeepEqual(p, want) {
		t.Errorf("got %#v", p)
	}
	if got := ParsePage("topics/a.md", "\n# Titolo H1\n").Title; got != "Titolo H1" {
		t.Errorf("titolo da H1 = %q", got)
	}
	if got := ParsePage("topics/senza-titolo.md", "testo").Title; got != "senza-titolo" {
		t.Errorf("titolo da nome file = %q", got)
	}
}

// TestBuildIndex: sezioni nell'ordine atteso, entità per tipo, fonti dalla
// pubblicazione più recente, conteggio delle fonti che linkano ogni topic/entità.
func TestBuildIndex(t *testing.T) {
	pages := []Page{
		{Path: "overview.md", Title: "Panoramica", Summary: "La tesi"},
		{Path: "sources/a.md", Title: "Fonte A", Summary: "A", Date: "2026-01-01", Published: "2025-06", Links: []string{"topics/scuola", "entities/mim"}},
		{Path: "sources/b.md", Title: "Fonte B", Date: "2026-02-01", Links: []string{"topics/scuola"}},
		{Path: "topics/scuola.md", Title: "scuola", Summary: "La scuola", Links: []string{"entities/mim"}},
		{Path: "topics/economia.md", Title: "economia", Summary: "Soldi"},
		{Path: "entities/mim.md", Title: "MIM", Tipo: "ente", Summary: "Ministero"},
		{Path: "entities/roma.md", Title: "Roma", Summary: "Città"},
		{Path: "analyses/2026-03-01-b.md", Title: "Analisi B", Summary: "Bilancio", Date: "2026-03-01", Links: []string{"topics/scuola"}},
		{Path: "analyses/2026-04-01-a.md", Title: "Analisi A", Summary: "Confronto", Date: "2026-04-01"},
	}
	want := `# Indice

<!-- Generato da ` + "`wizki index`" + `: non modificare a mano. -->

## Panoramica

- [[overview|Panoramica]] — La tesi

## Topic (2)

- [[topics/economia|economia]] — Soldi · 0 fonti
- [[topics/scuola|scuola]] — La scuola · 2 fonti

## Entità (2)

### altro

- [[entities/roma|Roma]] — Città · 0 fonti

### ente

- [[entities/mim|MIM]] — Ministero · 1 fonte

## Analisi (2)

- [[analyses/2026-04-01-a|Analisi A]] — Confronto · 2026-04-01
- [[analyses/2026-03-01-b|Analisi B]] — Bilancio · 2026-03-01

## Fonti (2)

- [[sources/a|Fonte A]] — A · pubbl. 2025-06 · acq. 2026-01-01
- [[sources/b|Fonte B]] — _(riassunto mancante)_ · pubbl. ? · acq. 2026-02-01
`
	if got := BuildIndex(pages); got != want {
		t.Errorf("BuildIndex =\n%s\nwant\n%s", got, want)
	}
	if got := BuildIndex(nil); !strings.Contains(got, "## Topic (0)\n\n_(nessuna pagina)_\n") || strings.Contains(got, "## Panoramica") {
		t.Errorf("wiki vuota:\n%s", got)
	}
}

// TestSummaryFindings: solo le pagine senza summary:, in ordine di path.
func TestSummaryFindings(t *testing.T) {
	got := SummaryFindings([]Page{{Path: "topics/b.md"}, {Path: "topics/a.md", Summary: "ok"}, {Path: "overview.md"}})
	want := []Finding{
		{"warn", "overview.md", "summary: mancante nel frontmatter"},
		{"warn", "topics/b.md", "summary: mancante nel frontmatter"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

// TestIndexFindings: indice mancante, non aggiornato o allineato alle pagine.
func TestIndexFindings(t *testing.T) {
	pages := []Page{{Path: "topics/a.md", Title: "a", Summary: "s"}}
	if fs := IndexFindings("", false, pages); len(fs) != 1 || !strings.Contains(fs[0].Msg, "mancante") {
		t.Errorf("mancante: %v", fs)
	}
	if fs := IndexFindings("# vecchio\n", true, pages); len(fs) != 1 || !strings.Contains(fs[0].Msg, "non aggiornato") {
		t.Errorf("non aggiornato: %v", fs)
	}
	if fs := IndexFindings(BuildIndex(pages), true, pages); fs != nil {
		t.Errorf("aggiornato: %v", fs)
	}
}

// TestNotOverview: la panoramica è esclusa dalle orfane, le altre pagine no.
func TestNotOverview(t *testing.T) {
	if NotOverview("overview.md") || !NotOverview("topics/overview.md") {
		t.Error("NotOverview sbagliato")
	}
}

// TestValidPublished: tre granularità di data o "sconosciuta"; niente altro.
func TestValidPublished(t *testing.T) {
	for _, ok := range []string{"2024", "2024-03", "2024-03-31", "sconosciuta"} {
		if !ValidPublished(ok) {
			t.Errorf("%q dovrebbe essere valido", ok)
		}
	}
	for _, bad := range []string{"", "24", "2024-13", "2024-03-32", "03/2024", "marzo 2024"} {
		if ValidPublished(bad) {
			t.Errorf("%q dovrebbe essere non valido", bad)
		}
	}
}

// TestPublishedFindings: solo le fonti, published: mancante o malformato.
func TestPublishedFindings(t *testing.T) {
	got := PublishedFindings([]Page{
		{Path: "sources/a.md"},
		{Path: "sources/b.md", Published: "marzo"},
		{Path: "sources/c.md", Published: "sconosciuta"},
		{Path: "topics/t.md"},
	})
	if len(got) != 2 || got[0].Subject != "sources/a.md" || !strings.Contains(got[1].Msg, `"marzo" non valido`) {
		t.Errorf("got %v", got)
	}
}

// TestBrokenLinkFindings: link verso pagine che non esistono, da qualunque pagina.
func TestBrokenLinkFindings(t *testing.T) {
	got := BrokenLinkFindings([]Page{
		{Path: "overview.md", Links: []string{"topics/scuola", "topics/sparito"}},
		{Path: "topics/scuola.md", Links: []string{"overview"}},
	})
	want := []Finding{{"warn", "overview.md", "link a pagina inesistente [[topics/sparito]]"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

// TestCrossRefFindings: due topic nella stessa fonte devono linkarsi (in un
// verso qualsiasi); le entità non contano.
func TestCrossRefFindings(t *testing.T) {
	pages := []Page{
		{Path: "sources/s1.md", Links: []string{"topics/b", "topics/a", "topics/c", "entities/e"}},
		{Path: "sources/s2.md", Links: []string{"topics/a", "topics/b"}},
		{Path: "topics/a.md"},
		{Path: "topics/b.md"},
		{Path: "topics/c.md", Links: []string{"topics/a"}},
		{Path: "entities/e.md"},
	}
	got := CrossRefFindings(pages)
	if len(got) != 2 {
		t.Fatalf("attese 2 coppie, got %v", got)
	}
	if got[0].Subject != "topics/a ~ topics/b" || !strings.Contains(got[0].Msg, "sources/s1.md, sources/s2.md") {
		t.Errorf("coppia a~b: %v", got[0])
	}
	if got[1].Subject != "topics/b ~ topics/c" {
		t.Errorf("coppia b~c: %v", got[1])
	}
}

// TestOutOfOrderFindings: una fonte più vecchia acquisita dopo una più recente
// genera un info sulla pagina che le linka entrambe; date mancanti ignorate.
func TestOutOfOrderFindings(t *testing.T) {
	pages := []Page{
		{Path: "sources/nuova.md", Date: "2026-01-10", Published: "2025", Links: []string{"topics/t", "entities/e"}},
		{Path: "sources/vecchia.md", Date: "2026-02-01", Published: "2019-05", Links: []string{"topics/t"}},
		{Path: "sources/ignota.md", Date: "2026-03-01", Published: "sconosciuta", Links: []string{"topics/t"}},
		{Path: "topics/t.md"},
		{Path: "entities/e.md"},
	}
	got := OutOfOrderFindings(pages)
	if len(got) != 1 || got[0].Level != "info" || got[0].Subject != "topics/t.md" ||
		!strings.Contains(got[0].Msg, "sources/vecchia.md (pubbl. 2019-05) acquisita dopo sources/nuova.md (pubbl. 2025)") {
		t.Errorf("got %v", got)
	}
	inOrder := []Page{pages[0], {Path: "sources/x.md", Date: "2026-02-01", Published: "2026", Links: []string{"topics/t"}}, pages[3]}
	if fs := OutOfOrderFindings(inOrder); fs != nil {
		t.Errorf("fonti in ordine: %v", fs)
	}
}

// TestAnalysisFindings: warn per l'analisi senza riferimenti; info per quella
// su cui sono arrivate fonti nuove (non per fonti già citate o più vecchie).
func TestAnalysisFindings(t *testing.T) {
	pages := []Page{
		{Path: "analyses/2026-02-01-ok.md", Date: "2026-02-01", Links: []string{"topics/scuola", "sources/vecchia"}},
		{Path: "analyses/2026-02-01-vuota.md", Date: "2026-02-01", Links: []string{"overview"}},
		{Path: "sources/vecchia.md", Date: "2026-01-01", Links: []string{"topics/scuola"}},
		{Path: "sources/nuova.md", Date: "2026-03-01", Links: []string{"topics/scuola"}},
		{Path: "sources/altro.md", Date: "2026-03-01", Links: []string{"topics/economia"}},
		{Path: "topics/scuola.md"},
	}
	got := AnalysisFindings(pages)
	if len(got) != 2 {
		t.Fatalf("attesi 2 finding, got %v", got)
	}
	if got[0].Level != "info" || got[0].Subject != "analyses/2026-02-01-ok.md" ||
		!strings.Contains(got[0].Msg, "dopo il 2026-02-01") || !strings.Contains(got[0].Msg, "sources/nuova.md") ||
		strings.Contains(got[0].Msg, "altro") || strings.Contains(got[0].Msg, "vecchia") {
		t.Errorf("info: %v", got[0])
	}
	if got[1] != (Finding{"warn", "analyses/2026-02-01-vuota.md", "analisi senza link a fonti, topic o entità"}) {
		t.Errorf("warn: %v", got[1])
	}
}

// TestBuildIndexEn: l'indice segue la lingua corrente; type: vale come tipo:.
func TestBuildIndexEn(t *testing.T) {
	defer i18n.Use(i18n.T())
	i18n.Use(&i18n.EN)
	pages := []Page{
		ParsePage("entities/rome.md", "---\ntitle: Rome\ntype: place\nsummary: city\n---\n"),
		ParsePage("sources/a.md", "---\ntitle: A\nsummary: s\ndate: 2026-01-01\n---\n[[entities/rome|Rome]]\n"),
	}
	got := BuildIndex(pages)
	for _, want := range []string{"# Index\n", "## Entities (1)\n\n### place\n", "· 1 source", "publ. ? · ingested 2026-01-01", "## Analyses (0)\n\n_(no pages)_"} {
		if !strings.Contains(got, want) {
			t.Errorf("manca %q in:\n%s", want, got)
		}
	}
	if !ValidPublished("unknown") || ValidPublished("sconosciuta") {
		t.Error("in inglese la data sconosciuta è \"unknown\"")
	}
}

// TestConfigLanguage: language letta solo dalla sezione [note], commenti a parte.
func TestConfigLanguage(t *testing.T) {
	cases := map[string]string{
		"[note]\nlanguage = \"en\"\n":                       "en",
		"# x\n[note]\nlanguage = 'it' # commento\n":         "it",
		"[format]\nlanguage = \"fr\"\n[note]\ntemplate = 1": "",
		"": "",
	}
	for in, want := range cases {
		if got := ConfigLanguage(in); got != want {
			t.Errorf("ConfigLanguage(%q) = %q, want %q", in, got, want)
		}
	}
}
