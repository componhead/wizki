<!--
Bilingue / Bilingual: le due metà hanno lo stesso contenuto. In caso di
discrepanza fa fede la versione italiana (le slash command in .claude/commands
citano le sue sezioni). / Both halves have the same content; if they diverge,
the Italian version prevails (the slash commands cite its sections).
-->

# 🇮🇹 Italiano

## Wiki personale (notebook zk in `library/`)

Pattern "LLM Wiki": le fonti grezze sono immutabili, la wiki è markdown
che mantieni tu, questo file è lo schema. Strumenti: `zk` (note, tag, link)
e `wizki` (binario Go di questo repo: estrazione PDF, log, lint).

    go build -o bin/wizki .        # una volta; poi ./bin/wizki ...

### Struttura

    library/
    ├── inbox/            PDF da elaborare
    ├── raw/pdf/          PDF originali (mai modificare)
    ├── raw/text/         testo estratto (mai modificare)
    ├── sources/          una nota per fonte: riassunto + tag + [[topic]]
    ├── topics/           un hub per topic: sintesi cumulativa
    └── log.md            registro in sola aggiunta

Una nuova wiki si crea con `./bin/wizki init <nome>`: crea `./<nome>/` con questa
struttura (un `.gitkeep` per cartella), `.zk/` (config e template copiati da `library/.zk/`), `.gitignore`
e un repository git (`git init -b master`). Non sovrascrive nulla: rilanciarlo
completa solo ciò che manca. Poi usala con `-root <nome>` (o `WIKI_ROOT`).

### Ingestion di un PDF

1. `./bin/wizki extract library/inbox/<file>.pdf` -> stampa il path del testo e,
   sulla riga dopo, l'id del PDF (primi 8 caratteri del suo SHA-256).
   Se il PDF è una scansione, `wizki extract` fa da solo l'OCR (pdftoppm + tesseract,
   lingua `-ocr-lang`, default `ita`) e lo segnala su stderr: l'OCR sbaglia qualche
   carattere, quindi leggi il testo con sospetto. Exit code 3 = nessun testo nemmeno
   con l'OCR; 4 = OCR non disponibile (mancano comandi o lingua); 5 = PDF già
   acquisito (stesso contenuto di un file in `raw/pdf/`, anche con nome diverso).
   In tutti questi casi il PDF resta in inbox: fermati e dimmelo.
2. Leggi il testo. Individua 1-3 topic: NOMI, italiano, minuscolo, singolare.
   Un topic conta solo se centrale, non se citato di passaggio.
3. `zk -W library topics` per vedere i topic esistenti. Riusali. Creane uno nuovo solo
   se nessuno copre il contenuto. Mai plurali o sinonimi di topic esistenti.
4. Mostrami titolo, topic scelti e perché. Procedi dopo la mia conferma.
5. Nota-fonte, con l'id stampato da `wizki extract` come `<sha>`:
   `zk -W library new sources --title "<titolo>" --extra sha=<sha> --print-path`
   (il file si chiama `<sha>-<titolo-slug>.md` e lo SHA finisce anche nel
   frontmatter come `id: <sha>`; senza `--extra sha` sarebbero vuoti)
   poi scrivi: riassunto di 5 righe, tag `#<topic>`, link `[[topics/<topic>|<Titolo topic>]]`,
   `pdf:` con il path in `raw/pdf/`.
6. Per ogni topic: se manca `topics/<topic>.md` crealo con
   `zk -W library new topics --title "<topic>" --print-path`.
   Aggiorna la sintesi integrando la nuova fonte, citala con `[[sources/<nota-fonte>|<Titolo fonte>]]`
   e segnala sotto "Contraddizioni" ciò che confligge con quanto già scritto.
7. Se due topic compaiono nella stessa fonte, linkali in "Vedi anche".
8. `./bin/wizki log ingest "<titolo>"`.

Una fonte può toccare 10-15 pagine: è normale.

### Query

Rispondi leggendo prima i topic rilevanti (`zk -W library by <topic>`, `zk -W library list --link-to topics/<topic>.md`),
poi le fonti. Se la risposta è un'analisi riutilizzabile, proponi di salvarla
come nota e registra `./bin/wizki log query "<titolo>"`.

### Lint (su richiesta)

`./bin/wizki lint` (exit 1 se ci sono warn): tag senza link e viceversa, plurali o
sinonimi tra topic, note orfane, topic con una sola fonte, `id:` mancanti o
duplicati tra le fonti. Poi controlla a mano
le contraddizioni tra le pagine topic. Registra con `./bin/wizki log lint "<data>"`.

### Regole

- Non modificare mai `raw/`. `wizki extract` rifiuta di sovrascrivere.
- Non creare cartelle tematiche né symlink: le viste per argomento sono query zk.
- Wikilink SEMPRE con percorso e titolo: `[[topics/scuola|scuola]]`,
  `[[sources/1a2b3c4d-titolo|Titolo della nota]]`. Il titolo dopo `|` è il `title:`
  della nota di destinazione. Mai `[[percorso]]` nudo.
  zk risolve `[[scuola]]` con un match parziale sul nome file e potrebbe puntare
  a una fonte il cui nome contiene "scuola"; il percorso esatto ha la precedenza.
- `log.md` è escluso dall'indice di zk (vedi `exclude` in `.zk/config.toml`).
- Lancia zk sempre dalla radice del repo con `zk -W library ...`, mai `cd library && zk`.
  `ZK_NOTEBOOK_DIR` punta a un altro notebook, e `--notebook-dir` da solo non basta:
  `zk new <cartella>` risolve la cartella rispetto alla directory corrente.

---

# 🇬🇧 English

## Personal wiki (zk notebook in `library/`)

"LLM Wiki" pattern: raw sources are immutable, the wiki is markdown
that you maintain, this file is the schema. Tools: `zk` (notes, tags, links)
and `wizki` (this repo's Go binary: PDF extraction, log, lint).

    go build -o bin/wizki .        # once; then ./bin/wizki ...

### Structure

    library/
    ├── inbox/            PDFs to process
    ├── raw/pdf/          original PDFs (never modify)
    ├── raw/text/         extracted text (never modify)
    ├── sources/          one note per source: summary + tags + [[topic]]
    ├── topics/           one hub per topic: cumulative synthesis
    └── log.md            append-only log

A new wiki is created with `./bin/wizki init <name>`: it creates `./<name>/` with this
layout (a `.gitkeep` per folder), `.zk/` (config and templates copied from `library/.zk/`), `.gitignore`
and a git repository (`git init -b master`). It never overwrites anything: running it
again only fills in what is missing. Then use it with `-root <name>` (or `WIKI_ROOT`).

### Ingesting a PDF

1. `./bin/wizki extract library/inbox/<file>.pdf` -> prints the path of the text and,
   on the next line, the PDF's id (first 8 chars of its SHA-256).
   If the PDF is a scan, `wizki extract` runs OCR on its own (pdftoppm + tesseract,
   language `-ocr-lang`, default `ita`) and reports it on stderr: OCR gets a few
   characters wrong, so read the text with suspicion. Exit code 3 = no text even
   with OCR; 4 = OCR not available (missing commands or language); 5 = PDF already
   ingested (same content as a file in `raw/pdf/`, even under a different name).
   In all these cases the PDF stays in inbox: stop and tell me.
2. Read the text. Identify 1-3 topics: NOUNS, Italian, lowercase, singular.
   A topic counts only if it is central, not if mentioned in passing.
3. `zk -W library topics` to see the existing topics. Reuse them. Create a new one only
   if none covers the content. Never plurals or synonyms of existing topics.
4. Show me the title, the chosen topics and why. Proceed after my confirmation.
5. Source note, with the id printed by `wizki extract` as `<sha>`:
   `zk -W library new sources --title "<title>" --extra sha=<sha> --print-path`
   (the file is named `<sha>-<title-slug>.md` and the SHA also goes into the
   frontmatter as `id: <sha>`; without `--extra sha` both would be empty)
   then write: a 5-line summary, tag `#<topic>`, link `[[topics/<topic>|<Topic title>]]`,
   `pdf:` with the path in `raw/pdf/`.
6. For each topic: if `topics/<topic>.md` is missing, create it with
   `zk -W library new topics --title "<topic>" --print-path`.
   Update the synthesis by integrating the new source, cite it with `[[sources/<source-note>|<Source title>]]`
   and flag under "Contraddizioni" (Contradictions) whatever conflicts with what is already written.
7. If two topics appear in the same source, link them under "Vedi anche" (See also).
8. `./bin/wizki log ingest "<title>"`.

A single source may touch 10-15 pages: that's normal.

### Query

Answer by first reading the relevant topics (`zk -W library by <topic>`, `zk -W library list --link-to topics/<topic>.md`),
then the sources. If the answer is a reusable analysis, offer to save it
as a note and record `./bin/wizki log query "<title>"`.

### Lint (on request)

`./bin/wizki lint` (exit 1 if there are warnings): tags without links and vice versa, plurals or
synonyms among topics, orphan notes, topics with a single source, missing or
duplicate `id:` among sources. Then manually check
for contradictions between topic pages. Record with `./bin/wizki log lint "<date>"`.

### Rules

- Never modify `raw/`. `wizki extract` refuses to overwrite.
- Don't create thematic folders or symlinks: per-topic views are zk queries.
- Wikilinks ALWAYS with path and title: `[[topics/scuola|scuola]]`,
  `[[sources/1a2b3c4d-titolo|Note title]]`. The title after `|` is the target
  note's `title:`. Never a bare `[[path]]`.
  zk resolves `[[scuola]]` with a partial match on the file name and might point
  to a source whose name contains "scuola"; the exact path takes precedence.
- `log.md` is excluded from the zk index (see `exclude` in `.zk/config.toml`).
- Always run zk from the repo root with `zk -W library ...`, never `cd library && zk`.
  `ZK_NOTEBOOK_DIR` points to another notebook, and `--notebook-dir` alone is not enough:
  `zk new <folder>` resolves the folder against the current directory.
