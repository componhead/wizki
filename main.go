// wizki: strumento di supporto per un notebook zk (pattern "LLM Wiki").
//
//	wizki [-root library] [-ocr-lang ita] [-no-ocr] extract <file>
//	                                              acquisisce una fonte: PDF (OCR se scansione), .md/.txt/.html
//	                                              (con le immagini locali), immagini (OCR facoltativo),
//	                                              .csv/.tsv/.json, .epub (capitoli e immagini); originale in raw/<tipo>, testo in raw/text;
//	                                              stampa il path del testo e, sulla riga dopo, l'id (SHA-256, 8 hex)
//	exit: 0 ok, 1 errore, 2 uso, 3 nessun testo (anche con OCR, o EPUB con DRM), 4 OCR non disponibile,
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
//	wizki [-locale it|en] deps|slug ...            lingua dei comandi senza wiki (anche WIKI_LOCALE); sulle
//	                                              wiki vale sempre la loro (language di .zk/config.toml)
//	wizki init <nome> [--locale it|en]             controlla le dipendenze come deps (senza git si ferma), poi
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
	"wizki/i18n"
)

// zkDirs: per lingua, la cartella (nell'embed) con la configurazione zk e i
// template copiati da `wizki init` in ogni nuova wiki. Per l'italiano l'unica
// fonte resta library/.zk/ di questo repo; le altre lingue stanno in locales/.
var zkDirs = map[string]string{"it": "library/.zk", "en": "locales/en/zk"}

// zkConfig è la configurazione zk di riferimento di ogni lingua (vedi zkDirs).
//
//go:embed library/.zk/config.toml library/.zk/templates/source.md library/.zk/templates/topic.md library/.zk/templates/entity.md library/.zk/templates/analysis.md
//go:embed locales/en/zk/config.toml locales/en/zk/templates/source.md locales/en/zk/templates/topic.md locales/en/zk/templates/entity.md locales/en/zk/templates/analysis.md
var zkConfig embed.FS

// wikiCommands sono i comandi che lavorano su una wiki esistente: parlano la
// lingua della wiki (language in .zk/config.toml).
var wikiCommands = map[string]bool{"extract": true, "inbox": true, "log": true, "lint": true, "index": true, "upgrade": true, "raw": true}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, out, errw io.Writer) int {
	// lingua di partenza: WIKI_LOCALE, o l'italiano; serve già ai testi dei flag
	fallback, ok := i18n.Lookup(envOr("WIKI_LOCALE", i18n.DefaultCode))
	if !ok {
		fallback, _ = i18n.Lookup(i18n.DefaultCode)
	}
	i18n.Use(fallback)
	d := i18n.T()
	fs := flag.NewFlagSet("wizki", flag.ContinueOnError)
	fs.SetOutput(errw)
	root := fs.String("root", envOr("WIKI_ROOT", "library"), d.FlagRoot)
	ocrLang := fs.String("ocr-lang", envOr("WIKI_OCR_LANG", "ita"), d.FlagOCRLang)
	noOCR := fs.Bool("no-ocr", false, d.FlagNoOCR)
	locale := fs.String("locale", "", d.FlagLocale)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != "" {
		chosen, ok := i18n.Lookup(*locale)
		if !ok {
			fmt.Fprintln(errw, d.ErrorPrefix, fmt.Sprintf(d.UnknownLocale, *locale, strings.Join(i18n.Codes(), ", ")))
			return 2
		}
		i18n.Use(chosen)
		d = chosen
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintf(errw, d.Usage+"\n", d.UsageMain)
		return 2
	}
	if wikiCommands[rest[0]] {
		useWikiLocale(*root, errw)
		d = i18n.T()
	}

	var err error
	code := 0
	switch rest[0] {
	case "init":
		var name, loc string
		name, loc, err = initArgs(rest[1:], *locale)
		if err == nil {
			err = initWiki(ctx, name, loc, *ocrLang, out, errw)
		}
	case "extract":
		err = need(rest, 2, d.UsageExtract)
		if err == nil {
			err = extract(ctx, *root, rest[1], *ocrLang, *noOCR, out, errw)
		}
	case "log":
		err = need(rest, 3, d.UsageLog)
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
		err = need(rest, 2, d.UsageRaw)
		if err == nil {
			err = rawRemote(ctx, *root, rest[1:], out)
		}
	case "slug":
		err = need(rest, 2, d.UsageSlug)
		if err == nil {
			fmt.Fprintln(out, core.Slug(strings.Join(rest[1:], " ")))
		}
	default:
		err = fmt.Errorf(d.UnknownCommand, rest[0])
	}
	if err != nil {
		fmt.Fprintln(errw, i18n.T().ErrorPrefix, err)
		if errors.Is(err, fx.ErrNoText) || errors.Is(err, core.ErrEmpty) || errors.Is(err, core.ErrEpubDRM) {
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

// wikiLanguage legge la lingua della wiki in root (language di
// .zk/config.toml). ok è false se la wiki non ha una config o non dice la lingua.
func wikiLanguage(root string) (code string, ok bool) {
	cfg, exists, err := fx.ReadIfExists(filepath.Join(root, ".zk", "config.toml"))
	if err != nil || !exists {
		return "", false
	}
	code = core.ConfigLanguage(cfg)
	return code, code != ""
}

// useWikiLocale passa alla lingua della wiki in root, se ne ha una. Se la
// lingua della wiki non è nel dizionario, lo dice su errw e tiene quella corrente.
func useWikiLocale(root string, errw io.Writer) {
	code, ok := wikiLanguage(root)
	if !ok {
		return
	}
	d, found := i18n.Lookup(code)
	if !found {
		fmt.Fprintf(errw, i18n.T().WikiLocaleUnsupported+"\n", code, i18n.T().Code)
		return
	}
	i18n.Use(d)
}

// initArgs legge gli argomenti di `init`: il nome della wiki e, in qualsiasi
// posizione, --locale <codice> (anche -locale, o --locale=<codice>).
// global è il -locale dato prima del comando ("" se non c'è).
func initArgs(args []string, global string) (name, locale string, err error) {
	d := i18n.T()
	usage := fmt.Errorf(d.Usage, d.UsageInit)
	locale = global
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--locale" || a == "-locale":
			if i+1 >= len(args) {
				return "", "", usage
			}
			i++
			locale = args[i]
		case strings.HasPrefix(a, "--locale=") || strings.HasPrefix(a, "-locale="):
			_, locale, _ = strings.Cut(a, "=")
		case name == "" && !strings.HasPrefix(a, "-"):
			name = a
		default:
			return "", "", usage
		}
	}
	if name == "" {
		return "", "", usage
	}
	if locale != "" {
		if _, ok := i18n.Lookup(locale); !ok {
			return "", "", fmt.Errorf(d.UnknownLocale, locale, strings.Join(i18n.Codes(), ", "))
		}
	}
	return name, locale, nil
}

// extract acquisisce una fonte della inbox: l'originale in raw/<tipo>/, il
// testo in raw/text/ (per i PDF con OCR se scansione; per i doc le immagini
// locali in raw/assets/<stem>/). Stampa il path del testo e l'id.
// Se qualcosa va storto, la fonte e le sue immagini tornano dov'erano.
func extract(ctx context.Context, root, src, lang string, noOCR bool, out, errw io.Writer) error {
	d := i18n.T()
	// raw/ in rete: se la condivisione non è montata, non tocco niente
	if err := fx.CheckRaw(root); err != nil {
		return err
	}
	in := core.SourceInput{Path: src}
	kind, _ := core.SourceKind(src)
	switch kind {
	case core.KindDoc, core.KindData, core.KindEpub:
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
				fmt.Fprintf(errw, d.AssetsMoved+"\n", d.LocalImages.Count(len(found)))
			}
			if len(missing) > 0 {
				fmt.Fprintf(errw, d.AssetsMissing+"\n", strings.Join(missing, ", "))
			}
			if n := core.RemoteImageCount(text); n > 0 {
				fmt.Fprintln(errw, d.RemoteImages.Count(n))
			}
		}
	case core.KindImage:
		if !noOCR {
			if err := fx.CheckTesseract(ctx, lang); err != nil {
				fmt.Fprintf(errw, d.NoImageOCR+"\n", err)
			} else {
				in.Lang = lang
			}
		}
	}
	steps, err := core.PlanSource(root, in)
	if err != nil {
		return err
	}
	if kind == core.KindEpub {
		// tutti i WriteFile tranne l'ultimo (il testo) sono immagini del libro
		if n := countWrites(steps) - 1; n > 0 {
			fmt.Fprintln(errw, d.EpubImages.Count(n))
		}
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
		return fmt.Errorf(d.DuplicateOf, fx.ErrDuplicate, dup, core.ShortID(sum))
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
			return fmt.Errorf(d.RawImmutable, dst)
		}
	}
	rollback := func(upTo int) {
		for j := upTo; j >= 0; j-- {
			_ = fx.Undo(steps[j])
		}
	}
	for i, s := range steps {
		err := fx.Exec(ctx, s)
		if errors.Is(err, fx.ErrScantText) {
			// testo quasi vuoto (scansione con poche parole): non è un errore,
			// ma provo l'OCR e tengo il testo migliore
			err = nil
			if !noOCR {
				improveScant(ctx, s.(core.Extract), lang, errw)
			}
		}
		if errors.Is(err, fx.ErrNoText) && !noOCR {
			// scansione: ripiego su OCR; se non è disponibile, annullo il move
			if cerr := fx.CheckOCR(ctx, lang); cerr != nil {
				rollback(i)
				return cerr
			}
			fmt.Fprintf(errw, d.TryOCR+"\n", lang)
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
func initWiki(ctx context.Context, name, locale, lang string, out, errw io.Writer) error {
	if err := core.ValidWikiName(name); err != nil {
		return err
	}
	// una wiki già iniziata resta nella sua lingua: senza --locale la si
	// completa in quella, con una --locale diversa ci si ferma
	if have, ok := wikiLanguage(name); ok {
		if locale != "" && !strings.EqualFold(strings.TrimSpace(locale), have) {
			return fmt.Errorf(i18n.T().LocaleMismatch, name, have, locale)
		}
		locale = have
	}
	if locale != "" {
		if d, ok := i18n.Lookup(locale); ok {
			i18n.Use(d)
		}
	}
	missing, err := checkDeps(ctx, lang, out, errw)
	if err != nil {
		return err
	}
	for _, d := range missing {
		if d.Required {
			return fmt.Errorf(i18n.T().InitMissingDep, d.Name)
		}
	}
	files, err := zkFiles(i18n.T().Code)
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
		status := i18n.T().Created
		if err != nil {
			status = i18n.T().AlreadyThere
		}
		switch s := s.(type) {
		case core.WriteFile:
			fmt.Fprintln(out, status, s.Path)
		case core.GitInit:
			fmt.Fprintln(out, status, filepath.Join(s.Dir, ".git"))
		}
	}
	fmt.Fprintf(out, i18n.T().WikiReady+"\n", name, name, name)
	return nil
}

// zkFiles legge la configurazione zk incorporata della lingua code: path
// relativo alla radice della wiki (es. .zk/config.toml) -> contenuto.
func zkFiles(code string) (map[string][]byte, error) {
	dir, ok := zkDirs[code]
	if !ok {
		return nil, fmt.Errorf(i18n.T().UnknownLocale, code, strings.Join(i18n.Codes(), ", "))
	}
	files := map[string][]byte{}
	for _, p := range []string{"config.toml", "templates/source.md", "templates/topic.md", "templates/entity.md", "templates/analysis.md"} {
		b, err := zkConfig.ReadFile(dir + "/" + p)
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
		fmt.Fprintln(out, i18n.T().LintOK)
	}
	if core.HasWarn(all) {
		return 1, nil
	}
	return 0, nil
}

func need(args []string, n int, usage string) error {
	if len(args) < n {
		return fmt.Errorf(i18n.T().Usage, usage)
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
	d := i18n.T()
	apply := false
	for _, a := range args {
		switch a {
		case "-apply", "--apply":
			apply = true
		default:
			return fmt.Errorf(d.Usage, d.UsageUpgrade)
		}
	}
	if !fx.Exists(filepath.Join(root, ".zk", "config.toml")) {
		return fmt.Errorf(d.NotAWikiInit, displayPath(root))
	}
	if err := fx.CheckRaw(root); err != nil {
		return err
	}
	files, err := zkFiles(d.Code)
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
		fmt.Fprintf(out, d.UpToDate+"\n", root)
		return nil
	}
	if !apply {
		fmt.Fprintln(out, d.DryRun)
	}
	for _, s := range steps {
		switch s := s.(type) {
		case core.Mkdir:
			fmt.Fprintf(out, d.PlanCreateDir+"\n", s.Path)
		case core.WriteFile:
			fmt.Fprintf(out, d.PlanCreate+"\n", s.Path)
		case core.Replace:
			fmt.Fprintf(out, d.PlanReplace+"\n", s.Path, s.Backup)
		}
		if apply {
			if err := fx.Exec(ctx, s); err != nil {
				return err
			}
		}
	}
	if !apply {
		if needIndex {
			fmt.Fprintf(out, d.PlanCreate+"\n", indexPath)
		}
		return nil
	}
	if err := writeIndex(root, out); err != nil {
		return err
	}
	fmt.Fprintln(out, d.UpgradeDone)
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
	t := i18n.T()
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
		fmt.Fprint(out, t.DepsHeader+"\n"+core.DepReport(deps, m))
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
			fmt.Fprintf(out, t.ZkNotPackaged+"\n", orUnknown(manager), core.ZkInstallURL)
		default:
			fmt.Fprintf(out, t.InstallByHand+"\n", d.Name)
		}
	}
	if cmd == nil {
		return missing, nil
	}
	line := strings.Join(cmd, " ")
	fmt.Fprintf(out, t.InstallCommand+"\n", manager, line)
	if !interactive() {
		fmt.Fprintln(out, t.RunItYourself)
		return missing, nil
	}
	fmt.Fprint(out, t.AskInstall)
	answer, _ := bufio.NewReader(stdin).ReadString('\n')
	if !core.YesAnswer(answer) {
		fmt.Fprintln(out, t.NothingInstalled)
		return missing, nil
	}
	if err := fx.RunInteractive(ctx, cmd, stdin, out, errw); err != nil {
		fmt.Fprintf(errw, t.InstallFailed+"\n", err)
	}
	missing = missingNow()
	report(missing)
	return missing, nil
}

func orUnknown(manager string) string {
	if manager == "" {
		return i18n.T().ThisSystem
	}
	return manager
}

// rawRemote sposta raw/ della wiki in una cartella remota (una condivisione
// di rete montata): copia verificata degli originali, la cartella locale resta
// come raw.local-<data>, raw/ diventa un link. Senza -apply mostra il piano.
func rawRemote(ctx context.Context, root string, args []string, out io.Writer) error {
	d := i18n.T()
	apply, target := false, ""
	for _, a := range args {
		switch {
		case a == "-apply" || a == "--apply":
			apply = true
		case target == "" && !strings.HasPrefix(a, "-"):
			target = a
		default:
			return fmt.Errorf(d.Usage, d.UsageRaw)
		}
	}
	if target == "" {
		return fmt.Errorf(d.Usage, d.UsageRaw)
	}
	if !fx.Exists(filepath.Join(root, ".zk", "config.toml")) {
		return fmt.Errorf(d.NotAWiki, displayPath(root))
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
		fmt.Fprintf(out, d.RawAlready+"\n", root, absTarget)
		return nil
	}
	if !apply {
		fmt.Fprintln(out, d.DryRun)
	}
	copies := 0
	for _, s := range steps {
		if _, ok := s.(core.CopyVerify); ok {
			copies++
		}
	}
	if copies > 0 || len(st.Same) > 0 {
		fmt.Fprintf(out, d.CopyVerify, d.Files.Count(copies), absTarget)
		if len(st.Same) > 0 {
			fmt.Fprintf(out, d.SameAlready, d.Files.Count(len(st.Same)))
		}
		fmt.Fprintln(out)
	}
	for _, s := range steps {
		switch s := s.(type) {
		case core.Move:
			fmt.Fprintf(out, d.PlanMoveKeep+"\n", s.From, s.To)
		case core.Symlink:
			fmt.Fprintf(out, d.PlanLink+"\n", s.Link, s.Target)
		case core.WriteFile:
			fmt.Fprintf(out, d.PlanCreate+"\n", s.Path)
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
			return fmt.Errorf(d.RawCopyFailed, err)
		}
		if _, ok := s.(core.Symlink); ok {
			if uerr := fx.Undo(*moved); uerr != nil {
				return fmt.Errorf(d.RawUndoFailed, err, moved.From, moved.To)
			}
			return fmt.Errorf(d.RawUndone, err)
		}
		return fmt.Errorf(d.RawRetry, err)
	}
	if err := fx.CheckRaw(root); err != nil {
		return err
	}
	fmt.Fprintf(out, d.RawDone+"\n", root, absTarget)
	for _, s := range steps {
		if m, ok := s.(core.Move); ok {
			fmt.Fprintf(out, d.RawDeleteLocal+"\n", m.To)
		}
	}
	return nil
}

// improveScant: pdftotext ha trovato poco testo (vedi core.ScantText). Se
// l'OCR è disponibile lo prova e tiene il testo con più lettere; altrimenti,
// o se l'OCR fallisce, tiene quello di pdftotext e lo dice su errw.
func improveScant(ctx context.Context, e core.Extract, lang string, errw io.Writer) {
	d := i18n.T()
	if err := fx.CheckOCR(ctx, lang); err != nil {
		fmt.Fprintf(errw, d.ScantKeep+"\n", err)
		return
	}
	fmt.Fprintf(errw, d.TryOCRScant+"\n", lang)
	used, err := fx.ImproveWithOCR(ctx, core.FallbackOCR(e, lang))
	switch {
	case err != nil:
		fmt.Fprintf(errw, d.ScantKeep+"\n", err)
	case used:
		fmt.Fprintln(errw, d.OCRUsed)
	default:
		fmt.Fprintln(errw, d.OCRNotBetter)
	}
}

// countWrites conta i passi WriteFile di un piano.
func countWrites(steps []core.Step) int {
	n := 0
	for _, s := range steps {
		if _, ok := s.(core.WriteFile); ok {
			n++
		}
	}
	return n
}

// displayPath restituisce la forma assoluta di p, da mostrare nei messaggi
// d'errore: con un -root relativo si vede subito dove wizki ha cercato.
// Se il percorso assoluto non si può calcolare, restituisce p così com'è.
func displayPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
