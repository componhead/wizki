package i18n

// Dict contiene tutti i testi di una lingua. I commenti indicano gli argomenti
// delle voci con verbi di formato, nell'ordine in cui vanno passati.
type Dict struct {
	Code string // codice della lingua, come in language di .zk/config.toml
	Name string // nome della lingua, nella lingua stessa

	// ---------------------------------------------------------- CLI
	FlagRoot, FlagOCRLang, FlagNoOCR, FlagLocale string
	Usage                                        string // comando
	UsageMain, UsageInit, UsageExtract, UsageLog string
	UsageUpgrade, UsageRaw, UsageSlug            string
	UnknownCommand                               string // comando
	ErrorPrefix                                  string
	UnknownLocale                                string // codice, codici disponibili
	WikiLocaleUnsupported                        string // codice della wiki, codice usato
	NotAWikiInit                                 string // radice
	DryRun                                       string
	PlanCreateDir, PlanCreate                    string // path
	PlanReplace                                  string // path, copia
	PlanLink                                     string // link, destinazione
	PlanMove                                     string // da, a

	// ------------------------------------------------------ extract
	AssetsMoved   string // "N immagini locali"
	AssetsMissing string // riferimenti
	LocalImages   Plural
	RemoteImages  Plural
	NoImageOCR    string // errore
	DuplicateOf   string // errore, path, id
	RawImmutable  string // path
	TryOCR        string // lingua OCR
	TryOCRScant   string // lingua OCR
	ScantKeep     string // motivo
	OCRUsed       string
	OCRNotBetter  string
	EpubImages    Plural
	EpubAuthor    string
	EpubDate      string
	EpubPublisher string
	EpubLanguage  string

	// --------------------------------------------------------- init
	InitMissingDep    string // dipendenza
	Created           string
	AlreadyThere      string
	WikiReady         string // nome, nome
	InitOldLayout     string // nome, nome
	LocaleMismatch    string // wiki, lingua della wiki, lingua chiesta
	WikiNameEmpty     string
	WikiNameInvalid   string // nome
	WikiNameSeparator string // nome
	WikiNameDash      string // nome

	// --------------------------------------------------------- lint
	LintOK         string
	TagNoLink      string // fonte, topic
	LinkNoTag      string // fonte, topic
	TopicNoSources string
	TopicOneSource string
	NearDuplicate  string
	Orphan         string
	IDMissing      string
	IDDuplicate    string // fonti
	SummaryMissing string
	IndexMissing   string
	IndexStale     string
	PubMissing     string // valore per "data sconosciuta"
	PubInvalid     string // valore, valore per "data sconosciuta"
	BrokenLink     string // pagina
	CrossRef       string // fonti
	OutOfOrder     string // fonte vecchia, data, fonte recente, data
	AnalysisNoRefs string
	AnalysisStale  string // data, fonti

	// ------------------------------------------------------ upgrade
	UpToDate        string // radice
	UpgradeDone     string
	MigrateConflict string // radice, library
	Migrated        string // wiki, progetto

	// --------------------------------------------------------- deps
	DepsHeader       string
	ZkNotPackaged    string // gestore, URL
	InstallByHand    string // dipendenza
	InstallCommand   string // gestore, comando
	RunItYourself    string
	AskInstall       string
	NothingInstalled string
	InstallFailed    string // errore
	ThisSystem       string
	PurposeGit       string
	PurposePdftotext string
	PurposePdftoppm  string
	PurposeTesseract string
	PurposeLang      string // lingua
	PurposeZk        string

	// ---------------------------------------------------------- raw
	RawAlready           string // radice, cartella
	CopyVerify           string // "N file", cartella
	Files                Plural
	SameAlready          string // "N file"
	PlanMoveKeep         string // da, a
	RawCopyFailed        string // errore
	RawUndoFailed        string // errore, path, path
	RawUndone            string // errore
	RawRetry             string // errore
	RawDone              string // radice, cartella
	RawDeleteLocal       string // path
	RawTargetNotAbs      string // cartella
	RawTargetTaken       string // cartella, wiki
	RawLinkedElsewhere   string // link, destinazione
	RawConflicts         string // file
	RawUnreachableLink   string // errore, link, destinazione
	RawMarkerMissing     string // errore, cartella, marcatore
	RawTargetUnreachable string // cartella, errore
	NotADir              string // path
	CannotWrite          string // cartella, errore

	// ------------------------------------------------------- errori
	ErrNoText, ErrScantText, ErrExists        string
	ErrOCRUnavailable                         string
	ErrRawUnreachable, ErrDuplicate, ErrEmpty string
	ErrEpubDRM, ErrEpubInvalid                string
	UnknownStep                               string // tipo del passo
	BackupCopy                                string // errore
	OCRMissingBin                             string // errore, comando
	OCRMissingLang                            string // errore, lingue
	OCRMissingTesseract                       string // errore
	NoPages                                   string
	CopyOf                                    string // path, errore
	CopyDiffers                               string
	CopyCheck                                 string // path, errore
	NotPDF                                    string // nome
	BadFileName                               string // nome
	Unsupported                               string // nome, estensioni
	UnknownLogKind                            string // tipo
	EmptyTitle                                string

	// ------------------------------------------- contenuto della wiki
	OCRHeading   string // intestazione del testo OCR di un'immagine, in raw/text/
	OverviewPage string // overview.md di una wiki nuova
	IndexHeader  string // titolo e commento in testa a index.md
	SecOverview  string
	SecTopics    string
	SecEntities  string
	SecAnalyses  string
	SecSources   string
	NoPagesYet   string
	NoSummary    string
	Sources      Plural
	SourceDates  string // data di pubblicazione, data di acquisizione
	OtherType    string // gruppo delle entità senza tipo
	Unknown      string // valore di published: per una fonte senza data
}

// IT è il dizionario italiano.
var IT = Dict{
	Code: "it",
	Name: "italiano",

	FlagRoot:              "radice del notebook zk",
	FlagOCRLang:           "lingua/e Tesseract (es. ita, ita+eng)",
	FlagNoOCR:             "non tentare l'OCR sulle scansioni",
	FlagLocale:            "lingua di wizki e delle nuove wiki (it, en); per una wiki esistente vale la sua",
	Usage:                 "uso: %s",
	UsageMain:             "wizki [-root dir] [-locale it|en] init|deps|extract|inbox|log|lint|index|upgrade|raw|slug ...",
	UsageInit:             "wizki init <nome> [--locale it|en]",
	UsageExtract:          "wizki extract <file>",
	UsageLog:              "wizki log <ingest|query|lint> <titolo>",
	UsageUpgrade:          "wizki [-root dir] upgrade [-apply]",
	UsageRaw:              "wizki [-root dir] raw <cartella> [-apply]",
	UsageSlug:             "wizki slug <testo>",
	UnknownCommand:        "comando sconosciuto %q",
	ErrorPrefix:           "errore:",
	UnknownLocale:         "lingua %q non disponibile (disponibili: %s)",
	WikiLocaleUnsupported: "la wiki è in lingua %q, che wizki non ha: uso %s",
	NotAWikiInit:          "%s non sembra una wiki (manca .zk/config.toml): per crearne una usa wizki init",
	DryRun:                "prova: nessuna modifica. Rilancia con -apply per applicare:",
	PlanCreateDir:         "crea: %s/",
	PlanCreate:            "crea: %s",
	PlanReplace:           "aggiorna: %s (copia del vecchio in %s)",
	PlanLink:              "collega: %s -> %s",
	PlanMove:              "sposta: %s -> %s",

	AssetsMoved:   "%s: con la fonte in raw/assets/",
	AssetsMissing: "immagini locali non trovate (restano come riferimenti): %s",
	LocalImages:   Plural{"immagine locale", "immagini locali"},
	RemoteImages:  Plural{"immagine remota non scaricata: resta come URL", "immagini remote non scaricate: restano come URL"},
	NoImageOCR:    "niente OCR dell'immagine (%v): salvo solo il riferimento",
	DuplicateOf:   "%w: stesso contenuto di %s (id %s)",
	RawImmutable:  "%s esiste già: raw/ è immutabile, non sovrascrivo",
	TryOCR:        "nessun testo nel PDF: provo l'OCR (%s)…",
	TryOCRScant:   "poco testo nel PDF (scansione?): provo l'OCR (%s)…",
	ScantKeep:     "tengo il poco testo del PDF: %v",
	OCRUsed:       "l'OCR ha letto più testo: uso quello",
	OCRNotBetter:  "l'OCR non ha letto più testo: tengo quello del PDF",
	EpubImages:    Plural{"immagine del libro in raw/assets/", "immagini del libro in raw/assets/"},
	EpubAuthor:    "Autore",
	EpubDate:      "Data",
	EpubPublisher: "Editore",
	EpubLanguage:  "Lingua",

	InitMissingDep:    "manca %s, che serve a init: installalo e rilancia (non ho creato nulla)",
	Created:           "creato:",
	AlreadyThere:      "già presente:",
	WikiReady:         "progetto %q pronto, con la wiki in library/: entra con cd %s e apri claude",
	InitOldLayout:     "%s è una wiki con il layout precedente (wiki nella radice): portala al layout attuale con wizki -root %s upgrade",
	LocaleMismatch:    "%s è una wiki in lingua %q: non la completo in %q (togli --locale o usa quella lingua)",
	WikiNameEmpty:     "nome della wiki vuoto",
	WikiNameInvalid:   "nome della wiki %q non valido",
	WikiNameSeparator: "nome della wiki %q non valido: niente separatori di path",
	WikiNameDash:      "nome della wiki %q non valido: non può iniziare con '-'",

	LintOK:         "ok: nessun problema",
	TagNoLink:      "%s ha il tag ma non linka [[topics/%s]]",
	LinkNoTag:      "%s linka [[topics/%s]] ma non ha il tag",
	TopicNoSources: "topic senza fonti",
	TopicOneSource: "una sola fonte",
	NearDuplicate:  "possibile plurale o sinonimo: unificare?",
	Orphan:         "nessun link in entrata",
	IDMissing:      "id mancante nel frontmatter",
	IDDuplicate:    "stesso PDF in più note: %s",
	SummaryMissing: "summary: mancante nel frontmatter",
	IndexMissing:   "mancante: lancia `wizki index`",
	IndexStale:     "non aggiornato: lancia `wizki index`",
	PubMissing:     "published: mancante (data del documento, o %s)",
	PubInvalid:     "published: %q non valido (AAAA, AAAA-MM, AAAA-MM-GG o %s)",
	BrokenLink:     "link a pagina inesistente [[%s]]",
	CrossRef:       "compaiono insieme in %s ma non si linkano in \"Vedi anche\"",
	OutOfOrder:     "%s (pubbl. %s) acquisita dopo %s (pubbl. %s): verifica che i dati più vecchi non sembrino attuali",
	AnalysisNoRefs: "analisi senza link a fonti, topic o entità",
	AnalysisStale:  "fonti acquisite dopo il %s sugli stessi topic/entità: %s; da rivedere?",

	UpToDate:        "%s è già aggiornata",
	UpgradeDone:     "fatto. Confronta i file .bak con quelli nuovi, poi cancellali; le note non sono state toccate: lancia wizki lint",
	MigrateConflict: "%s ha il layout precedente ma %s esiste già: spostala o rinominala e rilancia",
	Migrated:        "la wiki ora è in %s: lancia wizki e claude da %s, senza -root",

	DepsHeader:       "Dipendenze:",
	ZkNotPackaged:    "zk non è tra i pacchetti di %s: istruzioni su %s",
	InstallByHand:    "%s: installalo con il gestore di pacchetti del tuo sistema",
	InstallCommand:   "Per installare ciò che manca (%s):\n  %s",
	RunItYourself:    "Lancialo in un terminale, poi verifica con: wizki deps",
	AskInstall:       "Lo lancio ora? [s/N] ",
	NothingInstalled: "Non installo nulla. Puoi farlo dopo e verificare con: wizki deps",
	InstallFailed:    "installazione non riuscita (%v)",
	ThisSystem:       "questo sistema",
	PurposeGit:       "storia della wiki (serve a init)",
	PurposePdftotext: "testo dei PDF",
	PurposePdftoppm:  "OCR dei PDF scansionati",
	PurposeTesseract: "OCR di scansioni e immagini",
	PurposeLang:      "lingua %s per l'OCR",
	PurposeZk:        "note, tag e link (serve a lint)",

	RawAlready:           "%s/raw è già in %s",
	CopyVerify:           "copia con verifica: %s in %s",
	Files:                Plural{"file", "file"},
	SameAlready:          " (%s già lì, uguali)",
	PlanMoveKeep:         "sposta: %s -> %s (la copia locale resta lì)",
	RawCopyFailed:        "%w (la raw locale non è stata toccata: puoi rilanciare)",
	RawUndoFailed:        "%w; e non riesco a rimettere a posto %s: la raw locale è in %s",
	RawUndone:            "%w (ho rimesso a posto la raw locale)",
	RawRetry:             "%w (il link è fatto: rilancia lo stesso comando per completare)",
	RawDone:              "fatto: %s/raw ora è %s.",
	RawDeleteLocal:       "Quando hai controllato che i file ci siano tutti, puoi cancellare %s.",
	RawTargetNotAbs:      "la cartella remota deve essere un path assoluto: %q",
	RawTargetTaken:       "%s è già la cartella raw della wiki %q: scegline un'altra",
	RawLinkedElsewhere:   "%s punta già a %s: per cambiarla sposta i file e il link a mano",
	RawConflicts:         "nella destinazione ci sono file con lo stesso nome ma contenuto diverso: %s",
	RawUnreachableLink:   "%w: %s punta a %s, che non si raggiunge: la condivisione è montata?",
	RawMarkerMissing:     "%w: in %s manca %s: la condivisione è montata? (se è una cartella nuova, configurala con wizki raw)",
	RawTargetUnreachable: "la cartella remota %s non si raggiunge: %w",
	NotADir:              "%s non è una cartella",
	CannotWrite:          "non posso scrivere in %s: %w",

	ErrNoText:           "nessun testo estratto dal PDF",
	ErrScantText:        "poco testo nel PDF",
	ErrExists:           "esiste già",
	ErrOCRUnavailable:   "OCR non disponibile",
	ErrRawUnreachable:   "raw remoto non raggiungibile",
	ErrDuplicate:        "fonte già acquisita",
	ErrEmpty:            "nessun testo nella fonte",
	ErrEpubDRM:          "EPUB protetto da DRM: i capitoli sono cifrati, il testo non si legge",
	ErrEpubInvalid:      "EPUB non valido",
	UnknownStep:         "passo sconosciuto %T",
	BackupCopy:          "copia di sicurezza: %w",
	OCRMissingBin:       "%w: manca %s (servono poppler e tesseract)",
	OCRMissingLang:      "%w: manca il pacchetto lingua %s di Tesseract",
	OCRMissingTesseract: "%w: manca tesseract",
	NoPages:             "pdftoppm non ha prodotto nessuna pagina",
	CopyOf:              "copia di %s: %w",
	CopyDiffers:         "contenuto diverso dall'originale",
	CopyCheck:           "verifica della copia %s: %w",
	NotPDF:              "%q non è un PDF",
	BadFileName:         "nome file %q non utilizzabile",
	Unsupported:         "%q: tipo non supportato (estensioni: %s)",
	UnknownLogKind:      "tipo di log %q sconosciuto (ingest|query|lint)",
	EmptyTitle:          "titolo vuoto",

	OCRHeading: "## Testo (OCR)",
	OverviewPage: `---
title: Panoramica
summary:
---

# Panoramica

## Tesi

## Temi principali

## Domande aperte
`,
	IndexHeader: "# Indice\n\n<!-- Generato da `wizki index`: non modificare a mano. -->\n",
	SecOverview: "Panoramica",
	SecTopics:   "Topic",
	SecEntities: "Entità",
	SecAnalyses: "Analisi",
	SecSources:  "Fonti",
	NoPagesYet:  "_(nessuna pagina)_",
	NoSummary:   "_(riassunto mancante)_",
	Sources:     Plural{"fonte", "fonti"},
	SourceDates: "pubbl. %s · acq. %s",
	OtherType:   "altro",
	Unknown:     "sconosciuta",
}

// EN è il dizionario inglese.
var EN = Dict{
	Code: "en",
	Name: "English",

	FlagRoot:              "root of the zk notebook",
	FlagOCRLang:           "Tesseract language(s) (e.g. eng, eng+ita)",
	FlagNoOCR:             "don't try OCR on scans",
	FlagLocale:            "language of wizki and of new wikis (it, en); an existing wiki uses its own",
	Usage:                 "usage: %s",
	UsageMain:             "wizki [-root dir] [-locale it|en] init|deps|extract|inbox|log|lint|index|upgrade|raw|slug ...",
	UsageInit:             "wizki init <name> [--locale it|en]",
	UsageExtract:          "wizki extract <file>",
	UsageLog:              "wizki log <ingest|query|lint> <title>",
	UsageUpgrade:          "wizki [-root dir] upgrade [-apply]",
	UsageRaw:              "wizki [-root dir] raw <folder> [-apply]",
	UsageSlug:             "wizki slug <text>",
	UnknownCommand:        "unknown command %q",
	ErrorPrefix:           "error:",
	UnknownLocale:         "language %q not available (available: %s)",
	WikiLocaleUnsupported: "the wiki's language is %q, which wizki doesn't have: using %s",
	NotAWikiInit:          "%s doesn't look like a wiki (.zk/config.toml is missing): to create one use wizki init",
	DryRun:                "dry run: nothing changed. Run again with -apply to apply:",
	PlanCreateDir:         "create: %s/",
	PlanCreate:            "create: %s",
	PlanReplace:           "update: %s (old copy in %s)",
	PlanLink:              "link: %s -> %s",
	PlanMove:              "move: %s -> %s",

	AssetsMoved:   "%s: with the source in raw/assets/",
	AssetsMissing: "local images not found (left as references): %s",
	LocalImages:   Plural{"local image", "local images"},
	RemoteImages:  Plural{"remote image not downloaded: left as a URL", "remote images not downloaded: left as URLs"},
	NoImageOCR:    "no OCR for the image (%v): saving the reference only",
	DuplicateOf:   "%w: same content as %s (id %s)",
	RawImmutable:  "%s already exists: raw/ is immutable, not overwriting",
	TryOCR:        "no text in the PDF: trying OCR (%s)…",
	TryOCRScant:   "little text in the PDF (a scan?): trying OCR (%s)…",
	ScantKeep:     "keeping the PDF's scant text: %v",
	OCRUsed:       "OCR read more text: using it",
	OCRNotBetter:  "OCR didn't read more text: keeping the PDF's",
	EpubImages:    Plural{"image of the book in raw/assets/", "images of the book in raw/assets/"},
	EpubAuthor:    "Author",
	EpubDate:      "Date",
	EpubPublisher: "Publisher",
	EpubLanguage:  "Language",

	InitMissingDep:    "%s is missing and init needs it: install it and run again (nothing was created)",
	Created:           "created:",
	AlreadyThere:      "already there:",
	WikiReady:         "project %q ready, with the wiki in library/: cd %s and open claude",
	InitOldLayout:     "%s is a wiki with the previous layout (wiki at the root): bring it to the current layout with wizki -root %s upgrade",
	LocaleMismatch:    "%s is a wiki in %q: not completing it in %q (drop --locale or use that language)",
	WikiNameEmpty:     "empty wiki name",
	WikiNameInvalid:   "invalid wiki name %q",
	WikiNameSeparator: "invalid wiki name %q: no path separators",
	WikiNameDash:      "invalid wiki name %q: it can't start with '-'",

	LintOK:         "ok: no problems",
	TagNoLink:      "%s has the tag but doesn't link [[topics/%s]]",
	LinkNoTag:      "%s links [[topics/%s]] but doesn't have the tag",
	TopicNoSources: "topic without sources",
	TopicOneSource: "a single source",
	NearDuplicate:  "possible plural or synonym: merge?",
	Orphan:         "no incoming links",
	IDMissing:      "id missing from the frontmatter",
	IDDuplicate:    "same PDF in several notes: %s",
	SummaryMissing: "summary: missing from the frontmatter",
	IndexMissing:   "missing: run `wizki index`",
	IndexStale:     "out of date: run `wizki index`",
	PubMissing:     "published: missing (the document's date, or %s)",
	PubInvalid:     "published: %q is invalid (YYYY, YYYY-MM, YYYY-MM-DD or %s)",
	BrokenLink:     "link to a missing page [[%s]]",
	CrossRef:       "they appear together in %s but don't link each other under \"See also\"",
	OutOfOrder:     "%s (publ. %s) ingested after %s (publ. %s): check that the older data doesn't look current",
	AnalysisNoRefs: "analysis without links to sources, topics or entities",
	AnalysisStale:  "sources ingested after %s on the same topics/entities: %s; review it?",

	UpToDate:        "%s is already up to date",
	UpgradeDone:     "done. Compare the .bak files with the new ones, then delete them; notes were not touched: run wizki lint",
	MigrateConflict: "%s has the previous layout but %s already exists: move or rename it and run again",
	Migrated:        "the wiki is now in %s: run wizki and claude from %s, without -root",

	DepsHeader:       "Dependencies:",
	ZkNotPackaged:    "zk is not among the packages of %s: instructions at %s",
	InstallByHand:    "%s: install it with your system's package manager",
	InstallCommand:   "To install what is missing (%s):\n  %s",
	RunItYourself:    "Run it in a terminal, then check with: wizki deps",
	AskInstall:       "Run it now? [y/N] ",
	NothingInstalled: "Not installing anything. You can do it later and check with: wizki deps",
	InstallFailed:    "installation failed (%v)",
	ThisSystem:       "this system",
	PurposeGit:       "wiki history (needed by init)",
	PurposePdftotext: "text of PDFs",
	PurposePdftoppm:  "OCR of scanned PDFs",
	PurposeTesseract: "OCR of scans and images",
	PurposeLang:      "%s language for OCR",
	PurposeZk:        "notes, tags and links (needed by lint)",

	RawAlready:           "%s/raw is already in %s",
	CopyVerify:           "verified copy: %s to %s",
	Files:                Plural{"file", "files"},
	SameAlready:          " (%s already there, identical)",
	PlanMoveKeep:         "move: %s -> %s (the local copy stays there)",
	RawCopyFailed:        "%w (the local raw was not touched: you can run again)",
	RawUndoFailed:        "%w; and I can't put %s back: the local raw is in %s",
	RawUndone:            "%w (I put the local raw back)",
	RawRetry:             "%w (the link is in place: run the same command again to finish)",
	RawDone:              "done: %s/raw is now %s.",
	RawDeleteLocal:       "Once you have checked that all the files are there, you can delete %s.",
	RawTargetNotAbs:      "the remote folder must be an absolute path: %q",
	RawTargetTaken:       "%s is already the raw folder of wiki %q: pick another one",
	RawLinkedElsewhere:   "%s already points to %s: to change it, move the files and the link by hand",
	RawConflicts:         "the destination has files with the same name but different content: %s",
	RawUnreachableLink:   "%w: %s points to %s, which can't be reached: is the share mounted?",
	RawMarkerMissing:     "%w: %s lacks %s: is the share mounted? (if it is a new folder, set it up with wizki raw)",
	RawTargetUnreachable: "the remote folder %s can't be reached: %w",
	NotADir:              "%s is not a folder",
	CannotWrite:          "can't write to %s: %w",

	ErrNoText:           "no text extracted from the PDF",
	ErrScantText:        "little text in the PDF",
	ErrExists:           "already exists",
	ErrOCRUnavailable:   "OCR not available",
	ErrRawUnreachable:   "remote raw unreachable",
	ErrDuplicate:        "source already ingested",
	ErrEmpty:            "no text in the source",
	ErrEpubDRM:          "DRM-protected EPUB: the chapters are encrypted, the text can't be read",
	ErrEpubInvalid:      "invalid EPUB",
	UnknownStep:         "unknown step %T",
	BackupCopy:          "backup copy: %w",
	OCRMissingBin:       "%w: %s is missing (poppler and tesseract are needed)",
	OCRMissingLang:      "%w: Tesseract's %s language pack is missing",
	OCRMissingTesseract: "%w: tesseract is missing",
	NoPages:             "pdftoppm produced no pages",
	CopyOf:              "copy of %s: %w",
	CopyDiffers:         "content differs from the original",
	CopyCheck:           "checking the copy %s: %w",
	NotPDF:              "%q is not a PDF",
	BadFileName:         "file name %q can't be used",
	Unsupported:         "%q: unsupported type (extensions: %s)",
	UnknownLogKind:      "unknown log kind %q (ingest|query|lint)",
	EmptyTitle:          "empty title",

	OCRHeading: "## Text (OCR)",
	OverviewPage: `---
title: Overview
summary:
---

# Overview

## Thesis

## Main themes

## Open questions
`,
	IndexHeader: "# Index\n\n<!-- Generated by `wizki index`: do not edit by hand. -->\n",
	SecOverview: "Overview",
	SecTopics:   "Topics",
	SecEntities: "Entities",
	SecAnalyses: "Analyses",
	SecSources:  "Sources",
	NoPagesYet:  "_(no pages)_",
	NoSummary:   "_(missing summary)_",
	Sources:     Plural{"source", "sources"},
	SourceDates: "publ. %s · ingested %s",
	OtherType:   "other",
	Unknown:     "unknown",
}
