# wizki

A support tool for a personal wiki kept as a [zk](https://github.com/zk-org/zk)
notebook, following the "LLM Wiki" pattern: raw sources (PDFs and their extracted
text) are immutable, the wiki is markdown maintained by Claude Code, and
`CLAUDE.md` is the schema it follows.

It is an implementation of Andrej Karpathy's
[LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) idea,
built on a [zk](https://github.com/zk-org/zk) Zettelkasten notebook instead of
Obsidian: notes, tags and links are managed with `zk` from the command line, and
topic views are `zk` queries rather than folders.

## Build

    go build -o bin/wizki .

Prerequisites: Go 1.24+, `pdftotext` and `pdftoppm` (poppler), `tesseract` with the
`ita` language pack (only for scans), `zk` in the `PATH` (for `lint`).

## Usage

    wizki [-root library] [-ocr-lang ita] [-no-ocr] extract <file.pdf>
    wizki [-root library] log <ingest|query|lint> <title>
    wizki [-root library] lint
    wizki slug <text>

- `extract`: moves the PDF into `raw/pdf/` and extracts its text into `raw/text/`,
  printing the text path and, on the next line, the PDF's id (first 8 hex chars
  of its SHA-256). Scanned PDFs are read with OCR automatically
  (disable with `-no-ocr`). It refuses to overwrite existing files and refuses a
  PDF whose content is already in `raw/pdf/`, whatever its file name.
- `log`: appends an entry to `library/log.md`.
- `lint`: wiki health checks (tags without links and vice versa, plural or synonym
  topics, orphan notes, topics with a single source, missing or duplicate
  `id:` among source notes).
- `slug`: prints the slug of a text.

`-root` and `-ocr-lang` can also be set with the `WIKI_ROOT` and `WIKI_OCR_LANG`
environment variables.

Exit codes: 0 ok, 1 error (or `lint` warnings), 2 usage, 3 no text (even with OCR),
4 OCR not available, 5 PDF already ingested.

## Layout

    library/
    ├── inbox/            PDFs to process
    ├── raw/pdf/          original PDFs (never modify)
    ├── raw/text/         extracted text (never modify)
    ├── sources/          one note per source
    ├── topics/           one hub per topic
    └── log.md            append-only log

## Workflow with Claude Code

Open Claude Code in this folder and run `/ingest "<file>.pdf"` to ingest a PDF
from `library/inbox/`, or `/lint` for a health check. The rules Claude follows
are in `CLAUDE.md` (Italian first, English translation below).

To learn with sample PDFs, see the sibling project `../wizki-tutorial`.
