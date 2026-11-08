// wizki: strumento di supporto per un notebook zk (pattern "LLM Wiki").
//
//	wizki [-root library] [-ocr-lang ita] [-no-ocr] extract <file.pdf>
//	                                              sposta in raw/pdf ed estrae il testo (OCR se scansione);
//	                                              stampa il path del testo e, sulla riga dopo, l'id (SHA-256, 8 hex)
//	exit: 0 ok, 1 errore, 2 uso, 3 nessun testo (anche con OCR), 4 OCR non disponibile,
//	      5 PDF già acquisito (stesso contenuto di un file in raw/pdf, anche con nome diverso)
//	wizki [-root library] log <ingest|query|lint> <titolo>
//	wizki [-root library] lint                     controlli di salute (exit 1 se ci sono warn)
//	wizki slug <testo>
//	wizki init <nome>                              crea la wiki ./<nome>/ (cartelle, .zk/, log.md,
//	                                              .gitignore, git init -b master); non sovrascrive nulla
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"wizki/core"
	"wizki/fx"
)

// zkConfig è la configurazione zk di riferimento, copiata da `wizki init`
// in ogni nuova wiki: l'unica fonte resta library/.zk/ di questo repo.
//
//go:embed library/.zk/config.toml library/.zk/templates/source.md library/.zk/templates/topic.md
var zkConfig embed.FS

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("wizki", flag.ContinueOnError)
	fs.SetOutput(errw)
	root := fs.String("root", envOr("WIKI_ROOT", "library"), "radice del notebook zk")
	ocrLang := fs.String("ocr-lang", envOr("WIKI_OCR_LANG", "ita"), "lingua/e Tesseract (es. ita, ita+eng)")
	noOCR := fs.Bool("no-ocr", false, "non tentare l'OCR sulle scansioni")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(errw, "uso: wizki [-root dir] init|extract|log|lint|slug ...")
		return 2
	}

	var err error
	code := 0
	switch rest[0] {
	case "init":
		err = need(rest, 2, "wizki init <nome>")
		if err == nil {
			err = initWiki(ctx, rest[1], out)
		}
	case "extract":
		err = need(rest, 2, "wizki extract <file.pdf>")
		if err == nil {
			err = extract(ctx, *root, rest[1], *ocrLang, *noOCR, out, errw)
		}
	case "log":
		err = need(rest, 3, "wizki log <ingest|query|lint> <titolo>")
		if err == nil {
			err = logEntry(*root, rest[1], strings.Join(rest[2:], " "), out)
		}
	case "lint":
		code, err = lint(ctx, *root, out)
	case "slug":
		err = need(rest, 2, "wizki slug <testo>")
		if err == nil {
			fmt.Fprintln(out, core.Slug(strings.Join(rest[1:], " ")))
		}
	default:
		err = fmt.Errorf("comando sconosciuto %q", rest[0])
	}
	if err != nil {
		fmt.Fprintln(errw, "errore:", err)
		if errors.Is(err, fx.ErrNoText) {
			return 3
		}
		if errors.Is(err, fx.ErrOCRUnavailable) {
			return 4
		}
		if errors.Is(err, fx.ErrDuplicate) {
			return 5
		}
		return 1
	}
	return code
}

func extract(ctx context.Context, root, pdf, lang string, noOCR bool, out, errw io.Writer) error {
	steps, err := core.PlanExtract(root, pdf)
	if err != nil {
		return err
	}
	// controllo sul contenuto, prima di toccare raw/: il nome del file non conta
	sum, err := fx.HashFile(pdf)
	if err != nil {
		return err
	}
	known, err := fx.HashPDFs(filepath.Join(root, "raw", "pdf"))
	if err != nil {
		return err
	}
	if dup, ok := core.FindDuplicate(sum, known); ok {
		return fmt.Errorf("%w: stesso contenuto di %s (id %s)", fx.ErrDuplicate, dup, core.ShortID(sum))
	}
	rollback := func(upTo int) {
		for j := upTo; j >= 0; j-- {
			_ = fx.Undo(steps[j])
		}
	}
	for i, s := range steps {
		err := fx.Exec(ctx, s)
		triedOCR := false
		if errors.Is(err, fx.ErrNoText) && !noOCR {
			// scansione: ripiego su OCR; se non è disponibile, nulla è stato toccato oltre al move
			if cerr := fx.CheckOCR(ctx, lang); cerr != nil {
				rollback(i)
				return cerr
			}
			fmt.Fprintf(errw, "nessun testo nel PDF: provo l'OCR (%s)…\n", lang)
			ocr := core.FallbackOCR(s.(core.Extract), lang)
			steps[i] = ocr
			triedOCR = true
			err = fx.Exec(ctx, ocr)
		}
		if err != nil {
			// la scansione deve restare in inbox sia se non c'è testo, sia se l'OCR fallisce
			if triedOCR || errors.Is(err, fx.ErrNoText) {
				rollback(i)
			}
			return err
		}
	}
	var dest string
	switch last := steps[len(steps)-1].(type) {
	case core.Extract:
		dest = last.Out
	case core.OCR:
		dest = last.Out
	}
	fmt.Fprintln(out, dest)
	fmt.Fprintln(out, core.ShortID(sum))
	return nil
}

// initWiki crea la wiki <name>/ nella cartella corrente. È idempotente: ciò che
// esiste già viene lasciato com'è e segnalato, così si può rilanciare per
// completare una wiki a cui manca qualche pezzo.
func initWiki(ctx context.Context, name string, out io.Writer) error {
	files, err := zkFiles()
	if err != nil {
		return err
	}
	steps, err := core.PlanInit(name, files)
	if err != nil {
		return err
	}
	for _, s := range steps {
		err := fx.Exec(ctx, s)
		if err != nil && !errors.Is(err, fx.ErrExists) {
			return err
		}
		status := "creato:"
		if err != nil {
			status = "già presente:"
		}
		switch s := s.(type) {
		case core.WriteFile:
			fmt.Fprintln(out, status, s.Path)
		case core.GitInit:
			fmt.Fprintln(out, status, filepath.Join(s.Dir, ".git"))
		}
	}
	fmt.Fprintf(out, "wiki %q pronta: usa wizki -root %s (o WIKI_ROOT=%s)\n", name, name, name)
	return nil
}

// zkFiles legge la configurazione zk incorporata: path relativo alla radice
// della wiki (es. .zk/config.toml) -> contenuto.
func zkFiles() (map[string][]byte, error) {
	files := map[string][]byte{}
	for _, p := range []string{"config.toml", "templates/source.md", "templates/topic.md"} {
		b, err := zkConfig.ReadFile("library/.zk/" + p)
		if err != nil {
			return nil, err
		}
		files[filepath.Join(".zk", filepath.FromSlash(p))] = b
	}
	return files, nil
}

func logEntry(root, kind, title string, out io.Writer) error {
	line, err := core.LogLine(fx.Today(), kind, title)
	if err != nil {
		return err
	}
	if err := fx.AppendLog(filepath.Join(root, "log.md"), line); err != nil {
		return err
	}
	fmt.Fprint(out, line)
	return nil
}

// lint: letture da zk, poi un calcolo puro sui risultati.
func lint(ctx context.Context, root string, out io.Writer) (int, error) {
	topics, err := fx.TopicNames(root)
	if err != nil {
		return 1, err
	}
	var perTopic []core.Finding
	for _, t := range topics {
		tagged, err := fx.ZkList(ctx, root, "sources", "--tag", t)
		if err != nil {
			return 1, err
		}
		linked, err := fx.ZkList(ctx, root, "--link-to", "topics/"+t+".md")
		if err != nil {
			return 1, err
		}
		linked = core.Filter(linked, core.UnderDir("sources"))
		perTopic = append(perTopic, core.TopicFindings(core.TopicView{Topic: t, Tagged: tagged, Linked: linked})...)
	}
	orphans, err := fx.ZkList(ctx, root, "--orphan")
	if err != nil {
		return 1, err
	}
	ids, err := fx.SourceIDs(root)
	if err != nil {
		return 1, err
	}
	all := append(append(core.NearDuplicates(topics), perTopic...), core.Orphans(orphans)...)
	all = append(all, core.IDFindings(ids)...)
	for _, f := range all {
		fmt.Fprintf(out, "[%s] %s: %s\n", f.Level, f.Subject, f.Msg)
	}
	if len(all) == 0 {
		fmt.Fprintln(out, "ok: nessun problema")
	}
	if core.HasWarn(all) {
		return 1, nil
	}
	return 0, nil
}

func need(args []string, n int, usage string) error {
	if len(args) < n {
		return fmt.Errorf("uso: %s", usage)
	}
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
