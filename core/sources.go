package core

import (
	"fmt"
	"html"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"wizki/i18n"
)

// ---------------------------------------------------------------- tipi

// Kind è il tipo di una fonte acquisibile dalla inbox.
type Kind string

const (
	KindPDF   Kind = "pdf"   // testo con pdftotext, OCR se scansione
	KindDoc   Kind = "doc"   // markdown, testo, HTML: il testo è il contenuto stesso
	KindImage Kind = "image" // immagine: link + OCR facoltativo
	KindData  Kind = "data"  // CSV, TSV, JSON: il testo è il contenuto stesso
	KindEpub  Kind = "epub"  // EPUB: capitoli in markdown, immagini in raw/assets/<stem>/
)

var kindByExt = map[string]Kind{
	".pdf": KindPDF,
	".md":  KindDoc, ".markdown": KindDoc, ".txt": KindDoc, ".html": KindDoc, ".htm": KindDoc,
	".png": KindImage, ".jpg": KindImage, ".jpeg": KindImage, ".webp": KindImage,
	".gif": KindImage, ".tif": KindImage, ".tiff": KindImage,
	".csv": KindData, ".tsv": KindData, ".json": KindData,
	".epub": KindEpub,
}

// rawDirOf: la cartella sotto raw/ dove finisce l'originale di ogni tipo.
var rawDirOf = map[Kind]string{KindPDF: "pdf", KindDoc: "doc", KindImage: "assets", KindData: "data", KindEpub: "epub"}

// SourceKind restituisce il tipo di fonte dal nome del file (estensione,
// maiuscole comprese); ok è false per i tipi non supportati.
func SourceKind(name string) (Kind, bool) {
	k, ok := kindByExt[strings.ToLower(filepath.Ext(name))]
	return k, ok
}

// SupportedExts elenca le estensioni accettate, in ordine.
func SupportedExts() []string {
	exts := make([]string, 0, len(kindByExt))
	for e := range kindByExt {
		exts = append(exts, e)
	}
	sort.Strings(exts)
	return exts
}

// RawDirs sono le cartelle degli originali, relative alla radice: in quelle
// si cercano i duplicati (solo i file al primo livello: le sottocartelle di
// raw/assets/ sono le immagini allegate ad altre fonti).
func RawDirs() []string {
	return []string{
		filepath.Join("raw", "pdf"), filepath.Join("raw", "doc"),
		filepath.Join("raw", "assets"), filepath.Join("raw", "data"),
		filepath.Join("raw", "epub"),
	}
}

func isHTML(name string) bool {
	e := strings.ToLower(filepath.Ext(name))
	return e == ".html" || e == ".htm"
}

// ErrEmpty: una fonte di testo o di dati senza contenuto.
var ErrEmpty error = i18n.NewErr(func(d *i18n.Dict) string { return d.ErrEmpty })

// ---------------------------------------------------------------- passi

// ImageText scrive in Out il riferimento all'immagine (Header) e, se Lang non
// è vuoto, il testo letto da Tesseract.
type ImageText struct{ Image, Out, Header, Lang string }

func (ImageText) isStep() {}

// SourceInput descrive una fonte da acquisire. Content (solo doc e data) e
// Assets sono letti dal disco prima di pianificare.
type SourceInput struct {
	Path    string
	Content []byte
	// Assets: riferimento come scritto nel testo -> path su disco di
	// un'immagine locale esistente nella inbox, da spostare con la fonte.
	Assets map[string]string
	Lang   string // lingua OCR per le immagini; "" = niente OCR
}

// TextPath è il path del testo estratto di una fonte non PDF:
// raw/text/<stem>-<estensione>.md, così non collide con quello di un PDF
// omonimo (raw/text/<stem>.md).
func TextPath(root, stem, ext string) string {
	return filepath.Join(root, "raw", "text", stem+"-"+strings.TrimPrefix(strings.ToLower(ext), ".")+".md")
}

// PlanSource pianifica l'acquisizione di una fonte qualsiasi: i PDF passano
// da PlanExtract, gli altri tipi diventano originale in raw/<tipo>/ e testo in
// raw/text/. Le immagini locali di un doc vanno in raw/assets/<stem>/ e i loro
// riferimenti nel testo estratto vengono riscritti verso il nuovo path.
func PlanSource(root string, in SourceInput) ([]Step, error) {
	base := filepath.Base(in.Path)
	kind, ok := SourceKind(base)
	if !ok {
		return nil, fmt.Errorf(i18n.T().Unsupported, base, strings.Join(SupportedExts(), " "))
	}
	if kind == KindPDF {
		return PlanExtract(root, in.Path)
	}
	ext := filepath.Ext(base)
	stem := Slug(strings.TrimSuffix(base, ext))
	if stem == "" {
		return nil, fmt.Errorf(i18n.T().BadFileName, base)
	}
	rawDir := filepath.Join(root, "raw", rawDirOf[kind])
	txtDir := filepath.Join(root, "raw", "text")
	dst := filepath.Join(rawDir, stem+strings.ToLower(ext))
	out := TextPath(root, stem, ext)
	rel := func(p string) string { // path relativo alla radice, con "/"
		r, err := filepath.Rel(root, p)
		if err != nil {
			return filepath.ToSlash(p)
		}
		return filepath.ToSlash(r)
	}
	steps := []Step{Mkdir{rawDir}, Mkdir{txtDir}, Move{From: in.Path, To: dst}}

	switch kind {
	case KindImage:
		header := fmt.Sprintf("![%s](%s)\n", stem, rel(dst))
		return append(steps, ImageText{Image: dst, Out: out, Header: header, Lang: in.Lang}), nil
	case KindData:
		if strings.TrimSpace(string(in.Content)) == "" {
			return nil, fmt.Errorf("%w: %s", ErrEmpty, base)
		}
		return append(steps, WriteFile{Path: out, Data: in.Content}), nil
	case KindEpub:
		book, err := ParseEpub(in.Content)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", err, base)
		}
		rewrite := map[string]string{}
		if len(book.ImageOrder) > 0 {
			assetDir := filepath.Join(root, "raw", "assets", stem)
			steps = append(steps, Mkdir{assetDir})
			used := map[string]bool{}
			for _, img := range book.ImageOrder {
				to := filepath.Join(assetDir, uniqueName(path.Base(img), used))
				steps = append(steps, WriteFile{Path: to, Data: book.Images[img]})
				rewrite[img] = rel(to)
			}
		}
		return append(steps, WriteFile{Path: out, Data: []byte(EpubText(book, rewrite))}), nil
	}

	text := DocText(base, in.Content)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("%w: %s", ErrEmpty, base)
	}
	if len(in.Assets) > 0 {
		assetDir := filepath.Join(root, "raw", "assets", stem)
		steps = append(steps, Mkdir{assetDir})
		newPath := map[string]string{} // path su disco -> nuovo path
		used := map[string]bool{}
		refs := make([]string, 0, len(in.Assets))
		for r := range in.Assets {
			refs = append(refs, r)
		}
		sort.Strings(refs)
		rewrite := map[string]string{}
		for _, r := range refs {
			disk := in.Assets[r]
			if _, done := newPath[disk]; !done {
				name := uniqueName(filepath.Base(disk), used)
				to := filepath.Join(assetDir, name)
				newPath[disk] = to
				steps = append(steps, Move{From: disk, To: to})
			}
			rewrite[r] = rel(newPath[disk])
		}
		text = RewriteRefs(text, rewrite)
	}
	return append(steps, WriteFile{Path: out, Data: []byte(text)}), nil
}

// uniqueName dà a un'immagine allegata un nome sicuro nei link (slug,
// estensione minuscola: niente spazi) ed evita che due immagini con lo stesso
// nome (da cartelle diverse) finiscano sullo stesso path: la seconda diventa
// nome-2.ext, e così via.
func uniqueName(name string, used map[string]bool) string {
	ext := strings.ToLower(filepath.Ext(name))
	stem := Slug(strings.TrimSuffix(name, filepath.Ext(name)))
	if stem == "" {
		stem = "img"
	}
	out := stem + ext
	for i := 2; used[out]; i++ {
		out = fmt.Sprintf("%s-%d%s", stem, i, ext)
	}
	used[out] = true
	return out
}

// ----------------------------------------------------------- riferimenti

var (
	mdImage  = regexp.MustCompile(`!\[[^\]]*\]\(([^)]*)\)`)
	wikiEmb  = regexp.MustCompile(`!\[\[([^\]|#]+)((?:[|#][^\]]*)?)\]\]`)
	htmlImg  = regexp.MustCompile(`(?i)<img\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']`)
	hasProto = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// mdTarget estrae la destinazione dal contenuto di (...) di un'immagine
// markdown: "<a b.png>" oppure "a.png \"titolo\"".
func mdTarget(inner string) string {
	inner = strings.TrimSpace(inner)
	if strings.HasPrefix(inner, "<") {
		if end := strings.Index(inner, ">"); end > 0 {
			return inner[1:end]
		}
	}
	if i := strings.IndexAny(inner, " \t"); i >= 0 {
		return inner[:i]
	}
	return inner
}

// DocText è il testo estratto di un doc: l'HTML diventa markdown, il resto
// resta com'è.
func DocText(name string, content []byte) string {
	if isHTML(name) {
		return HTMLToMarkdown(string(content))
	}
	return string(content)
}

// ImageRefs restituisce i riferimenti a immagini locali di un testo
// markdown (![](x), ![[x]], <img src="x">), come scritti, senza duplicati e in
// ordine di apparizione. Esclusi URL (http:, data:, ...), path assoluti e ancore.
func ImageRefs(text string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(r string) {
		r = strings.TrimSpace(r)
		if r == "" || seen[r] || hasProto.MatchString(r) || strings.HasPrefix(r, "/") || strings.HasPrefix(r, "#") {
			return
		}
		seen[r] = true
		out = append(out, r)
	}
	for _, m := range mdImage.FindAllStringSubmatch(text, -1) {
		add(mdTarget(m[1]))
	}
	for _, m := range wikiEmb.FindAllStringSubmatch(text, -1) {
		add(m[1])
	}
	for _, m := range htmlImg.FindAllStringSubmatch(text, -1) {
		add(m[1])
	}
	return out
}

// RemoteImageCount conta le immagini remote (http/https) di un testo: non
// vengono scaricate, restano come URL.
func RemoteImageCount(text string) int {
	n := 0
	for _, m := range mdImage.FindAllStringSubmatch(text, -1) {
		if t := strings.ToLower(mdTarget(m[1])); strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://") {
			n++
		}
	}
	return n
}

// RefPath converte un riferimento in un path relativo: senza query né
// frammento, con le sequenze %XX decodificate.
func RefPath(ref string) string {
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	if p, err := url.PathUnescape(ref); err == nil {
		ref = p
	}
	return filepath.FromSlash(ref)
}

// RewriteRefs sostituisce le destinazioni delle immagini secondo m
// (riferimento come scritto -> nuovo path), lasciando intatto il resto.
func RewriteRefs(text string, m map[string]string) string {
	text = mdImage.ReplaceAllStringFunc(text, func(s string) string {
		inner := mdImage.FindStringSubmatch(s)[1]
		if to, ok := m[mdTarget(inner)]; ok {
			open := strings.LastIndex(s[:len(s)-len(inner)-1], "(")
			return s[:open+1] + strings.Replace(inner, mdTarget(inner), to, 1) + ")"
		}
		return s
	})
	text = wikiEmb.ReplaceAllStringFunc(text, func(s string) string {
		sm := wikiEmb.FindStringSubmatch(s)
		if to, ok := m[strings.TrimSpace(sm[1])]; ok {
			return "![[" + to + sm[2] + "]]"
		}
		return s
	})
	return htmlImg.ReplaceAllStringFunc(text, func(s string) string {
		src := htmlImg.FindStringSubmatch(s)[1]
		if to, ok := m[strings.TrimSpace(src)]; ok {
			return strings.Replace(s, src, to, 1)
		}
		return s
	})
}

// ------------------------------------------------------------------ HTML

var (
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	htmlDrop    = Map([]string{"script", "style", "noscript", "svg", "template", "iframe", "head"}, func(t string) *regexp.Regexp {
		return regexp.MustCompile(`(?is)<` + t + `\b.*?</` + t + `\s*>`)
	})
	htmlTitle   = regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title\s*>`)
	htmlHeading = regexp.MustCompile(`(?is)<h([1-6])\b[^>]*>(.*?)</h[1-6]\s*>`)
	htmlImgTag  = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	htmlAttr    = regexp.MustCompile(`(?is)\b(src|alt)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	htmlLink    = regexp.MustCompile(`(?is)<a\b[^>]*?\bhref\s*=\s*(?:"([^"]*)"|'([^']*)')[^>]*>(.*?)</a\s*>`)
	htmlStrong  = regexp.MustCompile(`(?is)</?(strong|b)\b[^>]*>`)
	htmlEm      = regexp.MustCompile(`(?is)</?(em|i)\b[^>]*>`)
	htmlLi      = regexp.MustCompile(`(?is)<li\b[^>]*>`)
	htmlPre     = regexp.MustCompile(`(?is)</?pre\b[^>]*>`)
	htmlBr      = regexp.MustCompile(`(?is)<br\s*/?>`)
	htmlCell    = regexp.MustCompile(`(?is)<t[dh]\b[^>]*>`)
	htmlBlock   = regexp.MustCompile(`(?is)</?(p|div|section|article|main|header|footer|aside|nav|ul|ol|table|thead|tbody|tr|blockquote|figure|figcaption|dl|dt|dd|hr)\b[^>]*>`)
	htmlAnyTag  = regexp.MustCompile(`(?s)<[^>]*>`)
	spaces      = regexp.MustCompile(`[ \t\x{a0}]+`)
	blankLines  = regexp.MustCompile(`\n{3,}`)
)

// innerOf restituisce il contenuto tra il primo <tag ...> e l'ultimo </tag>.
func innerOf(s, tag string) (string, bool) {
	lower := strings.ToLower(s)
	start := strings.Index(lower, "<"+tag)
	end := strings.LastIndex(lower, "</"+tag)
	if start < 0 || end <= start {
		return "", false
	}
	open := strings.Index(lower[start:], ">")
	if open < 0 || start+open+1 > end {
		return "", false
	}
	return s[start+open+1 : end], true
}

// stripTags toglie i tag e normalizza gli spazi di un frammento inline.
func stripTags(s string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(strings.ReplaceAll(htmlAnyTag.ReplaceAllString(s, ""), "\n", " "), " "))
}

// HTMLToMarkdown converte una pagina HTML in markdown leggibile: titolo,
// intestazioni, paragrafi, elenchi, link e immagini. Se c'è <article> o
// <main> tiene solo quello (scarta menu e piè di pagina). È una conversione
// approssimata, con la sola libreria standard: le tabelle diventano righe di
// testo e la formattazione del codice si perde.
func HTMLToMarkdown(s string) string {
	title := ""
	if m := htmlTitle.FindStringSubmatch(s); m != nil {
		title = html.UnescapeString(stripTags(m[1]))
	}
	s = htmlComment.ReplaceAllString(s, "")
	for _, re := range htmlDrop {
		s = re.ReplaceAllString(s, "")
	}
	for _, tag := range []string{"article", "main", "body"} {
		if inner, ok := innerOf(s, tag); ok {
			s = inner
			break
		}
	}
	s = htmlHeading.ReplaceAllStringFunc(s, func(h string) string {
		m := htmlHeading.FindStringSubmatch(h)
		return "\n\n" + strings.Repeat("#", int(m[1][0]-'0')) + " " + stripTags(m[2]) + "\n\n"
	})
	s = htmlImgTag.ReplaceAllStringFunc(s, func(tag string) string {
		attrs := map[string]string{}
		for _, a := range htmlAttr.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(a[1])] = a[2] + a[3]
		}
		if attrs["src"] == "" {
			return ""
		}
		return "![" + attrs["alt"] + "](" + attrs["src"] + ")"
	})
	s = htmlLink.ReplaceAllStringFunc(s, func(a string) string {
		m := htmlLink.FindStringSubmatch(a)
		href, text := m[1]+m[2], stripTags(m[3])
		if text == "" || href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return text
		}
		return "[" + text + "](" + href + ")"
	})
	s = htmlStrong.ReplaceAllString(s, "**")
	s = htmlEm.ReplaceAllString(s, "_")
	s = htmlLi.ReplaceAllString(s, "\n- ")
	s = htmlPre.ReplaceAllString(s, "\n```\n")
	s = htmlBr.ReplaceAllString(s, "\n")
	s = htmlCell.ReplaceAllString(s, " ")
	s = htmlBlock.ReplaceAllString(s, "\n\n")
	s = html.UnescapeString(htmlAnyTag.ReplaceAllString(s, ""))

	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(spaces.ReplaceAllString(l, " "))
	}
	body := strings.TrimSpace(blankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
	if title != "" && !strings.HasPrefix(body, "# ") {
		body = "# " + title + "\n\n" + body
	}
	if body == "" {
		return ""
	}
	return body + "\n"
}

// --------------------------------------------------------------- inbox

// InboxOrder ordina le fonti della inbox: prima i documenti (che possono
// portarsi dietro le loro immagini), poi le immagini; a parità, per nome.
// attached sono le immagini referenziate da un doc della inbox: si acquisiscono
// con quel doc, quindi non compaiono nell'elenco.
func InboxOrder(names []string, attached map[string]bool) []string {
	var out []string
	for _, n := range names {
		if _, ok := SourceKind(n); ok && !attached[n] && !strings.HasPrefix(n, ".") {
			out = append(out, n)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ki, _ := SourceKind(out[i])
		kj, _ := SourceKind(out[j])
		if (ki == KindImage) != (kj == KindImage) {
			return kj == KindImage
		}
		return out[i] < out[j]
	})
	return out
}
