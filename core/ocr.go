package core

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// OCR è il passo di ripiego per i PDF senza livello di testo: rasterizza le
// pagine (pdftoppm) e le legge con Tesseract. Non produce un PDF ricercabile,
// solo il testo che serve alla wiki.
type OCR struct{ PDF, Out, Lang string }

func (OCR) isStep() {}

// FallbackOCR è il passo OCR equivalente a un'estrazione che non ha trovato testo.
func FallbackOCR(e Extract, lang string) OCR {
	return OCR{PDF: e.PDF, Out: e.Out, Lang: lang}
}

// pageNo legge il numero di pagina da nomi come ".../p-12.png".
func pageNo(name string) int {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	n, err := strconv.Atoi(base[strings.LastIndex(base, "-")+1:])
	if err != nil {
		return 0
	}
	return n
}

// SortPages ordina le immagini per numero di pagina (p-2 prima di p-10),
// qualunque sia il riempimento con zeri scelto da pdftoppm.
func SortPages(names []string) []string {
	out := append([]string{}, names...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := pageNo(out[i]), pageNo(out[j])
		if a != b {
			return a < b
		}
		return out[i] < out[j]
	})
	return out
}

// JoinPages unisce i testi delle pagine separandoli con una riga vuota e
// scartando le pagine vuote. Nessun testo: stringa vuota.
func JoinPages(texts []string) string {
	pages := Filter(Map(texts, strings.TrimSpace), func(s string) bool { return s != "" })
	if len(pages) == 0 {
		return ""
	}
	return strings.Join(pages, "\n\n") + "\n"
}

// ParseLangs estrae i codici lingua dall'output di `tesseract --list-langs`.
func ParseLangs(listOutput string) []string {
	return Filter(Map(strings.Split(listOutput, "\n"), strings.TrimSpace), func(l string) bool {
		return l != "" && !strings.HasPrefix(l, "List of available languages")
	})
}

// MissingLangs: le lingue richieste (anche "ita+eng") che non sono installate.
func MissingLangs(available []string, lang string) []string {
	have := set(available)
	return Filter(strings.Split(lang, "+"), func(l string) bool {
		_, ok := have[l]
		return !ok
	})
}
