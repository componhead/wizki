# wizki

A support tool for a personal wiki kept as a [zk](https://github.com/zk-org/zk)
notebook, following the "LLM Wiki" pattern: raw sources (PDFs and their extracted
text) are immutable, the wiki is markdown maintained by Claude Code, and
`CLAUDE.md` is the schema it follows.

## Build

    go build -o wiki .

Prerequisites: Go 1.24+, `pdftotext` and `pdftoppm` (poppler), `tesseract` with the
`ita` language pack (only for scans), `zk` in the `PATH` (for `lint`).

## Usage

    wiki [-root library] [-ocr-lang ita] [-no-ocr] extract <file.pdf>
    wiki [-root library] log <ingest|query|lint> <title>
    wiki [-root library] lint
    wiki slug <text>

- `extract`: moves the PDF into `raw/pdf/` and extracts its text into `raw/text/`,
  printing the text path. Scanned PDFs are read with OCR automatically
  (disable with `-no-ocr`). It refuses to overwrite existing files.
- `log`: appends an entry to `library/log.md`.
- `lint`: wiki health checks (tags without links and vice versa, plural or synonym
  topics, orphan notes, topics with a single source).
- `slug`: prints the slug of a text.

`-root` and `-ocr-lang` can also be set with the `WIKI_ROOT` and `WIKI_OCR_LANG`
environment variables.

Exit codes: 0 ok, 1 error (or `lint` warnings), 2 usage, 3 no text (even with OCR),
4 OCR not available.

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
are in `CLAUDE.md` (English translation: `CLAUDE-en.md`).

To learn with sample PDFs, see the sibling project `../wizki-tutorial`.
