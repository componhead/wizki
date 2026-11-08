// wizki: strumento di supporto per un notebook zk (pattern "LLM Wiki").
//
//	wizki [-root library] [-ocr-lang ita] [-no-ocr] extract <file>
//	                                              acquisisce una fonte: PDF (OCR se scansione), .md/.txt/.html
//	                                              (con le immagini locali), immagini (OCR facoltativo),
//	                                              .csv/.tsv/.json; originale in raw/<tipo>, testo in raw/text;
//	                                              stampa il path del testo e, sulla riga dopo, l'id (SHA-256, 8 hex)
//	exit: 0 ok, 1 errore, 2 uso, 3 nessun testo (anche con OCR), 4 OCR non disponibile,
//	      5 fonte già acquisita (stesso contenuto di un originale in raw/, anche con nome diverso)
//	      6 raw/ in rete non raggiungibile (condivisione non montata: manca il marcatore .wizki-raw)
//	wizki [-root library] inbox                    elenca le fonti acquisibili, nell'ordine in cui acquisirle
//	wizki [-root library] log <ingest|query|lint> <titolo>
//	wizki [-root library] lint                     controlli di salute (exit 1 se ci sono warn)
//	wizki [-root library] index                    rigenera index.md: catalogo delle pagine con il loro summary:
//	wizki [-root library] upgrade [-apply]          porta una wiki esistente al layout e alla config zk
//	                                              attuali (copia .bak dei file sostituiti); senza -apply
//	                                              mostra solo cosa farebbe; non tocca le note
//	wizki [-ocr-lang ita] deps                     controlla le dipendenze (anche le lingue OCR) e propone
//	                                              il comando per installare ciò che manca (exit 1 se manca)
//	wizki [-root library] raw <cartella> [-apply]  sposta raw/ in una cartella di rete montata (copia
//	                                              verificata, raw.local-<data> come backup, raw diventa un
//	                                              link); senza -apply mostra solo il piano
//	wizki slug <testo>
//	wizki init <nome>                              controlla le dipendenze come deps (senza git si ferma), poi
//	                                              crea la wiki ./<nome>/ (cartelle, .zk/, log.md, overview.md, index.md,
//	                                              .gitignore, git init -b master); non sovrascrive nulla
package main

import (
	"bufio"
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
//go:embed library/.zk/config.toml library/.zk/templates/source.md library/.zk/templates/topic.md library/.zk/templates/entity.md library/.zk/templates/analysis.md
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
		fmt.Fprintln(errw, "uso: wizki [-root dir] init|deps|extract|inbox|log|lint|index|upgrade|raw|slug ...")
		return 2
	}

	var err error
	code := 0
	switch rest[0] {
	case "init":
		err = need(rest, 2, "wizki init <nome>")
		if err == nil {
			err = initWiki(ctx, rest[1], *ocrLang, out, errw)
		}
	case "extract":
		err = need(rest, 2, "wizki extract <file>")
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
	case "index":
		err = writeIndex(*root, out)
	case "upgrade":
		err = upgrade(ctx, *root, rest[1:], out)
	case "inbox":
		err = inbox(*root, out)
	case "deps":
		var missing []core.Dep
		missing, err = checkDeps(ctx, *ocrLang, out, errw)
		if err == nil && len(missing) > 0 {
			code = 1
		}
	case "raw":
		err = need(rest, 2, "wizki [-root dir] raw <cartella> [-apply]")
		if err == nil {
			err = rawRemote(ctx, *root, rest[1:], out)
		}
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
		if errors.Is(err, fx.ErrNoText) || errors.Is(err, core.ErrEmpty) {
			return 3
		}
		if errors.Is(err, fx.ErrOCRUnavailable) {
			return 4
		}
		if errors.Is(err, fx.ErrRawUnreachable) {
			return 6
		}
		if errors.Is(err, fx.ErrDuplicate) {
			return 5
		}
		return 1
	}
	return code
}

// extract acquisisce una fonte della inbox: l'originale in raw/<tipo>/, il
// testo in raw/text/ (per i PDF con OCR se scansione; per i doc le immagini
// locali in raw/assets/<stem>/). Stampa il path del testo e l'id.
// Se qualcosa va storto, la fonte e le sue immagini tornano dov'erano.
func extract(ctx context.Context, root, src, lang string, noOCR bool, out, errw io.Writer) error {
	// raw/ in rete: se la condivisione non è montata, non tocco niente
	if err := fx.CheckRaw(root); err != nil {
		return err
	}
	in := core.SourceInput{Path: src}
	kind, _ := core.SourceKind(src)
	switch kind {
	case core.KindDoc, core.KindData:
		b, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		in.Content = b
		if kind == core.KindDoc {
			text := core.DocText(src, b)
			found, missing := fx.ResolveAssets(src, core.ImageRefs(text))
			in.Assets = found
			if len(found) > 0 {
				fmt.Fprintf(errw, "%s: con la fonte in raw/assets/\n", core.CountPhrase(len(found), "immagine locale", "immagini locali"))
			}
			if len(missing) > 0 {
				fmt.Fprintf(errw, "immagini locali non trovate (restano come riferimenti): %s\n", strings.Join(missing, ", "))
			}
			if n := core.RemoteImageCount(text); n > 0 {
				fmt.Fprintln(errw, core.CountPhrase(n, "immagine remota non scaricata: resta come URL", "immagini remote non scaricate: restano come URL"))
			}
		}
	case core.KindImage:
		if !noOCR {
			if err := fx.CheckTesseract(ctx, lang); err != nil {
				fmt.Fprintf(errw, "niente OCR dell'immagine (%v): salvo solo il riferimento\n", err)
			} else {
				in.Lang = lang
			}
		}
	}
	steps, err := core.PlanSource(root, in)
	if err != nil {
		return err
	}
	// controllo sul contenuto, prima di toccare raw/: il nome del file non conta
	sum, err := fx.HashFile(src)
	if err != nil {
		return err
	}
	known, err := fx.HashRaw(root)
	if err != nil {
		return err
	}
	if dup, ok := core.FindDuplicate(sum, known); ok {
		return fmt.Errorf("%w: stesso contenuto di %s (id %s)", fx.ErrDuplicate, dup, core.ShortID(sum))
	}
	// raw/ è immutabile: se una destinazione esiste già, non tocco niente
	for _, s := range steps {
		var dst string
		switch s := s.(type) {
		case core.Move:
			dst = s.To
		case core.WriteFile:
			dst = s.Path
		case core.Extract:
			dst = s.Out
		case core.ImageText:
			dst = s.Out
		}
		if dst != "" && fx.Exists(dst) {
			return fmt.Errorf("%s esiste già: raw/ è immutabile, non sovrascrivo", dst)
		}
	}
	rollback := func(upTo int) {
		for j := upTo; j >= 0; j-- {
			_ = fx.Undo(steps[j])
		}
	}
	for i, s := range steps {
		err := fx.Exec(ctx, s)
		if errors.Is(err, fx.ErrNoText) && !noOCR {
			// scansione: ripiego su OCR; se non è disponibile, annullo il move
			if cerr := fx.CheckOCR(ctx, lang); cerr != nil {
				rollback(i)
				return cerr
			}
			fmt.Fprintf(errw, "nessun testo nel PDF: provo l'OCR (%s)…\n", lang)
			ocr := core.FallbackOCR(s.(core.Extract), lang)
			steps[i] = ocr
			err = fx.Exec(ctx, ocr)
		}
		if err != nil {
			// la fonte deve restare in inbox: annullo tutto quello che ho fatto
			rollback(i)
			return err
		}
	}
	var dest string
	switch last := steps[len(steps)-1].(type) {
	case core.Extract:
		dest = last.Out
	case core.OCR:
		dest = last.Out
	case core.WriteFile:
		dest = last.Path
	case core.ImageText:
		dest = last.Out
	}
	fmt.Fprintln(out, dest)
	fmt.Fprintln(out, core.ShortID(sum))
	return nil
}

// inbox stampa le fonti acquisibili della inbox, una per riga, nell'ordine in
// cui acquisirle: i documenti prima delle immagini, senza le immagini allegate
// a un documento (si acquisiscono con quello).
func inbox(root string, out io.Writer) error {
	names, err := fx.InboxSources(filepath.Join(root, "inbox"))
	if err != nil {
		return err
	}
	for _, n := range names {
		fmt.Fprintln(out, n)
	}
	return nil
}

// initWiki crea la wiki <name>/ nella cartella corrente. È idempotente: ciò che
// esiste già viene lasciato com'è e segnalato, così si può rilanciare per
// completare una wiki a cui manca qualche pezzo.
func initWiki(ctx context.Context, name, lang string, out, errw io.Writer) error {
	if err := core.ValidWikiName(name); err != nil {
		return err
	}
	missing, err := checkDeps(ctx, lang, out, errw)
	if err != nil {
		return err
	}
	for _, d := range missing {
		if d.Required {
			return fmt.Errorf("manca %s, che serve a init: installalo e rilancia (non ho creato nulla)", d.Name)
		}
	}
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
	for _, p := range []string{"config.toml", "templates/source.md", "templates/topic.md", "templates/entity.md", "templates/analysis.md"} {
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
	pages, err := fx.ReadPages(root)
	if err != nil {
		return 1, err
	}
	index, exists, err := fx.ReadIfExists(filepath.Join(root, core.IndexPath))
	if err != nil {
		return 1, err
	}
	orphans = core.Filter(orphans, core.NotOverview)
	all := append(append(core.NearDuplicates(topics), perTopic...), core.Orphans(orphans)...)
	all = append(all, core.SummaryFindings(pages)...)
	all = append(all, core.IndexFindings(index, exists, pages)...)
	all = append(all, core.PublishedFindings(pages)...)
	all = append(all, core.BrokenLinkFindings(pages)...)
	all = append(all, core.CrossRefFindings(pages)...)
	all = append(all, core.OutOfOrderFindings(pages)...)
	all = append(all, core.AnalysisFindings(pages)...)
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

// writeIndex rigenera <root>/index.md dalle pagine della wiki e ne stampa il path.
func writeIndex(root string, out io.Writer) error {
	pages, err := fx.ReadPages(root)
	if err != nil {
		return err
	}
	path := filepath.Join(root, core.IndexPath)
	if err := fx.WriteGenerated(path, core.BuildIndex(pages)); err != nil {
		return err
	}
	fmt.Fprintln(out, path)
	return nil
}

// upgrade porta la wiki in root al layout e alla configurazione zk incorporati:
// crea ciò che manca e sostituisce, con copia .bak, i file di .zk/ diversi.
// Senza -apply stampa soltanto il piano. Le note non vengono mai toccate.
func upgrade(ctx context.Context, root string, args []string, out io.Writer) error {
	apply := false
	for _, a := range args {
		switch a {
		case "-apply", "--apply":
			apply = true
		default:
			return fmt.Errorf("uso: wizki [-root dir] upgrade [-apply]")
		}
	}
	if !fx.Exists(filepath.Join(root, ".zk", "config.toml")) {
		return fmt.Errorf("%s non sembra una wiki (manca .zk/config.toml): per crearne una usa wizki init", root)
	}
	if err := fx.CheckRaw(root); err != nil {
		return err
	}
	files, err := zkFiles()
	if err != nil {
		return err
	}
	exists := map[string]bool{}
	for _, s := range core.Layout(root, filepath.Base(root), files) {
		switch s := s.(type) {
		case core.Mkdir:
			exists[s.Path] = fx.Exists(s.Path)
		case core.WriteFile:
			exists[s.Path] = fx.Exists(s.Path)
		}
	}
	have := map[string][]byte{}
	for rel := range files {
		p := filepath.Join(root, rel)
		exists[p+".bak"] = fx.Exists(p + ".bak")
		cur, ok, err := fx.ReadIfExists(p)
		if err != nil {
			return err
		}
		if ok {
			have[p] = []byte(cur)
		}
	}
	steps := core.PlanUpgrade(root, files, have, exists, fx.Today().Format("20060102-150405"))
	indexPath := filepath.Join(root, core.IndexPath)
	needIndex := !fx.Exists(indexPath)
	if len(steps) == 0 && !needIndex {
		fmt.Fprintf(out, "%s è già aggiornata\n", root)
		return nil
	}
	if !apply {
		fmt.Fprintln(out, "prova: nessuna modifica. Rilancia con -apply per applicare:")
	}
	for _, s := range steps {
		switch s := s.(type) {
		case core.Mkdir:
			fmt.Fprintf(out, "crea: %s/\n", s.Path)
		case core.WriteFile:
			fmt.Fprintf(out, "crea: %s\n", s.Path)
		case core.Replace:
			fmt.Fprintf(out, "aggiorna: %s (copia del vecchio in %s)\n", s.Path, s.Backup)
		}
		if apply {
			if err := fx.Exec(ctx, s); err != nil {
				return err
			}
		}
	}
	if !apply {
		if needIndex {
			fmt.Fprintf(out, "crea: %s\n", indexPath)
		}
		return nil
	}
	if err := writeIndex(root, out); err != nil {
		return err
	}
	fmt.Fprintln(out, "fatto. Confronta i file .bak con quelli nuovi, poi cancellali; le note non sono state toccate: lancia wizki lint")
	return nil
}

// stdin e interactive si sostituiscono nei test: lì nessuno risponde alle domande.
var (
	stdin       io.Reader = os.Stdin
	interactive           = fx.IsTerminal
)

// checkDeps controlla le dipendenze (comprese le lingue OCR di lang) e ne
// stampa l'elenco. Se manca qualcosa propone il comando che lo installa con il
// gestore di pacchetti del sistema; se c'è un terminale chiede se lanciarlo e,
// dopo l'installazione, ricontrolla. Restituisce ciò che manca ancora.
func checkDeps(ctx context.Context, lang string, out, errw io.Writer) ([]core.Dep, error) {
	deps := core.Deps(lang)
	missingNow := func() []core.Dep {
		m := fx.MissingDeps(ctx, deps)
		return core.Filter(deps, func(d core.Dep) bool { return m[d.Name] })
	}
	report := func(missing []core.Dep) {
		m := map[string]bool{}
		for _, d := range missing {
			m[d.Name] = true
		}
		fmt.Fprint(out, "Dipendenze:\n"+core.DepReport(deps, m))
	}
	missing := missingNow()
	report(missing)
	if len(missing) == 0 {
		return nil, nil
	}
	manager := fx.DetectManager()
	cmd, unpackaged := core.InstallCommand(manager, missing)
	for _, d := range unpackaged {
		switch {
		case d.Name == "zk":
			fmt.Fprintf(out, "zk non è tra i pacchetti di %s: istruzioni su %s\n", orUnknown(manager), core.ZkInstallURL)
		default:
			fmt.Fprintf(out, "%s: installalo con il gestore di pacchetti del tuo sistema\n", d.Name)
		}
	}
	if cmd == nil {
		return missing, nil
	}
	line := strings.Join(cmd, " ")
	fmt.Fprintf(out, "Per installare ciò che manca (%s):\n  %s\n", manager, line)
	if !interactive() {
		fmt.Fprintln(out, "Lancialo in un terminale, poi verifica con: wizki deps")
		return missing, nil
	}
	fmt.Fprint(out, "Lo lancio ora? [s/N] ")
	answer, _ := bufio.NewReader(stdin).ReadString('\n')
	if !core.YesAnswer(answer) {
		fmt.Fprintln(out, "Non installo nulla. Puoi farlo dopo e verificare con: wizki deps")
		return missing, nil
	}
	if err := fx.RunInteractive(ctx, cmd, stdin, out, errw); err != nil {
		fmt.Fprintf(errw, "installazione non riuscita (%v)\n", err)
	}
	missing = missingNow()
	report(missing)
	return missing, nil
}

func orUnknown(manager string) string {
	if manager == "" {
		return "questo sistema"
	}
	return manager
}

// rawRemote sposta raw/ della wiki in una cartella remota (una condivisione
// di rete montata): copia verificata degli originali, la cartella locale resta
// come raw.local-<data>, raw/ diventa un link. Senza -apply mostra il piano.
func rawRemote(ctx context.Context, root string, args []string, out io.Writer) error {
	apply, target := false, ""
	for _, a := range args {
		switch {
		case a == "-apply" || a == "--apply":
			apply = true
		case target == "" && !strings.HasPrefix(a, "-"):
			target = a
		default:
			return fmt.Errorf("uso: wizki [-root dir] raw <cartella> [-apply]")
		}
	}
	if target == "" {
		return fmt.Errorf("uso: wizki [-root dir] raw <cartella> [-apply]")
	}
	if !fx.Exists(filepath.Join(root, ".zk", "config.toml")) {
		return fmt.Errorf("%s non sembra una wiki (manca .zk/config.toml)", root)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	st, err := fx.ReadRawState(root, absTarget)
	if err != nil {
		return err
	}
	steps, err := core.PlanRaw(st, filepath.Base(absRoot), fx.Today().Format("20060102-150405"))
	if err != nil {
		return err
	}
	if len(steps) == 0 {
		fmt.Fprintf(out, "%s/raw è già in %s\n", root, absTarget)
		return nil
	}
	if !apply {
		fmt.Fprintln(out, "prova: nessuna modifica. Rilancia con -apply per applicare:")
	}
	copies := 0
	for _, s := range steps {
		if _, ok := s.(core.CopyVerify); ok {
			copies++
		}
	}
	if copies > 0 || len(st.Same) > 0 {
		fmt.Fprintf(out, "copia con verifica: %s in %s", core.CountPhrase(copies, "file", "file"), absTarget)
		if len(st.Same) > 0 {
			fmt.Fprintf(out, " (%s già lì, uguali)", core.CountPhrase(len(st.Same), "file", "file"))
		}
		fmt.Fprintln(out)
	}
	for _, s := range steps {
		switch s := s.(type) {
		case core.Move:
			fmt.Fprintf(out, "sposta: %s -> %s (la copia locale resta lì)\n", s.From, s.To)
		case core.Symlink:
			fmt.Fprintf(out, "collega: %s -> %s\n", s.Link, s.Target)
		case core.WriteFile:
			fmt.Fprintf(out, "crea: %s\n", s.Path)
		}
	}
	if !apply {
		return nil
	}
	var moved *core.Move
	for _, s := range steps {
		err := fx.Exec(ctx, s)
		switch {
		case err == nil:
			if m, ok := s.(core.Move); ok {
				moved = &m
			}
			continue
		case moved == nil:
			// fallita una copia: la raw locale è intatta, le copie già fatte
			// al prossimo giro risultano uguali e si saltano
			return fmt.Errorf("%w (la raw locale non è stata toccata: puoi rilanciare)", err)
		}
		if _, ok := s.(core.Symlink); ok {
			if uerr := fx.Undo(*moved); uerr != nil {
				return fmt.Errorf("%w; e non riesco a rimettere a posto %s: la raw locale è in %s", err, moved.From, moved.To)
			}
			return fmt.Errorf("%w (ho rimesso a posto la raw locale)", err)
		}
		return fmt.Errorf("%w (il link è fatto: rilancia lo stesso comando per completare)", err)
	}
	if err := fx.CheckRaw(root); err != nil {
		return err
	}
	fmt.Fprintf(out, "fatto: %s/raw ora è %s.\n", root, absTarget)
	for _, s := range steps {
		if m, ok := s.(core.Move); ok {
			fmt.Fprintf(out, "Quando hai controllato che i file ci siano tutti, puoi cancellare %s.\n", m.To)
		}
	}
	return nil
}
