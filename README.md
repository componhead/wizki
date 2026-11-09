# wizki

**wizki helps an AI assistant keep a personal encyclopedia of everything you read.**

You drop documents into a folder: PDFs, saved web articles, photos, spreadsheets.
The assistant ([Claude Code](https://claude.com/claude-code)) reads each one,
writes a summary, and files what it learned into a set of linked pages: one page
per subject, one per person or organization, an overall picture, an index. Every
new document makes those pages richer. When you have a question, the assistant
answers from pages it has already written, not by searching through piles of
files from scratch.

wizki is the toolbox the assistant uses to do this job carefully: it files the
original documents safely, pulls the text out of them, keeps the index and the
diary up to date, and runs a health check that catches mistakes.

You don't need to know how to program. You need to be able to type a few
commands, and this guide shows every one of them with real examples.

---

## Contents

- [The idea, in plain words](#the-idea-in-plain-words)
- [What your wiki looks like](#what-your-wiki-looks-like)
- [Before you start](#before-you-start)
- [The commands at a glance](#the-commands-at-a-glance)
- [A month with your wiki: thirteen scenarios](#a-month-with-your-wiki-thirteen-scenarios)
- [When something goes wrong](#when-something-goes-wrong)
- [The rules your wiki lives by](#the-rules-your-wiki-lives-by)
- [Glossary](#glossary)
- [Reference for the curious](#reference-for-the-curious)
- [Learn more](#learn-more)

---

## The idea, in plain words

Imagine you hired a tireless librarian.

- **You** do the interesting part: you find documents worth keeping, and you ask
  questions.
- **The librarian** does the boring part: reads everything, writes the index
  cards, cross-references them, notices when a new document contradicts an old
  one, and keeps the whole archive tidy.

Most "chat with your documents" tools work like a librarian with no memory: every
time you ask something, they run back to the shelves and skim everything again.
Nothing is built up. This project follows a different idea, called the
**LLM Wiki** pattern (proposed by Andrej Karpathy): the librarian writes things
down **once**, in a wiki, and then keeps that wiki up to date. Cross-references
are already there. Contradictions have already been flagged. The summary already
reflects everything you've read.

Three ingredients make it work:

| Ingredient | What it is | Who handles it |
|---|---|---|
| **Your documents** | The originals, stored untouched in a safe folder (`raw/`) | you add them, nobody ever changes them |
| **The wiki** | Plain text pages with links between them | the assistant writes them, you read them |
| **The rule book** | The file `CLAUDE.md`, which tells the assistant exactly how to work | you and the assistant improve it over time |

wizki itself is a small program that sits between them. It doesn't think: it
files, extracts, counts and checks. The thinking is the assistant's job, and the
deciding is yours. Before the assistant creates pages for a new document, it
always shows you what it plans to do and waits for your OK.

---

## What your wiki looks like

A wiki is just a folder on your computer. The main one in this project is called
`library/`:

    library/
    ├── inbox/            ← you drop new documents here
    ├── raw/              ← the safe: originals and their extracted text, never modified
    │   ├── pdf/              PDFs
    │   ├── doc/              web articles and text files (.md, .txt, .html)
    │   ├── assets/           pictures
    │   ├── data/             spreadsheets and data files (.csv, .tsv, .json)
    │   ├── epub/             e-books (.epub)
    │   └── text/             the text pulled out of each original
    ├── sources/          ← one page per document: summary, subjects, people involved
    ├── topics/           ← one page per subject, growing with every document
    ├── entities/         ← one page per person, organization, place, law, work or event
    ├── analyses/         ← answers to your questions that were worth keeping
    ├── overview.md       ← the big picture: main ideas, main subjects, open questions
    ├── index.md          ← the catalog of every page, with a one-line description
    └── log.md            ← the diary: what was added, asked or checked, and when

Every page is a plain text file (in a format called Markdown) that you can open
with any text editor. Links between pages look like `[[topics/orario|orario]]`:
the part before the `|` says where the link goes, and the part after it is the
name you read.

Subjects ("topics") are always **Italian nouns, lowercase, singular**: `scuola`,
`orario`, `trasporto`. This keeps the wiki from ending up with both `scuola` and
`scuole` as two separate pages.

Here is what a **document page** (in `sources/`) looks like once the assistant
has written it:

```markdown
---
id: 39a364a0
title: Circolare 12/2026 - nuovo orario
summary: From September school starts at 8:00 and buses are rescheduled
date: 2026-11-08
published: 2026-09-01
raw: raw/pdf/circolare-12-2026.pdf
---

# Circolare 12/2026 - nuovo orario

## Riassunto
From 1 September 2026 lessons start at 8:00 instead of 8:15 ...

## Topic
#orario #trasporto
[[topics/orario|orario]] · [[topics/trasporto|trasporto]]

## Entità
[[entities/liceo-galilei|Liceo Galilei]]
```

The block between the `---` lines is the page's **ID card**: `summary` is the
one-line description that ends up in the index, `date` is when the document was
added, `published` is the date written in the document itself, and `raw` points
to the original. The headings (`Riassunto`, `Topic`, `Entità`) are in Italian
because the wiki is.

A **topic page** (in `topics/`) gathers what every document says about one
subject, with sections for the synthesis, the sources, any contradictions
between sources, related subjects ("Vedi anche", see also) and saved analyses.

---

## Before you start

### What you need

- A computer with a **terminal**: the window where you type commands. On Linux
  and macOS it is called "Terminal"; on Windows, use WSL.
- **Claude Code**, the assistant that writes the wiki. Install it from
  [claude.com/claude-code](https://claude.com/claude-code).
- **Go** (version 1.24 or newer), used once to build wizki. Install it from
  [go.dev/dl](https://go.dev/dl) or with your system's package manager.
- A few free programs, which **wizki checks for you and offers to install**
  (see [Check what's missing](#check-whats-missing)):
  - **git**: keeps the history of your wiki. This is the only one wizki can't do
    without;
  - **poppler** (provides `pdftotext` and `pdftoppm`): reads PDFs;
  - **tesseract** with the **Italian language pack**: reads scans and photos.
    Without it, scanned documents and pictures can't be read automatically;
  - **zk**: manages the links between pages.

### How to type a command

In this guide, a command looks like this:

    ./bin/wizki inbox

Type it in the terminal exactly as written (or copy and paste it) and press
Enter. The lines shown below a command are what it prints back. Always type
commands from inside the project folder; to get there, type `cd` followed by the
folder's location, for example `cd ~/Documents/wizki`.

### Build wizki, once

    go build -o bin/wizki .

This turns the source code into a program, `bin/wizki`. You only do this once,
and again if you download a newer version of the project.

### Check what's missing

    ./bin/wizki deps

wizki lists every program it uses, with ✓ for those it found and ✗ for those
missing. On a computer that lacks only the Italian language for tesseract, it
prints:

    Dipendenze:
      ✓ git            storia della wiki (serve a init)
      ✓ pdftotext      testo dei PDF
      ✓ pdftoppm       OCR dei PDF scansionati
      ✓ tesseract      OCR di scansioni e immagini
      ✗ tesseract-ita  lingua ita per l'OCR
      ✓ zk             note, tag e link (serve a lint)
    Per installare ciò che manca (pacman):
      sudo pacman -S --needed tesseract-data-ita
    Lo lancio ora? [s/N]

*(Dependencies: wiki history (needed by init), PDF text, OCR of scanned PDFs, OCR
of scans and images, Italian language for OCR, notes/tags/links (needed by
lint). To install what's missing (pacman): … Shall I run it now? [y/N])*

wizki recognizes the most common package managers: **pacman** (Arch Linux),
**apt** (Debian, Ubuntu), **dnf** (Fedora) and **brew** (macOS). It builds the
right install command for yours and **asks before running it**:

- answer `s` (*sì*, yes) and press Enter, and it runs the command. The computer
  will ask for your password, because installing programs needs an
  administrator's permission. Afterwards wizki checks again and shows the new
  list.
- press Enter alone (or answer `n`) and nothing is installed. You can run the
  command yourself later.

When there's nobody to answer (for example when the assistant runs the command
for you), wizki only prints the install command and doesn't ask. Some programs
aren't available in every package manager. zk, for example, isn't in apt or
dnf, and in that case wizki tells you where to find it.

If your wiki reads documents in other languages too, include them:
`./bin/wizki -ocr-lang ita+eng deps` also checks English.

### A word about language

wizki speaks **Italian** or **English**, and so does each wiki. A wiki gets its
language when it is created (`wizki init <name> --locale en`, Italian if you
don't say), and keeps it: the templates, the overview, the catalog, the check-up
messages and everything wizki prints follow it. The language is written in the
wiki's settings (`language` in `.zk/config.toml`), so later commands find it on
their own. For commands that don't work on a wiki (`deps`, `slug`) choose it with
`-locale en`, or once for the session with `WIKI_LOCALE=en`.

`library/` is an Italian wiki, so in this guide wizki's messages are shown
exactly as they appear, followed by a translation.

---

## The commands at a glance

Most of the time you don't type these yourself: the assistant does, while it
follows the rule book. You type the two **slash commands** inside Claude Code,
and everything else happens on its own. It still helps to know what each one
does.

**Inside Claude Code** (open it in the project folder by typing `claude`):

| You type | What happens |
|---|---|
| `/ingest "<file>"` | The assistant adds a document from the inbox to the wiki. It shows you its plan first and waits for your OK. |
| `/lint` | The assistant gives the wiki a check-up and suggests fixes. It changes nothing without your OK. |
| a plain question | The assistant answers from the wiki and offers to save the answer if it's worth keeping. |

**In the terminal**:

| Command | In plain words | When |
|---|---|---|
| `wizki deps` | Check that the programs wizki needs are installed, and offer to install what's missing | Right after building wizki, or when something doesn't work |
| `wizki init <name> [--locale it\|en]` | Create a new, empty wiki in Italian or English (checking the programs first) | Starting a new subject area |
| `wizki inbox` | List what's waiting in the inbox, in the right order | Before adding documents |
| `wizki extract <file>` | Put a document in the safe and pull its text out | First step of every addition |
| `wizki index` | Rebuild the catalog of all pages | After pages change |
| `wizki log <ingest\|query\|lint> <title>` | Write a line in the diary | At the end of each activity |
| `wizki lint` | Run the automatic part of the check-up | Whenever you like |
| `wizki upgrade` | Show what's needed to update an older wiki | After installing a newer wizki |
| `wizki upgrade -apply` | Actually update it | After reading what `upgrade` showed |
| `wizki raw <folder>` / `raw <folder> -apply` | Move the safe (`raw/`) to a network drive: first show the plan, then do it | When originals take too much space, or should be shared |
| `wizki slug <text>` | Turn a title into a safe file name | Rarely; mostly used internally |

In the terminal, write `./bin/wizki` instead of just `wizki` (it means "the
wizki program in the `bin` folder").

---

## A month with your wiki: thirteen scenarios

Let's follow **Giulia**, a parent representative at her children's school. She
receives circulars, meeting minutes, notices and newsletters, and she always
struggles to remember what was decided and when. She decides to build a wiki.

### 1. Day one: the wiki is already there

This project comes with a ready-made wiki called `library/`. Giulia doesn't need
to create anything: she opens the terminal in the project folder, builds wizki
and checks that everything it needs is installed:

    cd ~/Documents/wizki
    go build -o bin/wizki .
    ./bin/wizki deps

The Italian language for tesseract is missing, so wizki offers the command to
install it. Giulia answers `s`, types her password, and the second list comes
back all ✓.

(If she ever wants a separate wiki for another subject, see
[scenario 10](#10-a-second-wiki-for-a-different-part-of-life).)

### 2. The first circular (a PDF)

Giulia downloads `Circolare 12-2026.pdf` from the school website and moves it
into `library/inbox/`. Then she opens Claude Code in the project folder:

    claude

and types:

    /ingest "Circolare 12-2026.pdf"

Here is what happens, step by step:

1. **The assistant files the original and reads it.** It runs:

       ./bin/wizki extract "library/inbox/Circolare 12-2026.pdf"

   which prints two lines:

       library/raw/text/circolare-12-2026.md
       39a364a0

   The first line is where the extracted text now lives. The second is the
   document's **ID**: a short code computed from the file's content, like a
   fingerprint. The original has moved from the inbox to
   `raw/pdf/circolare-12-2026.pdf`, where nothing will ever change it.

2. **The assistant reads the text and proposes a plan.** It looks at the
   subjects that already exist (so it reuses `orario` rather than inventing
   `orari`), then tells Giulia something like:

   > **Title:** Circolare 12/2026 – nuovo orario
   > **Document date:** 2026-09-01
   > **Topics:** `orario` (the circular is about the new timetable),
   > `trasporto` (it reschedules the school buses)
   > **Entities:** Liceo Galilei (new page), Comune di Milano (new page)
   > Shall I go ahead?

3. **Giulia checks and answers.** She can say "yes", or correct it: "use
   `autobus` rather than `trasporto`", or "the date is wrong, it's 2026-08-28".
   Nothing has been written yet.

4. **The assistant writes the pages:**
   - a document page in `sources/`, with a five-line summary;
   - the topic pages `orario` and `trasporto`, which it creates or enriches,
     noting any contradiction with what earlier documents said;
   - the pages for Liceo Galilei and Comune di Milano;
   - links between the two topics, since they appear together;
   - an updated `overview.md`, if the circular changes the big picture.

5. **It updates the catalog and the diary:**

       ./bin/wizki index
       ./bin/wizki log ingest "Circolare 12/2026 - nuovo orario"

   The diary, `log.md`, now ends with:

       ## [2026-11-08] ingest | Circolare 12/2026 - nuovo orario

A single document can touch 10 to 15 pages. That's normal: it's exactly the
filing work you'd never have time to do by hand.

> **What if it's a newer document on an old subject?** The assistant compares
> dates. Newer information replaces outdated information, and the page says
> what changed. An older document doesn't overwrite newer facts: its
> information is added as history ("in 2019, lessons started at 8:15").

### 3. A scanned letter

The headteacher's letter arrived on paper. Giulia scans it into a PDF and adds
it the same way. This time the PDF contains only an image of the page, not text,
so wizki notices and reads it with **OCR** (character recognition):

    nessun testo nel PDF: provo l'OCR (ita)…

*(no text in the PDF: trying OCR (Italian)…)*

The same happens when a scan carries only a scrap of text (page numbers, the
scanner app's watermark): if at least half of the pages have fewer than 40
letters, wizki tries OCR and keeps whichever text is longer:

    poco testo nel PDF (scansione?): provo l'OCR (ita)…
    l'OCR ha letto più testo: uso quello

*(little text in the PDF (a scan?): trying OCR (Italian)… OCR read more text: using it)*

In this case, if OCR isn't available or fails, wizki keeps the little text it
found, says so, and ingests the source anyway.

OCR occasionally misreads a letter or two, so the assistant reads the result
with a pinch of salt. If the Italian language pack for tesseract isn't
installed, wizki stops and explains, and the letter stays in the inbox
untouched:

    errore: OCR non disponibile: manca il pacchetto lingua ita di Tesseract

*(error: OCR not available: Tesseract's ita language pack is missing)*

For a document in another language, the language can be chosen:
`./bin/wizki -ocr-lang eng extract ...` for English, or `-ocr-lang ita+eng` for a
mix of both.

### 4. A web article with pictures

The local newspaper publishes an article about the new timetable. Giulia saves
it with **Obsidian Web Clipper**, a free browser extension that turns a web page
into a Markdown file and, if you ask it to, downloads the pictures too. She ends
up with `Timetable article.md` and a folder `images/` inside the inbox.

When the assistant adds the article, wizki moves the file **and the pictures it
uses**:

    1 immagine locale: con la fonte in raw/assets/
    library/raw/text/timetable-article-md.md
    47311eb0

*(1 local image: moved with the source into raw/assets/)*

The picture `images/enrolment chart.png` ends up in
`raw/assets/timetable-article/enrolment-chart.png`, and the link inside the
extracted text is corrected to point there. Pictures that were still on the
internet (links starting with `https://`) are **not** downloaded; wizki says so,
and they stay as web addresses:

    1 immagine remota non scaricata: resta come URL

*(1 remote image not downloaded: it stays as a URL)*

Giulia could also save the page as `.html` from her browser. wizki turns it
into readable text by itself, keeping the article and dropping menus and
footers. The conversion is rough, though, and the Web Clipper does a better job.

### 5. A photo of the notice board

Giulia takes a photo of a notice on the school board: `Notice board.jpg`. She
drops it in the inbox and adds it like any other document. The photo goes into
`raw/assets/`, and its text file contains a link to the picture plus whatever
OCR could read:

```markdown
![notice-board](raw/assets/notice-board.jpg)

## Testo (OCR)

PALESTRA CHIUSA
```

The assistant also **looks at the picture itself**: OCR can read printed words,
but it can't describe a chart, a map or a photo.

### 6. Numbers: a spreadsheet

The school council shares enrolment figures. Giulia saves the spreadsheet as CSV
(in most spreadsheet programs: *File → Save as → CSV*) and drops
`Enrolment 2026.csv` in the inbox. wizki files it in `raw/data/`, and the
assistant reads the numbers directly:

    library/raw/text/enrolment-2026-csv.md
    69f190a3

### 6b. A book: an EPUB

Giulia has the e-book of a history of Italian schools and drops
`Storia della scuola.epub` in the inbox. An EPUB is a zip of web pages, so wizki
opens it, reads the chapters in reading order and turns them into a single text:
the book's title, author, date, publisher and language at the top, then one
chapter after another, each under its title from the book's table of contents.
The original goes to `raw/epub/`, the pictures used by the chapters to
`raw/assets/storia-della-scuola/`:

    1 immagine del libro in raw/assets/
    library/raw/text/storia-della-scuola-epub.md
    5c1d2e3f

*(1 image of the book in raw/assets/)*

Links between chapters and to footnotes become plain text, since they lead
nowhere outside the book; web links stay. The date at the top often belongs to
the digital edition, not to the work: the assistant checks it before using it.
A book is long, so the assistant reads it chapter by chapter.

E-books bought from shops are often locked with DRM: then the chapters are
encrypted, wizki can't read them and stops, leaving the file in the inbox:

    errore: EPUB protetto da DRM: i capitoli sono cifrati, il testo non si legge: libro.epub

*(error: DRM-protected EPUB: the chapters are encrypted, the text can't be read)*

### 7. A full inbox

After a busy week the inbox holds six files. Before starting, Giulia asks wizki
what's there:

    ./bin/wizki inbox

    Enrolment 2026.csv
    Timetable article.md
    copy of enrolment.csv
    empty-note.txt
    Notice board.jpg

The list comes in the order the files should be added: documents first, pictures
last. Pictures that belong to an article (like the chart in scenario 4) aren't
listed at all, because they travel with their article. Giulia then adds the
files one by one with `/ingest`, confirming each plan.

Two of these will be refused, and that's a good thing. See
[When something goes wrong](#when-something-goes-wrong).

### 8. Asking questions

A month later, at a parents' meeting, someone asks: *"When did the timetable
change, and what happened to the buses?"* Giulia types the question in Claude
Code, just like that.

The assistant first reads `index.md` (the catalog) and `overview.md` (the big
picture). It checks whether a similar question has already been answered, then
reads the relevant topic and entity pages, and finally the documents
themselves. It answers with links to the pages it used.

If the answer is worth keeping (a comparison, a timeline, a connection nobody
had noticed), the assistant offers to save it as an **analysis**. With Giulia's
OK it creates a page like `analyses/2026-12-03-timetable-and-buses.md`, with
the question, the answer and the pages it's based on. It links the analysis
from the topics it touches, and the diary records:

    ## [2026-12-03] query | Timetable and buses

Next time the same question comes up, the answer is already in the wiki. If new
circulars arrive on the same subjects, the check-up (next scenario) will suggest
reviewing the analysis.

### 9. The monthly check-up

Once a month Giulia types `/lint` in Claude Code. The assistant first runs the
automatic checks:

    ./bin/wizki lint

    [info] orario: una sola fonte
    [warn] sources/47311eb0-new-timetable-for-2026.md: published: mancante (data del documento, o sconosciuta)
    [warn] sources/47311eb0-new-timetable-for-2026.md: link a pagina inesistente [[entities/liceo-galilei]]
    [warn] topics/orario ~ topics/trasporto: compaiono insieme in sources/47311eb0-new-timetable-for-2026.md ma non si linkano in "Vedi anche"
    [warn] index.md: non aggiornato: lancia `wizki index`

In English:

- *info:* the topic `orario` has only one source so far. This is just
  information, not a problem.
- *warning:* a document page is missing its publication date.
- *warning:* a document page links to a page that doesn't exist (maybe it was
  renamed, or never created).
- *warning:* two topics appear in the same document but don't link to each
  other.
- *warning:* the catalog is out of date: `wizki index` needs to run.

Lines marked `[warn]` are things to fix; lines marked `[info]` are worth knowing
but are not errors. When everything is fine, wizki prints
`ok: nessun problema` (*ok: no problems*).

The full list of automatic checks:

- **Tags and links**: a document tagged with a topic must also link to that topic's page, and the other way round.
- **Near-duplicate topics**: names that look like plurals or misspellings of each other (`scuola` / `scuole`).
- **Orphan pages**: pages no other page links to.
- **IDs**: missing IDs, or the same document added twice.
- **One-line descriptions**: pages without a `summary`.
- **Publication dates**: missing or written in the wrong format.
- **Links to missing pages.**
- **Related topics that don't link each other.**
- **Analyses that link no page.**
- **The catalog**: missing or out of date.
- **As information**:
  - an older document added after a newer one on the same subject, which could make old facts look current;
  - an analysis that newer documents may have overtaken.

Then the assistant does what a program can't do, by reading the pages:

- spots contradictions;
- spots claims that newer documents have overtaken;
- notices the same person or organization filed under two names;
- finds subjects mentioned in several documents that deserve a page of their own;
- notices missing cross-references;
- spots gaps that a web search could fill;
- suggests new questions to explore.

It presents everything as a report with proposed fixes and **changes nothing
until Giulia says yes**. It never searches the web without asking either, and
anything found online enters the wiki only as a new document in the inbox, never
straight into the pages. The questions Giulia approves go into the "open
questions" section of `overview.md`. Finally the check-up goes in the diary:

    ./bin/wizki log lint "2026-12-01"

### 10. A second wiki for a different part of life

Giulia is planning a family trip to Japan and wants to keep it separate from the
school wiki. She creates a new one:

    ./bin/wizki init viaggio-giappone

    Dipendenze:
      ✓ git            storia della wiki (serve a init)
      ...
      ✓ zk             note, tag e link (serve a lint)
    creato: viaggio-giappone/inbox/.gitkeep
    creato: viaggio-giappone/raw/pdf/.gitkeep
    ...
    creato: viaggio-giappone/overview.md
    creato: viaggio-giappone/index.md
    creato: viaggio-giappone/.git
    wiki "viaggio-giappone" pronta: usa wizki -root viaggio-giappone (o WIKI_ROOT=viaggio-giappone)

*(Dependencies: …; created: …; wiki "viaggio-giappone" ready: use wizki -root viaggio-giappone, or WIKI_ROOT=viaggio-giappone)*

Like `deps`, `init` first checks the programs and offers to install what's
missing. Since her guide-book PDFs are partly in English, Giulia could run
`./bin/wizki -ocr-lang ita+eng init viaggio-giappone` to check the English OCR
language too. If **git** is missing, `init` stops before creating anything: it
needs git to give the wiki its history.

The new folder has the same layout as `library/`, its own settings, and its own
history (it is a separate git repository). Running `init` again never overwrites
anything: it only fills in what's missing, and says `già presente` (*already
there*) for the rest.

A wiki can also be in English: `./bin/wizki init japan-trip --locale en` creates
the same layout with English templates (`## Summary`, `## See also`, `type:` for
entities…), an English overview and catalog, and wizki then talks English
whenever it works on it. Running `init` again on an existing wiki keeps its
language; asking for a different one with `--locale` is refused.

To work on this wiki instead of `library/`, every command gets `-root`:

    ./bin/wizki -root viaggio-giappone inbox
    ./bin/wizki -root viaggio-giappone lint

To avoid typing it every time, the choice can be set once for the whole terminal
session, with `export WIKI_ROOT=viaggio-giappone` (or `set -x WIKI_ROOT
viaggio-giappone` in the Fish shell).

Without `-root` and `WIKI_ROOT`, wizki uses the current folder if it is a wiki
(it contains `.zk/config.toml`), otherwise `library/`. So, from inside the wiki
folder, plain `wizki lint` is enough.

### 11. After updating wizki

Months later, Giulia downloads a newer version of this project, which has new
features such as analyses. Her `viaggio-giappone` wiki was created with the old
version, so it lacks some folders and settings. She asks wizki what's missing:

    ./bin/wizki -root viaggio-giappone upgrade

    prova: nessuna modifica. Rilancia con -apply per applicare:
    crea: viaggio-giappone/analyses/
    crea: viaggio-giappone/analyses/.gitkeep
    aggiorna: viaggio-giappone/.zk/config.toml (copia del vecchio in viaggio-giappone/.zk/config.toml.bak)
    crea: viaggio-giappone/.zk/templates/analysis.md
    aggiorna: viaggio-giappone/.zk/templates/topic.md (copia del vecchio in viaggio-giappone/.zk/templates/topic.md.bak)

*(trial run: no changes. Run again with -apply to apply. create: … / update: … (old copy saved in ….bak))*

This first run is only a **trial**: nothing changes. When she's happy with the
list, she applies it:

    ./bin/wizki -root viaggio-giappone upgrade -apply

wizki creates what's missing, replaces the outdated settings files (keeping each
old one as a `.bak` copy, in case she had customized something), and rebuilds
the catalog. **Her pages are never touched.** Running `upgrade` again now says
`… è già aggiornata` (*… is already up to date*). Older pages may lack newer
fields, such as the publication date; the next check-up will point them out.

### 12. Keeping the originals on the family NAS

After a year, Giulia's wiki holds hundreds of scans and photos, and her laptop
is running out of space. The family has a **NAS**, a small storage box on the
home network, which her computer already shows as a folder: `/mnt/nas/wiki`.
(It's the same for any network folder, as long as the computer shows it as an
ordinary folder: a NAS, a shared folder on another computer, a remote server
mounted with sshfs, a cloud drive mounted with rclone.)

She asks wizki to move the safe, `raw/`, there. As always, the first run is
only a trial:

    ./bin/wizki raw /mnt/nas/wiki

    prova: nessuna modifica. Rilancia con -apply per applicare:
    copia con verifica: 412 file in /mnt/nas/wiki
    sposta: library/raw -> library/raw.local-20261108-202121 (la copia locale resta lì)
    collega: library/raw -> /mnt/nas/wiki
    crea: /mnt/nas/wiki/.wizki-raw

*(trial run: no changes. Copy with verification: 412 files to /mnt/nas/wiki;
move: library/raw → library/raw.local-… (the local copy stays there); link:
library/raw → /mnt/nas/wiki; create: /mnt/nas/wiki/.wizki-raw)*

Then she runs it for real:

    ./bin/wizki raw /mnt/nas/wiki -apply

Here is what wizki does:

1. **It copies every original to the NAS and checks each copy.** It computes the
   fingerprint of the copy and compares it with the original's, so a file
   damaged in transit is caught immediately. Files already on the NAS with the
   same content are skipped. If the NAS already has a file with the same name
   but different content, wizki stops before copying anything.
2. **It keeps the old folder** as `raw.local-<date>`. Nothing is deleted. Once
   Giulia has checked that everything is on the NAS, she can delete that folder
   herself to free up space.
3. **It turns `raw` into a shortcut** (a "symbolic link") to the NAS folder. The
   pages keep saying `raw/pdf/circolare.pdf`, and that path now leads to the NAS.
   The assistant, Obsidian and every link keep working as before.
4. **It leaves a small label file**, `.wizki-raw`, in the NAS folder. It records
   which wiki the folder belongs to, so the same folder can't be used by two
   wikis by mistake.

From now on, new documents go straight from the inbox to the NAS. They are
copied, checked, and only then removed from the inbox.

**When the NAS is off.** If the NAS is switched off, or the laptop is away from
home, the folder `/mnt/nas/wiki` may look empty or be missing. wizki notices
the label file is missing and **refuses to file anything**, so that nothing
ends up in the wrong place:

    errore: raw remoto non raggiungibile: in /mnt/nas/wiki manca .wizki-raw: la condivisione è montata? (se è una cartella nuova, configurala con wizki raw)

*(error: remote raw unreachable: .wizki-raw is missing in /mnt/nas/wiki: is the
share mounted? If it's a new folder, set it up with wizki raw)*

The document simply waits in the inbox. When she's home again, she adds it
as usual. Reading the wiki, asking questions and running the check-up keep
working without the NAS, because the pages stay on the laptop. Only the
originals and their extracted text are on the NAS.

To avoid rereading hundreds of files over the network every time she adds a
document (to spot duplicates), wizki keeps a small list of fingerprints on the
laptop and only rereads files that have changed.

### 13. Small helpers

A few commands mostly run behind the scenes, but can be useful by hand.

**Rebuild the catalog** after editing pages yourself:

    ./bin/wizki index

    library/index.md

The catalog lists the overview, all topics, entities grouped by kind, analyses
and documents (newest first), each with its one-line description and how many
documents mention it:

```markdown
## Topic (2)

- [[topics/orario|orario]] — School hours · 3 fonti
- [[topics/trasporto|trasporto]] — School buses · 1 fonte
```

Never edit `index.md` by hand: the next rebuild would undo your changes.

**Write in the diary** by hand, for example after a session you handled
yourself. The first word must be `ingest`, `query` or `lint`:

    ./bin/wizki log query "Bus routes compared"

    ## [2026-12-10] query | Bus routes compared

The diary can be searched with simple tools. For example, this shows the last
five entries:

    grep "^## \[" library/log.md | tail -5

**See how a title becomes a file name**, lowercase, without accents or symbols:

    ./bin/wizki slug "Circolare N° 12/2026: orari"

    circolare-n-12-2026-orari

---

## When something goes wrong

wizki is careful. When it refuses something, the document stays in the inbox
exactly as it was. Here are the messages you're most likely to meet:

| Message | Meaning | What to do |
|---|---|---|
| `fonte già acquisita: stesso contenuto di raw/data/enrolment-2026.csv (id 69f190a3)` | *Source already added.* This file has exactly the same content as one already in the wiki, even if its name is different. | Nothing: delete the copy from the inbox. |
| `nessun testo nella fonte: empty-note.txt` | *No text in the source.* The file is empty. | Check the file; maybe it didn't save properly. |
| `"Minutes.docx": tipo non supportato (estensioni: …)` | *Type not supported.* wizki can't read this kind of file. | Save it as PDF (in Word: *File → Save as → PDF*) and add the PDF. |
| `nessun testo estratto dal PDF` | *No text extracted from the PDF.* It's a scan and OCR was switched off or couldn't read anything. | Try a clearer scan, or make sure OCR is available. |
| `OCR non disponibile: manca il pacchetto lingua ita di Tesseract` | *OCR not available: the Italian language pack is missing.* | Install the Italian pack for tesseract. |
| `niente OCR dell'immagine (…): salvo solo il riferimento` | *No OCR for the image: saving only the reference.* Not an error: the picture is filed, but its text wasn't read automatically. | Fine as it is: the assistant can still look at the picture. |
| `immagini locali non trovate (restano come riferimenti): …` | *Local images not found (left as references).* An article mentions a picture that isn't in the inbox. | Add the picture next to the article and try again, or ignore it. |
| `… esiste già: raw/ è immutabile, non sovrascrivo` | *Already exists: raw/ cannot be changed, I won't overwrite.* A file with the same name is already in the safe. | Rename the new document and try again. |
| `manca git, che serve a init: installalo e rilancia (non ho creato nulla)` | *git is missing, and init needs it: install it and try again (nothing was created).* | Install git (`wizki deps` shows the command), then run `init` again. |
| `raw remoto non raggiungibile: … la condivisione è montata?` | *Remote raw unreachable: is the share mounted?* The network folder holding the originals isn't available. Nothing was moved. | Switch on or connect the NAS (or reconnect to the network), then try again. |
| `<absolute path> non sembra una wiki (manca .zk/config.toml): per crearne una usa wizki init` | *This doesn't look like a wiki.* The command was pointed at the wrong folder; the message shows the absolute path it checked. | Check the `-root` name (a relative path starts from the current folder); use `init` for a brand-new wiki. |

---

## The rules your wiki lives by

These rules are written in `CLAUDE.md`, and the assistant follows them every
time:

- **Originals are sacred.** Nothing in `raw/` is ever modified or overwritten.
- **You decide.** The assistant proposes; it creates pages, applies fixes, saves
  answers or searches the web only after your OK.
- **One name per thing.** Subjects are Italian singular nouns, and they are
  reused rather than duplicated; people and organizations go by their most
  common full name ("Ministero dell'Istruzione e del Merito", not "MIM").
- **Every claim has a source.** Pages cite the documents they come from, and
  contradictions between documents are flagged, not hidden.
- **Dates matter.** Newer documents update the picture; older ones become history.
- **Links are complete**: they always include the page's location and its title.
- **The catalog is generated**, never written by hand.
- **Subjects are not folders.** "Everything about `orario`" is a search the
  assistant runs, not a folder you maintain.

You can change these rules: edit `CLAUDE.md` (it's in Italian, with an English
translation below it), or simply ask the assistant to improve them with you.

---

## Glossary

- **Analysis**: a saved answer to a question, with the pages it relies on.
- **Catalog** (`index.md`): the list of every page with a one-line description.
- **Claude Code**: the AI assistant, used in the terminal, that writes the wiki.
- **Entity**: a person, organization, place, law, work or event with its own page.
- **Extract**: put a document in the safe and pull its text out.
- **ID**: a short code computed from a document's content, like a fingerprint.
  Two identical files always have the same ID.
- **Inbox**: the folder where you drop new documents.
- **Ingest**: the whole process of adding a document to the wiki.
- **Lint** / **check-up**: the wiki's health check.
- **Markdown**: the simple text format the pages are written in.
- **OCR**: reading text from an image or a scan.
- **Overview** (`overview.md`): the big picture of the whole wiki.
- **Raw**: the safe where originals are kept untouched.
- **Slug**: a title turned into a file name (`Circolare N° 12` → `circolare-n-12`).
- **Terminal**: the window where you type commands.
- **Topic**: a subject, with a page that grows as documents arrive.
- **zk**: the program that keeps track of pages, tags and links.

---

## Reference for the curious

Every command, with its options:

    wizki [-root library] [-ocr-lang ita] [-no-ocr] extract <file>
    wizki [-root library] inbox
    wizki [-root library] log <ingest|query|lint> <title>
    wizki [-root library] lint
    wizki [-root library] index
    wizki [-root library] upgrade [-apply]
    wizki [-root library] raw <folder> [-apply]
    wizki [-locale it|en] slug <text>
    wizki [-locale it|en] [-ocr-lang ita] deps
    wizki [-ocr-lang ita] init <name> [--locale it|en]

- `-root`: which wiki to work on (default `library`); can also be set with the
  `WIKI_ROOT` environment variable.
- `-ocr-lang`: OCR language(s), e.g. `ita`, `eng`, `ita+eng` (default `ita`);
  can also be set with `WIKI_OCR_LANG`.
- `-no-ocr`: never try OCR.
- `-locale`: wizki's language (`it`, default, or `en`) for commands without a
  wiki and for new wikis; can also be set with `WIKI_LOCALE`. A command working
  on an existing wiki always uses that wiki's language.
- `--locale` after `init <name>`: the new wiki's language.

What `extract` accepts, and where each kind of file goes:

| Kind | Extensions | Original goes to | Extracted text |
|---|---|---|---|
| PDF | `.pdf` | `raw/pdf/` | text via pdftotext; OCR if scanned (no text, or very little) |
| Text | `.md` `.markdown` `.txt` `.html` `.htm` | `raw/doc/` (pictures in `raw/assets/<name>/`) | the content (HTML converted to Markdown) |
| Picture | `.png` `.jpg` `.jpeg` `.webp` `.gif` `.tif` `.tiff` | `raw/assets/` | link to the picture + OCR text |
| Data | `.csv` `.tsv` `.json` | `raw/data/` | the content |
| E-book | `.epub` | `raw/epub/` (pictures in `raw/assets/<name>/`) | metadata, then the chapters in reading order (HTML converted to Markdown); DRM-protected books are refused |

Exit codes (useful in scripts): 0 ok; 1 error, warnings from `lint`, or programs missing for `deps`; 2 wrong
usage; 3 no text (even with OCR, or a DRM-protected EPUB); 4 OCR not available; 5 source already added;
6 network `raw/` unreachable.

wizki is written in Go, with no dependencies beyond the standard library. It
calls `pdftotext`, `pdftoppm`, `tesseract`, `zk` and `git` when needed. The
code is split into `core/` (pure logic, fully tested), `fx/` (everything that
touches files and other programs) and `i18n/` (the dictionary: every text wizki
prints or writes, one struct per language). The zk settings and templates for
Italian are `library/.zk/`; for English, `locales/en/zk/`. Run the tests with `go test ./...`.

---

## Learn more

- `CLAUDE.md`: the complete rule book the assistant follows, step by step.
- Andrej Karpathy's original note on the
  [LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f)
  idea.
- Optional: the pages are plain Markdown with links in the same format as
  [Obsidian](https://obsidian.md), so you can open the `library/` folder in
  Obsidian to browse the wiki and see its links as a graph.
