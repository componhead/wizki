package core

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path"
	"regexp"
	"strings"

	"wizki/i18n"
)

// Un EPUB è uno zip: META-INF/container.xml indica il pacchetto (.opf), che
// ha i metadati, l'elenco dei file (manifest) e l'ordine di lettura (spine).
// I capitoli sono XHTML: diventano markdown con HTMLToMarkdown.

// ErrEpubDRM: i capitoli dell'EPUB sono cifrati (DRM), il testo non si legge.
var ErrEpubDRM error = i18n.NewErr(func(d *i18n.Dict) string { return d.ErrEpubDRM })

// ErrEpubInvalid: il file non è un EPUB leggibile (zip rotto, pacchetto mancante...).
var ErrEpubInvalid error = i18n.NewErr(func(d *i18n.Dict) string { return d.ErrEpubInvalid })

// maxEpubEntry limita la dimensione decompressa di un singolo file dello zip:
// un libro vero non ci arriva, uno zip malevolo sì.
const maxEpubEntry = 64 << 20

// Book è il contenuto di un EPUB, pronto per la wiki.
type Book struct {
	Title, Date, Language, Publisher string
	Authors                          []string
	// Chapters: il markdown di ogni capitolo, nell'ordine dello spine. Le
	// immagini puntano ai path dentro lo zip (vedi Images).
	Chapters []string
	// Images: path nello zip -> contenuto, delle immagini citate dai capitoli.
	Images map[string][]byte
	// ImageOrder: le chiavi di Images in ordine di apparizione.
	ImageOrder []string
}

type epubContainer struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type epubPackage struct {
	Metadata struct {
		Title     []string `xml:"title"`
		Creator   []string `xml:"creator"`
		Date      []string `xml:"date"`
		Language  []string `xml:"language"`
		Publisher []string `xml:"publisher"`
	} `xml:"metadata"`
	Manifest []struct {
		ID         string `xml:"id,attr"`
		Href       string `xml:"href,attr"`
		MediaType  string `xml:"media-type,attr"`
		Properties string `xml:"properties,attr"`
	} `xml:"manifest>item"`
	Spine struct {
		Toc      string `xml:"toc,attr"`
		Itemrefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

type ncxPoint struct {
	Label   string `xml:"navLabel>text"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Points []ncxPoint `xml:"navPoint"`
}

type epubEncryption struct {
	Refs []struct {
		URI string `xml:"URI,attr"`
	} `xml:"EncryptedData>CipherData>CipherReference"`
}

var (
	mdLink        = regexp.MustCompile(`(!?)\[([^\[\]]*)\]\(([^)]*)\)`)
	startsHeading = regexp.MustCompile(`^\s*#`)
	mdHeading     = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s+(.*)$`)
)

// ParseEpub legge un EPUB dai suoi byte: metadati, capitoli in markdown
// nell'ordine di lettura e immagini citate. ErrEpubDRM se i capitoli sono
// cifrati, ErrEpubInvalid se il file non è un EPUB, ErrEmpty se non c'è testo.
func ParseEpub(content []byte) (Book, error) {
	zr, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return Book{}, fmt.Errorf("%w: %v", ErrEpubInvalid, err)
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	read := func(name string) ([]byte, error) {
		f, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrEpubInvalid, name)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %v", ErrEpubInvalid, name, err)
		}
		defer rc.Close()
		b, err := io.ReadAll(io.LimitReader(rc, maxEpubEntry+1))
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %v", ErrEpubInvalid, name, err)
		}
		if len(b) > maxEpubEntry {
			return nil, fmt.Errorf("%w: %s", ErrEpubInvalid, name)
		}
		return b, nil
	}
	unmarshal := func(name string, v any) error {
		b, err := read(name)
		if err != nil {
			return err
		}
		if err := xml.Unmarshal(b, v); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrEpubInvalid, name, err)
		}
		return nil
	}

	var c epubContainer
	if err := unmarshal("META-INF/container.xml", &c); err != nil {
		return Book{}, err
	}
	if len(c.Rootfiles) == 0 || c.Rootfiles[0].FullPath == "" {
		return Book{}, fmt.Errorf("%w: container.xml", ErrEpubInvalid)
	}
	opfPath := c.Rootfiles[0].FullPath
	var pkg epubPackage
	if err := unmarshal(opfPath, &pkg); err != nil {
		return Book{}, err
	}
	opfDir := path.Dir(opfPath)
	href := map[string]string{} // id -> path nello zip
	var navPath, ncxPath string
	for _, it := range pkg.Manifest {
		p := epubResolve(opfDir, it.Href)
		href[it.ID] = p
		if strings.Contains(" "+it.Properties+" ", " nav ") {
			navPath = p
		}
		if it.ID == pkg.Spine.Toc || it.MediaType == "application/x-dtbncx+xml" {
			ncxPath = p
		}
	}
	var spine []string
	for _, ir := range pkg.Spine.Itemrefs {
		if p, ok := href[ir.IDRef]; ok {
			spine = append(spine, p)
		}
	}

	// DRM: encryption.xml c'è anche per i font offuscati, che non contano;
	// conta solo se tra i file cifrati c'è un capitolo
	if _, ok := files["META-INF/encryption.xml"]; ok {
		var enc epubEncryption
		if err := unmarshal("META-INF/encryption.xml", &enc); err != nil {
			return Book{}, err
		}
		inSpine := set(spine)
		for _, r := range enc.Refs {
			if _, ok := inSpine[epubResolve("", r.URI)]; ok {
				return Book{}, ErrEpubDRM
			}
		}
	}

	labels := epubTOC(read, navPath, ncxPath)
	book := Book{
		Title:     first(pkg.Metadata.Title),
		Date:      first(pkg.Metadata.Date),
		Language:  first(pkg.Metadata.Language),
		Publisher: first(pkg.Metadata.Publisher),
		Authors:   Filter(Map(pkg.Metadata.Creator, strings.TrimSpace), func(s string) bool { return s != "" }),
		Images:    map[string][]byte{},
	}
	for _, ch := range spine {
		b, err := read(ch)
		if err != nil {
			return Book{}, err
		}
		// il <title> dei capitoli è spesso il titolo del libro: lo tolgo,
		// il titolo del capitolo viene dall'indice (labels)
		md := HTMLToMarkdown(htmlTitle.ReplaceAllString(string(b), ""))
		if strings.TrimSpace(md) == "" {
			continue
		}
		// il titolo dall'indice va in testa solo se il capitolo non ce l'ha già:
		// spesso c'è un'epigrafe prima dell'<h1>, e il titolo uscirebbe due volte
		if l := labels[ch]; l != "" && !startsHeading.MatchString(md) && !hasHeading(md, l) {
			md = "# " + l + "\n\n" + md
		}
		md = epubLinks(md, path.Dir(ch), func(img string) bool {
			if _, done := book.Images[img]; done {
				return true
			}
			data, err := read(img)
			if err != nil {
				return false
			}
			book.Images[img] = data
			book.ImageOrder = append(book.ImageOrder, img)
			return true
		})
		book.Chapters = append(book.Chapters, md)
	}
	if len(book.Chapters) == 0 {
		return Book{}, ErrEmpty
	}
	return book, nil
}

// epubResolve risolve un href del pacchetto (relativo a dir, con %XX e
// frammento) nel path dentro lo zip.
func epubResolve(dir, href string) string {
	p := path.Join(dir, strings.ReplaceAll(RefPath(href), "\\", "/"))
	return strings.TrimPrefix(p, "./")
}

// epubLinks sistema i link markdown di un capitolo (che sta in dir): le
// immagini interne puntano al loro path nello zip, se keep le accetta; i link
// interni (ad altri capitoli, alle note) diventano testo semplice, perché
// fuori dall'EPUB non portano da nessuna parte. Gli URL restano.
func epubLinks(md, dir string, keep func(img string) bool) string {
	return mdLink.ReplaceAllStringFunc(md, func(s string) string {
		m := mdLink.FindStringSubmatch(s)
		bang, text, target := m[1], m[2], mdTarget(m[3])
		if hasProto.MatchString(target) {
			return s
		}
		if bang == "" {
			return text
		}
		img := epubResolve(dir, target)
		if strings.HasPrefix(target, "/") || !keep(img) {
			return "" // immagine mancante: niente riferimento rotto
		}
		return "![" + text + "](" + img + ")"
	})
}

// epubTOC: path del capitolo -> titolo, dall'indice EPUB 3 (nav) o, se manca,
// da quello EPUB 2 (NCX). Vale il primo titolo di ogni file.
func epubTOC(read func(string) ([]byte, error), navPath, ncxPath string) map[string]string {
	labels := map[string]string{}
	add := func(dir, src, label string) {
		label = strings.Join(strings.Fields(html.UnescapeString(stripTags(label))), " ")
		if src == "" || label == "" {
			return
		}
		p := epubResolve(dir, src)
		if _, ok := labels[p]; !ok {
			labels[p] = label
		}
	}
	if navPath != "" {
		if b, err := read(navPath); err == nil {
			for _, m := range htmlLink.FindAllStringSubmatch(string(b), -1) {
				add(path.Dir(navPath), m[1]+m[2], m[3])
			}
			if len(labels) > 0 {
				return labels
			}
		}
	}
	if ncxPath != "" {
		if b, err := read(ncxPath); err == nil {
			var ncx struct {
				Points []ncxPoint `xml:"navMap>navPoint"`
			}
			if xml.Unmarshal(b, &ncx) == nil {
				var walk func([]ncxPoint)
				walk = func(ps []ncxPoint) {
					for _, p := range ps {
						add(path.Dir(ncxPath), p.Content.Src, p.Label)
						walk(p.Points)
					}
				}
				walk(ncx.Points)
			}
		}
	}
	return labels
}

// EpubText compone il testo del libro: titolo, metadati, poi i capitoli.
// rewrite: path nello zip di un'immagine -> path nella wiki.
func EpubText(b Book, rewrite map[string]string) string {
	d := i18n.T()
	var sb strings.Builder
	if b.Title != "" {
		sb.WriteString("# " + b.Title + "\n\n")
	}
	meta := []struct{ label, value string }{
		{d.EpubAuthor, strings.Join(b.Authors, "; ")},
		{d.EpubDate, b.Date},
		{d.EpubPublisher, b.Publisher},
		{d.EpubLanguage, b.Language},
	}
	n := 0
	for _, m := range meta {
		if strings.TrimSpace(m.value) != "" {
			sb.WriteString("- " + m.label + ": " + strings.TrimSpace(m.value) + "\n")
			n++
		}
	}
	if n > 0 {
		sb.WriteString("\n")
	}
	for i, ch := range b.Chapters {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(RewriteRefs(ch, rewrite))
	}
	return sb.String()
}

func first(ss []string) string {
	for _, s := range ss {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	return ""
}

// hasHeading dice se il markdown md ha già un titolo (di qualunque livello)
// con il testo label, ignorando maiuscole, spazi ed enfasi.
func hasHeading(md, label string) bool {
	want := headingKey(label)
	for _, m := range mdHeading.FindAllStringSubmatch(md, -1) {
		if headingKey(m[1]) == want {
			return true
		}
	}
	return false
}

// headingKey normalizza il testo di un titolo per confrontarlo: senza enfasi
// markdown (*, _), # finali, spazi ripetuti e maiuscole.
func headingKey(s string) string {
	s = strings.NewReplacer("*", "", "_", "").Replace(s)
	s = strings.TrimRight(strings.TrimSpace(s), "# ")
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
