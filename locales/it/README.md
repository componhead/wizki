# Wiki personale

Questo progetto è una wiki personale creata con `wizki init`. Segue il pattern
**LLM Wiki**: le fonti originali (PDF, articoli, foto, tabelle, libri) restano
intatte, e un assistente (Claude Code) ne ricava pagine markdown collegate tra
loro: una per fonte, una per topic, una per entità. Le pagine si aggiornano a
ogni fonte nuova: i collegamenti ci sono già, le contraddizioni sono già
segnalate, la panoramica tiene conto di tutto quello che è stato letto.

Tre ruoli:

| Chi | Cosa fa |
|---|---|
| **Tu** | Metti le fonti in `library/inbox/`, fai domande, approvi i piani dell'assistente |
| **L'assistente** | Legge le fonti, scrive e aggiorna le pagine, segnala contraddizioni e lacune |
| **wizki** | Archivia gli originali, estrae il testo, rigenera l'indice, tiene il registro, fa i controlli meccanici |

Le regole che l'assistente segue sono in `CLAUDE.md`: è lo schema della wiki, e
può evolvere con l'uso.

> Questo file, `CLAUDE.md` e `.claude/commands/` sono generati da wizki:
> `wizki upgrade -apply` li sostituisce con la versione attuale, salvando i
> vecchi in `<file>.bak`. Se vuoi annotare qualcosa sulla wiki, fallo in
> `library/overview.md`.

---

## Struttura

    ./
    ├── README.md             questo file
    ├── CLAUDE.md             lo schema: le regole per l'assistente
    ├── .claude/commands/     i comandi /ingest e /lint di Claude Code
    ├── .git/                 la storia del progetto
    └── library/              la wiki
        ├── inbox/            fonti da elaborare
        ├── raw/              originali, MAI da modificare
        │   ├── pdf/          PDF
        │   ├── doc/          .md, .txt, .html
        │   ├── assets/       immagini (in <fonte>/ se allegate a un documento)
        │   ├── data/         .csv, .tsv, .json
        │   ├── epub/         libri .epub
        │   └── text/         testo estratto dagli originali
        ├── sources/          una nota per fonte: riassunto, topic, entità
        ├── topics/           un hub per topic: sintesi cumulativa di tutte le fonti
        ├── entities/         una pagina per entità: persona, ente, luogo, norma, opera, evento
        ├── analyses/         risposte alle domande salvate perché riutilizzabili
        ├── overview.md       panoramica: tesi, temi principali, domande aperte
        ├── index.md          catalogo di tutte le pagine (generato, non modificarlo)
        ├── log.md            registro delle attività (solo in aggiunta)
        └── .zk/              configurazione e template di zk

- **`library/raw/`** è la cassaforte: `wizki extract` ci copia l'originale e il
  testo estratto, e si rifiuta di sovrascrivere. Nessuno modifica questi file.
- **Ogni pagina** ha nel frontmatter `title:` e `summary:` (una riga, quella che
  compare nell'indice). Le fonti hanno anche `id:` (i primi 8 caratteri dello
  SHA-256 dell'originale), `date:` (giorno di acquisizione), `published:` (data
  del documento: `AAAA-MM-GG`, `AAAA-MM`, `AAAA` o `sconosciuta`) e `raw:` (path
  dell'originale). Le entità hanno `tipo:`.
- **I tag** `#topic` stanno solo nelle note-fonte; topic, entità e analisi non
  hanno tag.
- **I link** sono sempre wikilink con percorso e titolo:
  `[[topics/scuola|scuola]]`, `[[sources/1a2b3c4d-circolare|Circolare 12/2026]]`.
  Mai `[[scuola]]` nudo: zk lo risolverebbe con un match parziale sul nome file.

---

## Prerequisiti

- **wizki** nel `PATH`: dal repository di wizki, `go install .` (finisce in
  `$(go env GOPATH)/bin`). Rilancialo a ogni aggiornamento di wizki.
- **zk**, **git**, **poppler** (`pdftotext`, `pdftoppm`) e **tesseract** con la
  lingua OCR (di default `ita`).
- **Claude Code** (`claude`).

`wizki deps` controlla cosa manca e propone il comando per installarlo.

**Tutti i comandi si lanciano dalla radice del progetto**, la cartella di
questo file: wizki trova da solo la wiki in `library/`. Da un'altra cartella,
indica il progetto con `wizki -root <progetto> ...`.

zk va lanciato sempre con `-W library`: `zk -W library ...`. La variabile
`ZK_NOTEBOOK_DIR` potrebbe puntare a un altro notebook.

---

## Il lavoro di tutti i giorni

Apri Claude Code nella radice del progetto:

    claude

### 1. Aggiungere una fonte

1. Metti il file in `library/inbox/`. Formati: PDF (anche scansioni, con OCR),
   `.md`, `.txt`, `.html` (con le immagini locali accanto), immagini, `.csv`,
   `.tsv`, `.json`, `.epub` (senza DRM). Per un documento Word, salvalo prima
   in PDF.
2. In Claude Code scrivi `/ingest`: ti propone le fonti in attesa, nell'ordine
   in cui acquisirle. Oppure `/ingest @library/inbox/<file>` (con Tab dopo `@`
   per completare il nome).
3. L'assistente:
   - lancia `wizki extract library/inbox/<file>`, che sposta l'originale in
     `library/raw/<tipo>/` e il testo in `library/raw/text/`, e stampa il
     path del testo e l'id della fonte;
   - legge il testo, sceglie 1-3 topic (nomi, minuscolo, singolare, riusando
     quelli esistenti) e le entità su cui la fonte dice qualcosa di sostanziale;
   - **ti mostra il piano** (titolo, data, punti chiave, topic, entità) e
     aspetta la tua conferma: è il momento di dirgli su cosa insistere;
   - scrive la nota-fonte, aggiorna topic, entità e `overview.md`, rigenera
     `index.md` e scrive nel registro.

   Una fonte può toccare 10-15 pagine: è normale.

Se `extract` si ferma, la fonte resta in `library/inbox/`:

| Exit | Significato | Cosa fare |
|---|---|---|
| 3 | Nessun testo (file vuoto, PDF illeggibile anche con OCR, EPUB con DRM) | Controlla il file o procurane una versione migliore |
| 4 | OCR non disponibile (manca tesseract o la lingua) | `wizki deps`, poi riprova |
| 5 | Fonte già acquisita (stesso contenuto, anche con nome diverso) | Cancella la copia dalla inbox |
| 6 | `raw/` in rete non raggiungibile | Monta la condivisione e riprova |

### 2. Fare domande

Chiedi all'assistente in linguaggio naturale. Parte da `library/index.md` e
`library/overview.md`, poi legge topic, entità e fonti, e risponde citando le
pagine. Se la risposta è riutilizzabile (un confronto, una sintesi tra più
topic) ti propone di salvarla in `library/analyses/`; l'analisi viene linkata
dai topic e dalle entità su cui si basa.

Per cercare da solo:

    zk -W library list --match "<termini>"          # nelle pagine (anche "lezion*")
    zk -W library by <topic>                        # fonti con quel tag
    zk -W library list --link-to topics/<topic>.md  # chi linka un topic
    zk -W library topics                            # tutti i topic, con il conteggio
    rg -i "<termini>" library/raw/text/             # nel testo delle fonti

### 3. Il controllo di salute

Ogni tanto scrivi `/lint` in Claude Code. Prima i controlli meccanici
(`wizki lint`, exit 1 se ci sono warn): tag senza link e viceversa, plurali o
sinonimi tra topic, note orfane, topic con una sola fonte, `id:` mancanti o
duplicati, `summary:` e `published:` mancanti o malformati, link rotti, topic
che compaiono insieme ma non si linkano, analisi senza basi, `index.md` non
aggiornato; come `info`, le fonti vecchie acquisite dopo quelle nuove e le
analisi su cui sono arrivate fonti nuove.

Poi l'assistente fa i controlli di giudizio (contraddizioni, affermazioni
superate, entità duplicate, concetti senza pagina, lacune da colmare con una
ricerca web) e ti presenta un report. **Non applica nulla senza la tua
conferma.**

---

## Comandi

| Comando | Cosa fa |
|---|---|
| `wizki inbox` | Elenca le fonti in `library/inbox/`, nell'ordine in cui acquisirle |
| `wizki extract <file>` | Archivia l'originale in `library/raw/` ed estrae il testo |
| `wizki index` | Rigenera `library/index.md` dai frontmatter delle pagine |
| `wizki log <ingest\|query\|lint> "<titolo>"` | Aggiunge una riga a `library/log.md` |
| `wizki lint` | Controlli meccanici di salute |
| `wizki upgrade [-apply]` | Porta il progetto alla versione attuale di wizki |
| `wizki raw <cartella> [-apply]` | Sposta `library/raw/` su una cartella di rete |
| `wizki deps` | Controlla le dipendenze |
| `wizki slug "<testo>"` | Mostra come un titolo diventa nome file |

Opzioni di `extract`: `-ocr-lang ita+eng` per le fonti in più lingue,
`-no-ocr` per non tentare l'OCR sulle scansioni.

Le note si creano con zk:

    zk -W library new sources --title "<titolo>" --extra sha=<id> --print-path
    zk -W library new topics --title "<topic>" --print-path
    zk -W library new entities --title "<Nome>" --extra tipo=<tipo> --print-path
    zk -W library new analyses --title "<titolo>" --print-path

`<tipo>` delle entità: `persona`, `ente`, `luogo`, `norma`, `opera`, `evento`.

---

## Manutenzione

### Dopo aver aggiornato wizki

    go install .              # nel repository di wizki
    wizki upgrade             # qui: mostra cosa farebbe
    wizki upgrade -apply      # lo fa

Crea ciò che manca, sostituisce i file di `library/.zk/`, questo README,
`CLAUDE.md` e `.claude/commands/` se diversi da quelli incorporati (salvando il
vecchio in `<file>.bak`) e rigenera `library/index.md`. Non tocca le note.
Dopo, confronta i `.bak` con i file nuovi, riporta a mano le personalizzazioni
(anche quelle di `CLAUDE.md`), cancella i `.bak` e lancia `wizki lint`: le
note vecchie possono non avere i campi nuovi.

### Originali su un NAS

    wizki raw /mnt/nas/wiki          # mostra il piano
    wizki raw /mnt/nas/wiki -apply   # lo esegue

Copia gli originali verificandone lo SHA-256, lascia la vecchia `raw/` come
`library/raw.local-<data>` (da cancellare a mano dopo un controllo), trasforma
`library/raw` in un link alla cartella remota e ci scrive il marcatore
`.wizki-raw`. I path `raw/...` nelle note restano validi. Se la condivisione
non è montata, `extract` e `upgrade` si fermano con exit 6 senza toccare nulla;
leggere la wiki e fare domande funziona comunque.

### Storia

Il progetto è un repository git (branch `master`). `library/.zk/notebook.db` è
escluso: zk lo rigenera. Per vedere come è cambiata una pagina:

    zk -W library hist topics/<topic>.md

---

## Regole

- Non modificare mai `library/raw/`.
- Non modificare mai `library/index.md` a mano: si rigenera con `wizki index`.
- `library/log.md` si scrive solo con `wizki log`, in aggiunta.
- Niente cartelle tematiche né symlink: le viste per argomento sono query zk.
  L'unico link ammesso è `library/raw`, creato da `wizki raw`.
- Mai due pagine per la stessa entità: il titolo è il nome completo più comune,
  sigle e varianti vanno nella "Descrizione".
- Le date contano: una fonte più recente aggiorna i dati superati; una più
  vecchia aggiunge informazioni storiche, datate, senza sovrascrivere.
- Ciò che arriva dal web entra nella wiki solo come nuova fonte in
  `library/inbox/`.
