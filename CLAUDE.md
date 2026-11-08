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
e `wizki` (binario Go di questo repo: estrazione delle fonti, indice, log, lint).

    go build -o bin/wizki .        # una volta; poi ./bin/wizki ...

### Struttura

    library/
    ├── inbox/            fonti da elaborare (PDF, .md, .txt, .html, immagini, .csv/.tsv/.json)
    ├── raw/pdf/          PDF originali (mai modificare, come tutto raw/)
    ├── raw/doc/          .md, .txt, .html originali
    ├── raw/assets/       immagini: singole, o in <fonte>/ se allegate a un documento
    ├── raw/data/         .csv, .tsv, .json originali
    ├── raw/text/         testo estratto (mai modificare)
    ├── sources/          una nota per fonte: riassunto + tag + [[topic]] + [[entità]]
    ├── topics/           un hub per topic (temi e concetti): sintesi cumulativa
    ├── entities/         una pagina per entità: persona, ente, luogo, norma, opera, evento
    ├── analyses/         analisi salvate dalle query: domanda, risposta, pagine su cui si basa
    ├── overview.md       panoramica: tesi in evoluzione, temi principali, domande aperte
    ├── index.md          catalogo di tutte le pagine (generato da `wizki index`)
    └── log.md            registro in sola aggiunta

Ogni pagina (fonte, topic, entità, analisi, panoramica) ha nel frontmatter `summary:`,
un riassunto di **una riga** che finisce nell'indice. Tienilo aggiornato quando
la pagina cambia.

Una nuova wiki si crea con `./bin/wizki init <nome>`. Prima controlla le dipendenze
(git, poppler, tesseract con la lingua OCR, zk) e, se manca qualcosa, propone il
comando per installarlo con il gestore di pacchetti (pacman, apt, dnf, brew),
chiedendo conferma prima di lanciarlo; senza git si ferma. Lo stesso controllo si
fa in ogni momento con `./bin/wizki deps`. Poi crea `./<nome>/` con questa
struttura (un `.gitkeep` per cartella, `overview.md` vuota, `index.md` iniziale), `.zk/` (config e template copiati da `library/.zk/`), `.gitignore`
e un repository git (`git init -b master`). Non sovrascrive nulla: rilanciarlo
completa solo ciò che manca. Poi usala con `-root <nome>` (o `WIKI_ROOT`).

Per portare una wiki creata con una versione precedente al layout attuale:
`./bin/wizki -root <nome> upgrade` mostra cosa farebbe, `upgrade -apply` lo fa.
Crea ciò che manca, sostituisce i file di `.zk/` diversi da quelli incorporati
(salvando il vecchio in `<file>.bak`) e rigenera `index.md`; non tocca le note.
Dopo, confronta i `.bak` con i file nuovi (eventuali personalizzazioni vanno
riportate a mano), cancellali e lancia `./bin/wizki -root <nome> lint`: le note
vecchie possono non avere i campi nuovi (es. `summary:`, `published:`).

### Ingestion di una fonte

`./bin/wizki inbox` elenca le fonti della inbox nell'ordine in cui acquisirle
(senza le immagini allegate a un documento, che si acquisiscono con quello).

1. `./bin/wizki extract library/inbox/<file>` -> stampa il path del testo e,
   sulla riga dopo, l'id della fonte (primi 8 caratteri dello SHA-256 dell'originale).
   L'originale va in `raw/<tipo>/`, il testo in `raw/text/`:
   - **PDF**: testo con pdftotext. Se è una scansione, `wizki extract` fa da solo
     l'OCR (pdftoppm + tesseract, lingua `-ocr-lang`, default `ita`) e lo segnala
     su stderr: l'OCR sbaglia qualche carattere, quindi leggi il testo con sospetto.
   - **.md, .txt, .html**: il testo è il contenuto (l'HTML convertito in markdown,
     in modo approssimato: per gli articoli web è meglio il markdown di Obsidian
     Web Clipper). Le immagini locali referenziate vanno in `raw/assets/<fonte>/`
     e i riferimenti nel testo vengono riscritti; le immagini remote restano URL
     e non vengono scaricate (lo dice su stderr).
   - **Immagini**: il testo contiene il riferimento all'immagine e, se Tesseract
     ha la lingua, il testo OCR. **Guarda l'immagine** con il tool di lettura: l'OCR
     non descrive grafici, foto o schemi.
   - **.csv, .tsv, .json**: il testo è il contenuto.
   Nei documenti con immagini, leggi prima il testo, poi guarda le immagini che
   contano per capirlo.
   Exit code 3 = nessun testo (PDF anche con l'OCR, o file di testo vuoto);
   4 = OCR del PDF non disponibile (mancano comandi o lingua); 5 = fonte già
   acquisita (stesso contenuto di un originale in `raw/`, anche con nome diverso).
   In tutti questi casi la fonte resta in inbox: fermati e dimmelo.
2. Leggi il testo. Individua 1-3 topic: NOMI, italiano, minuscolo, singolare.
   Un topic conta solo se centrale, non se citato di passaggio.
3. `zk -W library topics` per vedere i topic esistenti. Riusali. Creane uno nuovo solo
   se nessuno copre il contenuto. Mai plurali o sinonimi di topic esistenti.
4. Individua le entità: persone, enti, luoghi, norme, opere, eventi su cui la
   fonte dice qualcosa di sostanziale (un ruolo, una decisione, un dato), anche
   se non sono centrali. Una semplice menzione non basta. Cerca quelle esistenti
   in `library/index.md` (sezione "Entità") e riusale: mai due pagine per la stessa
   entità. Il titolo è il nome completo più comune (es. "Ministero dell'Istruzione
   e del Merito", non "MIM"); sigle e varianti vanno nella "Descrizione".
   Trova anche la data del documento (data in testa, protocollo, anno di
   pubblicazione): `AAAA-MM-GG`, `AAAA-MM` o `AAAA`, oppure `sconosciuta` se
   nel testo non c'è. Non usare la data di oggi né quella del file.
5. Mostrami titolo, data del documento, topic scelti e perché, ed entità da
   creare o aggiornare. Procedi dopo la mia conferma.
6. Nota-fonte, con l'id stampato da `wizki extract` come `<sha>`:
   `zk -W library new sources --title "<titolo>" --extra sha=<sha> --print-path`
   (il file si chiama `<sha>-<titolo-slug>.md` e lo SHA finisce anche nel
   frontmatter come `id: <sha>`; senza `--extra sha` sarebbero vuoti)
   poi scrivi: `summary:`, `published:` (la data del documento), riassunto di 5 righe,
   tag `#<topic>`, link `[[topics/<topic>|<Titolo topic>]]`,
   link alle entità sotto "Entità" (`[[entities/<entità>|<Nome entità>]]`),
   `raw:` con il path dell'originale in `raw/` (es. `raw/pdf/circolare.pdf`).
7. Per ogni topic: se manca `topics/<topic>.md` crealo con
   `zk -W library new topics --title "<topic>" --print-path`.
   Aggiorna la sintesi integrando la nuova fonte, citala con `[[sources/<nota-fonte>|<Titolo fonte>]]`
   e segnala sotto "Contraddizioni" ciò che confligge con quanto già scritto.
   Linka nella sintesi le entità che contano per il topic.
   Confronta il `published:` della fonte con quello delle fonti già citate
   (in `index.md`). Se la fonte è più recente, aggiorna i dati superati e scrivi
   cosa sostituisce. Se è più vecchia, non sovrascrivere i dati più recenti:
   aggiungi le sue informazioni come storiche, datate ("nel 2019 ...").
8. Per ogni entità: se manca, creala con
   `zk -W library new entities --title "<Nome>" --extra tipo=<tipo> --print-path`
   (`<tipo>`: `persona`, `ente`, `luogo`, `norma`, `opera`, `evento`).
   Aggiungi sotto "Nelle fonti" cosa dice questa fonte, citandola; aggiorna
   "Descrizione" e `summary:`; segnala le "Contraddizioni" tra fonti.
   Le date contano come per i topic: i dati più vecchi restano, datati, come storia.
   Le entità non hanno tag: i tag `#` sono solo per i topic delle fonti.
9. Se due topic compaiono nella stessa fonte, linkali in "Vedi anche".
10. Aggiorna `overview.md`: come la fonte rafforza o mette in discussione la
    "Tesi", i "Temi principali" (link ai topic) e le "Domande aperte". Se la
    fonte non cambia il quadro, lasciala com'è. Aggiorna `summary:`.
11. `./bin/wizki index` per rigenerare `index.md`.
12. `./bin/wizki log ingest "<titolo>"`.

Una fonte può toccare 10-15 pagine: è normale.

### Query

1. Leggi prima `library/index.md` per trovare le pagine rilevanti e `library/overview.md`
   per il quadro d'insieme. Guarda anche la sezione "Analisi": se una domanda
   simile ha già un'analisi, parti da quella.
2. Poi leggi i topic e le entità (`zk -W library by <topic>`,
   `zk -W library list --link-to topics/<topic>.md`, `zk -W library list --link-to entities/<entità>.md`),
   e infine le fonti.
3. Rispondi citando le pagine con wikilink completi (percorso e titolo).
4. Se la risposta è riutilizzabile (un confronto, una sintesi tra più topic, un
   collegamento non ovvio), proponi di salvarla come analisi. Se esiste già
   un'analisi sulla stessa domanda, proponi di aggiornarla invece di crearne una nuova.
5. Dopo la mia conferma:
   - nuova analisi: `zk -W library new analyses --title "<titolo>" --print-path`
     (il file si chiama `<data>-<titolo-slug>.md`);
   - scrivi `summary:`, "Domanda" (com'è stata posta), "Risposta" (con le
     citazioni `[[sources/...|...]]`), "Basata su" (link a topic, entità e fonti
     usati) e "Vedi anche". Le analisi non hanno tag;
   - per aggiornare un'analisi esistente, porta `date:` a oggi e aggiungi in fondo
     alla "Risposta" cosa è cambiato e perché.
6. Linka l'analisi dalla sezione "Analisi" di ogni topic su cui si basa (se la
   sezione manca, aggiungila) e dal "Vedi anche" delle entità.
7. Se l'analisi risponde a una "Domanda aperta" di `overview.md`, sostituisci
   la domanda con il link all'analisi.
8. `./bin/wizki index`, poi `./bin/wizki log query "<titolo>"`.

### Lint (su richiesta)

1. `./bin/wizki lint` (exit 1 se ci sono warn). Controlli meccanici:
   - tag senza link e viceversa;
   - plurali o sinonimi tra topic;
   - note orfane;
   - topic con una sola fonte;
   - `id:` mancanti o duplicati tra le fonti;
   - `summary:` mancanti;
   - `published:` mancanti o malformati;
   - link a pagine inesistenti;
   - topic che compaiono nella stessa fonte ma non si linkano;
   - analisi senza link a fonti, topic o entità;
   - `index.md` mancante o non aggiornato;
   - come `info`, le fonti più vecchie acquisite dopo fonti più recenti sullo
     stesso topic o entità, e le analisi su cui sono arrivate fonti dopo la loro `date:`.
2. Controlli di giudizio, a mano, leggendo `index.md` e le pagine:
   - **Contraddizioni** tra pagine topic ed entità.
   - **Affermazioni superate**: per ogni topic o entità con fonti di date diverse
     (prima di tutto quelle segnalate come `info`), rileggi le citazioni in ordine
     di `published:` e cerca ciò che è presentato come attuale ma è smentito o
     aggiornato da una fonte più recente.
   - **Entità duplicate**: la stessa entità con nomi diversi.
   - **Concetti ed entità senza pagina**: nomi o concetti che compaiono in almeno
     due fonti (`zk -W library list --match "<nome>"`, oppure `rg` su `library/raw/text/`)
     ma non hanno una pagina.
   - **Riferimenti incrociati mancanti** oltre a quelli segnalati: entità che
     contano per un topic ma non sono linkate dalla sintesi, pagine che parlano
     della stessa cosa senza linkarsi.
   - **Lacune**: dati mancanti o vecchi che una ricerca web potrebbe colmare.
     Proponi query precise; non cercare senza la mia conferma. Ciò che trovi sul
     web entra nella wiki solo come nuova fonte in `inbox/`, mai direttamente
     nelle pagine.
   - **Analisi da rivedere**: per quelle segnalate come `info`, leggi le fonti nuove e
     dimmi se le cambiano; proponi l'aggiornamento come al punto 5 di "Query".
   - **Nuove domande e fonti**: proponi 3-5 domande da approfondire e fonti da cercare.
3. Presenta un report: ogni problema con pagina, motivo e correzione proposta.
   Non applicare nulla senza la mia conferma. Le domande che approvo vanno in
   "Domande aperte" di `overview.md`. Dopo le correzioni, `./bin/wizki index`.
4. Registra con `./bin/wizki log lint "<data>"`.

### Regole

- Non modificare mai `raw/`. `wizki extract` rifiuta di sovrascrivere.
- Le note-fonte vecchie hanno `pdf:` invece di `raw:`: sono valide, non serve migrarle.
- Un'immagine usata da due documenti della inbox va con il primo acquisito: nel
  secondo il riferimento resta senza file (lo segnala `wizki extract`). In quel
  caso linka l'immagine già in `raw/assets/` dalla nota-fonte.
- Non creare cartelle tematiche né symlink: le viste per argomento sono query zk.
- Wikilink SEMPRE con percorso e titolo: `[[topics/scuola|scuola]]`,
  `[[sources/1a2b3c4d-titolo|Titolo della nota]]`. Il titolo dopo `|` è il `title:`
  della nota di destinazione. Mai `[[percorso]]` nudo.
  zk risolve `[[scuola]]` con un match parziale sul nome file e potrebbe puntare
  a una fonte il cui nome contiene "scuola"; il percorso esatto ha la precedenza.
- `log.md` e `index.md` sono esclusi dall'indice di zk (vedi `exclude` in `.zk/config.toml`):
  altrimenti l'indice, che linka tutto, nasconderebbe le note orfane.
- Non modificare mai `index.md` a mano: si rigenera con `./bin/wizki index`
  da `title:`, `summary:`, `tipo:`, `date:`, `published:` e dai link delle fonti.
- Lancia zk sempre dalla radice del repo con `zk -W library ...`, mai `cd library && zk`.
  `ZK_NOTEBOOK_DIR` punta a un altro notebook, e `--notebook-dir` da solo non basta:
  `zk new <cartella>` risolve la cartella rispetto alla directory corrente.

---

# 🇬🇧 English

## Personal wiki (zk notebook in `library/`)

"LLM Wiki" pattern: raw sources are immutable, the wiki is markdown
that you maintain, this file is the schema. Tools: `zk` (notes, tags, links)
and `wizki` (this repo's Go binary: source extraction, index, log, lint).

    go build -o bin/wizki .        # once; then ./bin/wizki ...

### Structure

    library/
    ├── inbox/            sources to process (PDF, .md, .txt, .html, images, .csv/.tsv/.json)
    ├── raw/pdf/          original PDFs (never modify, like all of raw/)
    ├── raw/doc/          original .md, .txt, .html
    ├── raw/assets/       images: standalone, or in <source>/ when attached to a document
    ├── raw/data/         original .csv, .tsv, .json
    ├── raw/text/         extracted text (never modify)
    ├── sources/          one note per source: summary + tags + [[topic]] + [[entity]]
    ├── topics/           one hub per topic (themes and concepts): cumulative synthesis
    ├── entities/         one page per entity: person, organization, place, law, work, event
    ├── analyses/         analyses saved from queries: question, answer, pages it is based on
    ├── overview.md       overview: evolving thesis, main themes, open questions
    ├── index.md          catalog of every page (generated by `wizki index`)
    └── log.md            append-only log

Every page (source, topic, entity, analysis, overview) has a `summary:` in its frontmatter,
a **one-line** summary that goes into the index. Keep it up to date when the
page changes.

A new wiki is created with `./bin/wizki init <name>`. It first checks the dependencies
(git, poppler, tesseract with the OCR language, zk) and, if something is missing,
proposes the command to install it with the package manager (pacman, apt, dnf, brew),
asking for confirmation before running it; without git it stops. The same check
can be run at any time with `./bin/wizki deps`. Then it creates `./<name>/` with this
layout (a `.gitkeep` per folder, an empty `overview.md`, an initial `index.md`), `.zk/` (config and templates copied from `library/.zk/`), `.gitignore`
and a git repository (`git init -b master`). It never overwrites anything: running it
again only fills in what is missing. Then use it with `-root <name>` (or `WIKI_ROOT`).

To bring a wiki created with an earlier version up to the current layout:
`./bin/wizki -root <name> upgrade` shows what it would do, `upgrade -apply` does it.
It creates what is missing, replaces the `.zk/` files that differ from the embedded
ones (saving the old one as `<file>.bak`) and regenerates `index.md`; it never touches notes.
Afterwards, compare the `.bak` files with the new ones (any customizations must be
carried over by hand), delete them and run `./bin/wizki -root <name> lint`: old
notes may lack the new fields (e.g. `summary:`, `published:`).

### Ingesting a source

`./bin/wizki inbox` lists the sources in the inbox in the order to ingest them
(without the images attached to a document, which are ingested with it).

1. `./bin/wizki extract library/inbox/<file>` -> prints the path of the text and,
   on the next line, the source's id (first 8 chars of the original's SHA-256).
   The original goes into `raw/<type>/`, the text into `raw/text/`:
   - **PDF**: text via pdftotext. If it is a scan, `wizki extract` runs OCR on
     its own (pdftoppm + tesseract, language `-ocr-lang`, default `ita`) and
     reports it on stderr: OCR gets a few characters wrong, so read the text with suspicion.
   - **.md, .txt, .html**: the text is the content (HTML converted to markdown,
     roughly: for web articles, Obsidian Web Clipper's markdown is better). Local
     images it references go into `raw/assets/<source>/` and the references in
     the text are rewritten; remote images stay as URLs and are not downloaded
     (reported on stderr).
   - **Images**: the text holds the reference to the image and, if Tesseract has
     the language, the OCR text. **Look at the image** with the read tool: OCR
     does not describe charts, photos or diagrams.
   - **.csv, .tsv, .json**: the text is the content.
   For documents with images, read the text first, then look at the images that
   matter to understand it.
   Exit code 3 = no text (PDF even with OCR, or empty text file); 4 = PDF OCR
   not available (missing commands or language); 5 = source already ingested
   (same content as an original in `raw/`, even under a different name).
   In all these cases the source stays in inbox: stop and tell me.
2. Read the text. Identify 1-3 topics: NOUNS, Italian, lowercase, singular.
   A topic counts only if it is central, not if mentioned in passing.
3. `zk -W library topics` to see the existing topics. Reuse them. Create a new one only
   if none covers the content. Never plurals or synonyms of existing topics.
4. Identify the entities: people, organizations, places, laws, works, events the
   source says something substantial about (a role, a decision, a figure), even
   if they are not central. A mere mention is not enough. Look for existing ones
   in `library/index.md` ("Entità" section) and reuse them: never two pages for the
   same entity. The title is the most common full name (e.g. "Ministero dell'Istruzione
   e del Merito", not "MIM"); acronyms and variants go in "Descrizione".
   Also find the document's date (date at the top, protocol, year of
   publication): `YYYY-MM-DD`, `YYYY-MM` or `YYYY`, or `sconosciuta` (unknown) if
   the text has none. Never use today's date or the file's date.
5. Show me the title, the document's date, the chosen topics and why, and the
   entities to create or update. Proceed after my confirmation.
6. Source note, with the id printed by `wizki extract` as `<sha>`:
   `zk -W library new sources --title "<title>" --extra sha=<sha> --print-path`
   (the file is named `<sha>-<title-slug>.md` and the SHA also goes into the
   frontmatter as `id: <sha>`; without `--extra sha` both would be empty)
   then write: `summary:`, `published:` (the document's date), a 5-line summary,
   tag `#<topic>`, link `[[topics/<topic>|<Topic title>]]`,
   links to the entities under "Entità" (`[[entities/<entity>|<Entity name>]]`),
   `raw:` with the path of the original in `raw/` (e.g. `raw/pdf/circolare.pdf`).
7. For each topic: if `topics/<topic>.md` is missing, create it with
   `zk -W library new topics --title "<topic>" --print-path`.
   Update the synthesis by integrating the new source, cite it with `[[sources/<source-note>|<Source title>]]`
   and flag under "Contraddizioni" (Contradictions) whatever conflicts with what is already written.
   Link the entities that matter for the topic from the synthesis.
   Compare the source's `published:` with that of the sources already cited
   (in `index.md`). If the source is newer, update the superseded data and say
   what it replaces. If it is older, don't overwrite the newer data: add its
   information as historical, dated ("in 2019 ...").
8. For each entity: if it is missing, create it with
   `zk -W library new entities --title "<Name>" --extra tipo=<type> --print-path`
   (`<type>`: `persona`, `ente`, `luogo`, `norma`, `opera`, `evento`).
   Add under "Nelle fonti" (In the sources) what this source says, citing it; update
   "Descrizione" (Description) and `summary:`; flag "Contraddizioni" between sources.
   Dates matter as for topics: older data stays, dated, as history.
   Entities have no tags: `#` tags are only for the topics of sources.
9. If two topics appear in the same source, link them under "Vedi anche" (See also).
10. Update `overview.md`: how the source strengthens or challenges the
    "Tesi" (Thesis), the "Temi principali" (Main themes, links to topics) and the
    "Domande aperte" (Open questions). If the source does not change the picture,
    leave it as is. Update `summary:`.
11. `./bin/wizki index` to regenerate `index.md`.
12. `./bin/wizki log ingest "<title>"`.

A single source may touch 10-15 pages: that's normal.

### Query

1. First read `library/index.md` to find the relevant pages and `library/overview.md`
   for the big picture. Also look at the "Analisi" (Analyses) section: if a similar
   question already has an analysis, start from it.
2. Then read the topics and entities (`zk -W library by <topic>`,
   `zk -W library list --link-to topics/<topic>.md`, `zk -W library list --link-to entities/<entity>.md`),
   and finally the sources.
3. Answer citing the pages with full wikilinks (path and title).
4. If the answer is reusable (a comparison, a synthesis across topics, a
   non-obvious connection), offer to save it as an analysis. If an analysis on
   the same question already exists, offer to update it instead of creating a new one.
5. After my confirmation:
   - new analysis: `zk -W library new analyses --title "<title>" --print-path`
     (the file is named `<date>-<title-slug>.md`);
   - write `summary:`, "Domanda" (Question, as it was asked), "Risposta" (Answer,
     with `[[sources/...|...]]` citations), "Basata su" (Based on: links to the
     topics, entities and sources used) and "Vedi anche". Analyses have no tags;
   - to update an existing analysis, set `date:` to today and add at the end of
     the "Risposta" what changed and why.
6. Link the analysis from the "Analisi" section of every topic it is based on (add
   the section if it is missing) and from the "Vedi anche" of the entities.
7. If the analysis answers an "open question" in `overview.md`, replace the
   question with the link to the analysis.
8. `./bin/wizki index`, then `./bin/wizki log query "<title>"`.

### Lint (on request)

1. `./bin/wizki lint` (exit 1 if there are warnings). Mechanical checks:
   - tags without links and vice versa;
   - plurals or synonyms among topics;
   - orphan notes;
   - topics with a single source;
   - missing or duplicate `id:` among sources;
   - missing `summary:`;
   - missing or malformed `published:`;
   - links to missing pages;
   - topics that appear in the same source but don't link each other;
   - analyses without links to sources, topics or entities;
   - missing or stale `index.md`;
   - as `info`, older sources ingested after newer ones on the same topic or
     entity, and analyses that got new sources after their `date:`.
2. Judgment checks, by hand, reading `index.md` and the pages:
   - **Contradictions** between topic and entity pages.
   - **Superseded claims**: for every topic or entity with sources of different
     dates (starting with those flagged as `info`), reread the citations in
     `published:` order and look for what is presented as current but is
     contradicted or updated by a newer source.
   - **Duplicate entities**: the same entity under different names.
   - **Concepts and entities without a page**: names or concepts that appear in at
     least two sources (`zk -W library list --match "<name>"`, or `rg` over `library/raw/text/`)
     but have no page.
   - **Missing cross-references** beyond those flagged: entities that matter for
     a topic but aren't linked from its synthesis, pages about the same thing
     that don't link each other.
   - **Gaps**: missing or old data that a web search could fill. Propose
     precise queries; don't search without my confirmation. What you find on
     the web enters the wiki only as a new source in `inbox/`, never directly
     in the pages.
   - **Analyses to review**: for those flagged as `info`, read the new sources and
     say whether they change them; propose the update as in step 5 of "Query".
   - **New questions and sources**: propose 3-5 questions to dig into and sources to look for.
3. Present a report: each problem with page, reason and proposed fix.
   Apply nothing without my confirmation. The questions I approve go into
   "Domande aperte" (Open questions) in `overview.md`. After the fixes, `./bin/wizki index`.
4. Record with `./bin/wizki log lint "<date>"`.

### Rules

- Never modify `raw/`. `wizki extract` refuses to overwrite.
- Older source notes have `pdf:` instead of `raw:`: they are valid, no need to migrate them.
- An image used by two documents in the inbox goes with the first one ingested:
  in the second the reference is left without a file (`wizki extract` reports it).
  In that case link the image already in `raw/assets/` from the source note.
- Don't create thematic folders or symlinks: per-topic views are zk queries.
- Wikilinks ALWAYS with path and title: `[[topics/scuola|scuola]]`,
  `[[sources/1a2b3c4d-titolo|Note title]]`. The title after `|` is the target
  note's `title:`. Never a bare `[[path]]`.
  zk resolves `[[scuola]]` with a partial match on the file name and might point
  to a source whose name contains "scuola"; the exact path takes precedence.
- `log.md` and `index.md` are excluded from the zk index (see `exclude` in `.zk/config.toml`):
  otherwise the index, which links everything, would hide orphan notes.
- Never edit `index.md` by hand: it is regenerated with `./bin/wizki index`
  from `title:`, `summary:`, `tipo:`, `date:`, `published:` and the links of the sources.
- Always run zk from the repo root with `zk -W library ...`, never `cd library && zk`.
  `ZK_NOTEBOOK_DIR` points to another notebook, and `--notebook-dir` alone is not enough:
  `zk new <folder>` resolves the folder against the current directory.
