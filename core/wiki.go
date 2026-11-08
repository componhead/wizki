package core

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// ------------------------------------------------------------------ pages

// Path fissi alla radice della wiki.
const (
	OverviewPath = "overview.md" // panoramica: tesi in evoluzione, mantenuta dall'LLM
	IndexPath    = "index.md"    // catalogo generato da `wizki index`, escluso da zk
)

// PageDirs sono le cartelle delle pagine catalogate nell'indice.
var PageDirs = []string{"sources", "topics", "entities", "analyses"}

// Page è una pagina della wiki ridotta a ciò che serve a indice e lint.
type Page struct {
	Path      string   // relativo alla radice, con estensione: "topics/scuola.md"
	Title     string   // title: del frontmatter, o il primo "# ", o il nome file
	Summary   string   // summary: del frontmatter (una riga)
	Tipo      string   // tipo: delle entità (persona, ente, luogo, ...)
	Date      string   // date: giorno dell'acquisizione (fonti) o della stesura (analisi)
	Published string   // published: delle fonti (data del documento, vedi ValidPublished)
	Links     []string // destinazioni dei wikilink, normalizzate con PageKey
}

// PageKey è il nome di una pagina come appare in un wikilink: path con
// separatori "/" e senza ".md" (es. "topics/scuola").
func PageKey(path string) string {
	return strings.TrimSuffix(filepath.ToSlash(filepath.Clean(path)), ".md")
}

var wikiLink = regexp.MustCompile(`\[\[([^\]|#]+)`)

// WikiLinks restituisce le destinazioni distinte dei wikilink di content,
// normalizzate con PageKey, nell'ordine in cui compaiono.
func WikiLinks(content string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range wikiLink.FindAllStringSubmatch(content, -1) {
		k := PageKey(strings.TrimSpace(m[1]))
		if k != "." && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// ParsePage legge una pagina dal suo contenuto.
func ParsePage(path, content string) Page {
	p := Page{
		Path:      filepath.ToSlash(path),
		Title:     FrontmatterField(content, "title"),
		Summary:   FrontmatterField(content, "summary"),
		Tipo:      FrontmatterField(content, "tipo"),
		Date:      FrontmatterField(content, "date"),
		Published: FrontmatterField(content, "published"),
		Links:     WikiLinks(content),
	}
	if p.Title == "" {
		for l := range strings.SplitSeq(content, "\n") {
			if t, ok := strings.CutPrefix(strings.TrimSpace(l), "# "); ok {
				p.Title = strings.TrimSpace(t)
				break
			}
		}
	}
	if p.Title == "" {
		p.Title = filepath.Base(PageKey(path))
	}
	return p
}

// OverviewPage è la panoramica di una wiki appena creata.
func OverviewPage() []byte {
	return []byte(`---
title: Panoramica
summary:
---

# Panoramica

## Tesi

## Temi principali

## Domande aperte
`)
}

// ------------------------------------------------------------------ index

// pagesIn restituisce le pagine sotto dir/.
func pagesIn(pages []Page, dir string) []Page {
	return Filter(pages, func(p Page) bool { return strings.HasPrefix(p.Path, dir+"/") })
}

// sourceCounts conta, per ogni pagina, quante fonti la linkano.
func sourceCounts(pages []Page) map[string]int {
	n := map[string]int{}
	for _, s := range pagesIn(pages, "sources") {
		for _, l := range s.Links {
			n[l]++
		}
	}
	return n
}

// byTitle ordina le pagine per titolo (senza maiuscole), poi per path.
func byTitle(ps []Page) []Page {
	out := append([]Page{}, ps...)
	sort.Slice(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Title), strings.ToLower(out[j].Title)
		if a != b {
			return a < b
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// indexLine è la riga di una pagina nell'indice: link, riassunto, dettaglio.
func indexLine(p Page, detail string) string {
	sum := p.Summary
	if sum == "" {
		sum = "_(riassunto mancante)_"
	}
	line := fmt.Sprintf("- [[%s|%s]] — %s", PageKey(p.Path), p.Title, sum)
	if detail != "" {
		line += " · " + detail
	}
	return line
}

// sourcesDetail descrive quante fonti linkano una pagina.
func sourcesDetail(n int) string {
	if n == 1 {
		return "1 fonte"
	}
	return fmt.Sprintf("%d fonti", n)
}

// BuildIndex genera index.md: panoramica, topic, entità (per tipo), analisi
// (dalla più recente) e fonti
// (dalla pubblicazione più recente), ognuna con link, riassunto di una riga e dettaglio.
// L'output dipende solo dalle pagine, così il lint può confrontarlo con il file.
func BuildIndex(pages []Page) string {
	var b strings.Builder
	b.WriteString("# Indice\n\n<!-- Generato da `wizki index`: non modificare a mano. -->\n")
	counts := sourceCounts(pages)
	section := func(title string, ps []Page, detail func(Page) string) {
		fmt.Fprintf(&b, "\n## %s (%d)\n\n", title, len(ps))
		if len(ps) == 0 {
			b.WriteString("_(nessuna pagina)_\n")
		}
		for _, p := range ps {
			fmt.Fprintln(&b, indexLine(p, detail(p)))
		}
	}
	for _, p := range pages {
		if p.Path == OverviewPath {
			fmt.Fprintf(&b, "\n## Panoramica\n\n%s\n", indexLine(p, ""))
		}
	}
	count := func(p Page) string { return sourcesDetail(counts[PageKey(p.Path)]) }
	section("Topic", byTitle(pagesIn(pages, "topics")), count)

	entities := pagesIn(pages, "entities")
	fmt.Fprintf(&b, "\n## Entità (%d)\n", len(entities))
	if len(entities) == 0 {
		b.WriteString("\n_(nessuna pagina)_\n")
	}
	byTipo := map[string][]Page{}
	for _, e := range entities {
		t := e.Tipo
		if t == "" {
			t = "altro"
		}
		byTipo[t] = append(byTipo[t], e)
	}
	tipi := make([]string, 0, len(byTipo))
	for t := range byTipo {
		tipi = append(tipi, t)
	}
	sort.Strings(tipi)
	for _, t := range tipi {
		fmt.Fprintf(&b, "\n### %s\n\n", t)
		for _, e := range byTitle(byTipo[t]) {
			fmt.Fprintln(&b, indexLine(e, count(e)))
		}
	}

	analyses := pagesIn(pages, "analyses")
	sort.Slice(analyses, func(i, j int) bool {
		if analyses[i].Date != analyses[j].Date {
			return analyses[i].Date > analyses[j].Date
		}
		return analyses[i].Path < analyses[j].Path
	})
	section("Analisi", analyses, func(p Page) string { return p.Date })

	sources := pagesIn(pages, "sources")
	sort.Slice(sources, func(i, j int) bool {
		a, b := sources[i], sources[j]
		if a.Published != b.Published {
			return a.Published > b.Published
		}
		if a.Date != b.Date {
			return a.Date > b.Date
		}
		return a.Path < b.Path
	})
	section("Fonti", sources, datesDetail)
	return b.String()
}

// datesDetail: data di pubblicazione e di acquisizione di una fonte.
func datesDetail(p Page) string {
	pub := p.Published
	if pub == "" {
		pub = "?"
	}
	return fmt.Sprintf("pubbl. %s · acq. %s", pub, p.Date)
}

// ------------------------------------------------------------------- lint

// SummaryFindings segnala le pagine senza summary:, che nell'indice
// resterebbero senza descrizione.
func SummaryFindings(pages []Page) []Finding {
	var fs []Finding
	for _, p := range pages {
		if p.Summary == "" {
			fs = append(fs, Finding{"warn", p.Path, "summary: mancante nel frontmatter"})
		}
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].Subject < fs[j].Subject })
	return fs
}

// IndexFindings confronta index.md (current, se exists) con quello che
// genererebbe `wizki index` sulle pagine attuali.
func IndexFindings(current string, exists bool, pages []Page) []Finding {
	switch {
	case !exists:
		return []Finding{{"warn", IndexPath, "mancante: lancia `wizki index`"}}
	case current != BuildIndex(pages):
		return []Finding{{"warn", IndexPath, "non aggiornato: lancia `wizki index`"}}
	}
	return nil
}

// NotOverview: la panoramica non ha link in entrata per costruzione (l'indice
// è escluso da zk), quindi non va segnalata come orfana.
func NotOverview(path string) bool { return filepath.ToSlash(path) != OverviewPath }

// --------------------------------------------------------- lint semantico

// PublishedUnknown è il valore di published: per una fonte senza data.
const PublishedUnknown = "sconosciuta"

var publishedRe = regexp.MustCompile(`^\d{4}(-(0[1-9]|1[0-2])(-(0[1-9]|[12]\d|3[01]))?)?$`)

// ValidPublished: published: è AAAA, AAAA-MM, AAAA-MM-GG o "sconosciuta".
// I tre formati di data si confrontano come stringhe.
func ValidPublished(s string) bool { return s == PublishedUnknown || publishedRe.MatchString(s) }

// PublishedFindings segnala le fonti con published: mancante o malformato:
// senza la data del documento non si capisce quali affermazioni sono superate.
func PublishedFindings(pages []Page) []Finding {
	var fs []Finding
	for _, p := range pagesIn(pages, "sources") {
		switch {
		case p.Published == "":
			fs = append(fs, Finding{"warn", p.Path, "published: mancante (data del documento, o " + PublishedUnknown + ")"})
		case !ValidPublished(p.Published):
			fs = append(fs, Finding{"warn", p.Path, fmt.Sprintf("published: %q non valido (AAAA, AAAA-MM, AAAA-MM-GG o %s)", p.Published, PublishedUnknown)})
		}
	}
	return fs
}

// BrokenLinkFindings segnala i wikilink verso pagine che non esistono.
func BrokenLinkFindings(pages []Page) []Finding {
	exists := map[string]bool{}
	for _, p := range pages {
		exists[PageKey(p.Path)] = true
	}
	var fs []Finding
	for _, p := range pages {
		for _, l := range p.Links {
			if !exists[l] {
				fs = append(fs, Finding{"warn", p.Path, fmt.Sprintf("link a pagina inesistente [[%s]]", l)})
			}
		}
	}
	return fs
}

// linkedEitherWay: a linka b o b linka a.
func linkedEitherWay(a, b Page) bool {
	ka, kb := PageKey(a.Path), PageKey(b.Path)
	return slices.Contains(a.Links, kb) || slices.Contains(b.Links, ka)
}

// CrossRefFindings segnala le coppie di topic che compaiono nella stessa fonte
// ma le cui pagine non si linkano: lo schema vuole un "Vedi anche".
func CrossRefFindings(pages []Page) []Finding {
	topics := map[string]Page{}
	for _, t := range pagesIn(pages, "topics") {
		topics[PageKey(t.Path)] = t
	}
	shared := map[[2]string][]string{}
	for _, s := range pagesIn(pages, "sources") {
		var ts []string
		for _, l := range s.Links {
			if _, ok := topics[l]; ok {
				ts = append(ts, l)
			}
		}
		sort.Strings(ts)
		for i := range ts {
			for j := i + 1; j < len(ts); j++ {
				k := [2]string{ts[i], ts[j]}
				shared[k] = append(shared[k], s.Path)
			}
		}
	}
	var fs []Finding
	for k, srcs := range shared {
		if !linkedEitherWay(topics[k[0]], topics[k[1]]) {
			sort.Strings(srcs)
			fs = append(fs, Finding{"warn", k[0] + " ~ " + k[1],
				"compaiono insieme in " + strings.Join(srcs, ", ") + " ma non si linkano in \"Vedi anche\""})
		}
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].Subject < fs[j].Subject })
	return fs
}

// OutOfOrderFindings segnala, per ogni topic o entità, le fonti acquisite
// dopo una fonte pubblicata più di recente: integrando la più vecchia, la
// pagina rischia di presentare come attuali dati già superati.
// Le fonti senza data valida sono ignorate (le segnala PublishedFindings).
func OutOfOrderFindings(pages []Page) []Finding {
	dated := Filter(pagesIn(pages, "sources"), func(s Page) bool {
		return s.Published != PublishedUnknown && publishedRe.MatchString(s.Published) && s.Date != ""
	})
	var fs []Finding
	for _, p := range append(pagesIn(pages, "topics"), pagesIn(pages, "entities")...) {
		key := PageKey(p.Path)
		srcs := Filter(dated, func(s Page) bool { return slices.Contains(s.Links, key) })
		for _, old := range srcs {
			for _, recent := range srcs {
				if old.Date > recent.Date && old.Published < recent.Published {
					fs = append(fs, Finding{"info", p.Path, fmt.Sprintf(
						"%s (pubbl. %s) acquisita dopo %s (pubbl. %s): verifica che i dati più vecchi non sembrino attuali",
						old.Path, old.Published, recent.Path, recent.Published)})
				}
			}
		}
	}
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].Subject < fs[j].Subject })
	return fs
}

// --------------------------------------------------------------- analisi

// AnalysisFindings controlla le analisi salvate dalle query:
//   - warn se un'analisi non linka nessuna fonte, topic o entità: senza
//     riferimenti non si può verificare né aggiornare;
//   - info se, dopo la sua data, sono state acquisite fonti che linkano i
//     topic o le entità su cui si basa: potrebbe essere da rivedere.
func AnalysisFindings(pages []Page) []Finding {
	isRef := func(k string) bool {
		return strings.HasPrefix(k, "sources/") || strings.HasPrefix(k, "topics/") || strings.HasPrefix(k, "entities/")
	}
	sources := pagesIn(pages, "sources")
	var fs []Finding
	for _, a := range pagesIn(pages, "analyses") {
		refs := Filter(a.Links, isRef)
		if len(refs) == 0 {
			fs = append(fs, Finding{"warn", a.Path, "analisi senza link a fonti, topic o entità"})
			continue
		}
		if a.Date == "" {
			continue
		}
		var newer []string
		for _, s := range sources {
			if s.Date <= a.Date || slices.Contains(a.Links, PageKey(s.Path)) {
				continue
			}
			if slices.ContainsFunc(refs, func(r string) bool { return !strings.HasPrefix(r, "sources/") && slices.Contains(s.Links, r) }) {
				newer = append(newer, s.Path)
			}
		}
		if len(newer) > 0 {
			sort.Strings(newer)
			fs = append(fs, Finding{"info", a.Path, fmt.Sprintf(
				"fonti acquisite dopo il %s sugli stessi topic/entità: %s; da rivedere?", a.Date, strings.Join(newer, ", "))})
		}
	}
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].Subject < fs[j].Subject })
	return fs
}
