// Package core contiene solo funzioni pure: nessun I/O, nessun orologio,
// nessun processo esterno. Tutto ciò che tocca il mondo sta in package fx.
package core

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------- helpers

func Map[A, B any](xs []A, f func(A) B) []B {
	out := make([]B, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

func Filter[A any](xs []A, keep func(A) bool) []A {
	out := make([]A, 0, len(xs))
	for _, x := range xs {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

func FlatMap[A, B any](xs []A, f func(A) []B) []B {
	var out []B
	for _, x := range xs {
		out = append(out, f(x)...)
	}
	return out
}

func set(xs []string) map[string]struct{} {
	m := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		m[x] = struct{}{}
	}
	return m
}

// Diff restituisce a \ b, ordinato e senza duplicati.
func Diff(a, b []string) []string {
	inB := set(b)
	seen := map[string]struct{}{}
	out := Filter(a, func(x string) bool {
		if _, ok := inB[x]; ok {
			return false
		}
		if _, ok := seen[x]; ok {
			return false
		}
		seen[x] = struct{}{}
		return true
	})
	sort.Strings(out)
	return out
}

func unionSize(a, b []string) int {
	return len(set(append(append([]string{}, a...), b...)))
}

// UnderDir è un predicato: il path sta sotto la directory (relativa) indicata.
func UnderDir(dir string) func(string) bool {
	prefix := strings.TrimSuffix(dir, "/") + "/"
	return func(p string) bool { return strings.HasPrefix(filepath.ToSlash(p), prefix) }
}

// ------------------------------------------------------------------- slug

var accents = strings.NewReplacer(
	"à", "a", "á", "a", "è", "e", "é", "e", "ì", "i", "í", "i",
	"ò", "o", "ó", "o", "ù", "u", "ú", "u",
)

// Slug: minuscolo, senza accenti, solo [a-z0-9] separati da un trattino.
func Slug(s string) string {
	s = accents.Replace(strings.ToLower(strings.TrimSpace(s)))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// ----------------------------------------------------------------- extract

// Step è una somma chiusa: il piano è dato, l'esecuzione sta altrove.
type Step interface{ isStep() }

type Mkdir struct{ Path string }
type Move struct{ From, To string }
type Extract struct{ PDF, Out string }

func (Mkdir) isStep()   {}
func (Move) isStep()    {}
func (Extract) isStep() {}

// IsPDF: il nome file ha estensione .pdf (maiuscole comprese).
func IsPDF(name string) bool { return strings.EqualFold(filepath.Ext(name), ".pdf") }

// PlanExtract: il PDF diventa immutabile in raw/pdf/, il testo in raw/text/.
func PlanExtract(root, pdfPath string) ([]Step, error) {
	base := filepath.Base(pdfPath)
	if !IsPDF(base) {
		return nil, fmt.Errorf("%q non è un PDF", base)
	}
	stem := Slug(strings.TrimSuffix(base, filepath.Ext(base)))
	if stem == "" {
		return nil, fmt.Errorf("nome file %q non utilizzabile", base)
	}
	pdfDir := filepath.Join(root, "raw", "pdf")
	txtDir := filepath.Join(root, "raw", "text")
	dst := filepath.Join(pdfDir, stem+".pdf")
	return []Step{
		Mkdir{pdfDir},
		Mkdir{txtDir},
		Move{From: pdfPath, To: dst},
		Extract{PDF: dst, Out: filepath.Join(txtDir, stem+".md")},
	}, nil
}

// ------------------------------------------------------------------- log

var logKinds = map[string]bool{"ingest": true, "query": true, "lint": true}

// LogLine produce una riga nel formato "## [YYYY-MM-DD] kind | titolo",
// analizzabile con `grep "^## \[" log.md | tail -5`.
func LogLine(day time.Time, kind, title string) (string, error) {
	if !logKinds[kind] {
		return "", fmt.Errorf("tipo di log %q sconosciuto (ingest|query|lint)", kind)
	}
	t := strings.Join(strings.Fields(title), " ")
	if t == "" {
		return "", fmt.Errorf("titolo vuoto")
	}
	return fmt.Sprintf("## [%s] %s | %s\n", day.Format("2006-01-02"), kind, t), nil
}

// ------------------------------------------------------------------ lint

type Finding struct{ Level, Subject, Msg string } // Level: "warn" | "info"

// TopicView: per un topic, le fonti che hanno il tag e quelle che linkano l'hub.
type TopicView struct {
	Topic          string
	Tagged, Linked []string
}

// TopicFindings controlla l'allineamento tag/link e la consistenza delle fonti.
func TopicFindings(v TopicView) []Finding {
	var fs []Finding
	for _, p := range Diff(v.Tagged, v.Linked) {
		fs = append(fs, Finding{"warn", v.Topic, fmt.Sprintf("%s ha il tag ma non linka [[topics/%s]]", p, v.Topic)})
	}
	for _, p := range Diff(v.Linked, v.Tagged) {
		fs = append(fs, Finding{"warn", v.Topic, fmt.Sprintf("%s linka [[topics/%s]] ma non ha il tag", p, v.Topic)})
	}
	switch unionSize(v.Tagged, v.Linked) {
	case 0:
		fs = append(fs, Finding{"info", v.Topic, "topic senza fonti"})
	case 1:
		fs = append(fs, Finding{"info", v.Topic, "una sola fonte"})
	}
	return fs
}

// NearDuplicates segnala topic quasi uguali (plurali, sinonimi ortografici).
func NearDuplicates(topics []string) []Finding {
	ts := append([]string{}, topics...)
	sort.Strings(ts)
	var fs []Finding
	for i := range ts {
		for j := i + 1; j < len(ts); j++ {
			limit := 1
			if min(len(ts[i]), len(ts[j])) >= 6 {
				limit = 2
			}
			if lev(ts[i], ts[j]) <= limit {
				fs = append(fs, Finding{"warn", ts[i] + " ~ " + ts[j], "possibile plurale o sinonimo: unificare?"})
			}
		}
	}
	return fs
}

func Orphans(paths []string) []Finding {
	return Map(paths, func(p string) Finding {
		return Finding{"warn", p, "nessun link in entrata"}
	})
}

func HasWarn(fs []Finding) bool {
	return len(Filter(fs, func(f Finding) bool { return f.Level == "warn" })) > 0
}

func lev(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// -------------------------------------------------------------------- id

// IDLen è la lunghezza dell'id di una nota-fonte: prefisso dello SHA-256 del PDF.
const IDLen = 8

// ShortID riduce uno SHA-256 esadecimale all'id della nota-fonte.
func ShortID(sum string) string {
	if len(sum) < IDLen {
		return sum
	}
	return sum[:IDLen]
}

// FindDuplicate cerca sum tra gli hash noti (path -> SHA-256 completo) e
// restituisce il primo path, in ordine alfabetico, con lo stesso contenuto.
func FindDuplicate(sum string, known map[string]string) (string, bool) {
	paths := make([]string, 0, len(known))
	for p := range known {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if known[p] == sum {
			return p, true
		}
	}
	return "", false
}

// FrontmatterField legge un campo scalare dal frontmatter YAML (tra due righe
// "---" in testa al file). Restituisce "" se il frontmatter o il campo mancano.
func FrontmatterField(content, key string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "---" {
			return ""
		}
		k, v, ok := strings.Cut(l, ":")
		if ok && strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return ""
}

// IDFindings controlla gli id delle note-fonte (path -> id): segnala id
// mancanti e id condivisi da più note, cioè lo stesso PDF acquisito due volte.
func IDFindings(ids map[string]string) []Finding {
	byID := map[string][]string{}
	var fs []Finding
	for p, id := range ids {
		if id == "" {
			fs = append(fs, Finding{"warn", p, "id mancante nel frontmatter"})
			continue
		}
		byID[id] = append(byID[id], p)
	}
	for id, ps := range byID {
		if len(ps) > 1 {
			sort.Strings(ps)
			fs = append(fs, Finding{"warn", "id " + id, "stesso PDF in più note: " + strings.Join(ps, ", ")})
		}
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].Subject < fs[j].Subject })
	return fs
}
