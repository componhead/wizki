# Personal wiki (zk notebook in `library/`)

"LLM Wiki" pattern: raw sources are immutable, the wiki is markdown
that you maintain, this file is the schema. Tools: `zk` (notes, tags, links)
and `wiki` (this repo's Go binary: PDF extraction, log, lint).

    go build -o wiki .        # once; then ./wiki ...

## Structure

    library/
    ├── inbox/            PDFs to process
    ├── raw/pdf/          original PDFs (never modify)
    ├── raw/text/         extracted text (never modify)
    ├── sources/          one note per source: summary + tags + [[topic]]
    ├── topics/           one hub per topic: cumulative synthesis
    └── log.md            append-only log

## Ingesting a PDF

1. `./wiki extract library/inbox/<file>.pdf` -> prints the path of the text.
   If the PDF is a scan, `wiki extract` runs OCR on its own (pdftoppm + tesseract,
   language `-ocr-lang`, default `ita`) and reports it on stderr: OCR gets a few
   characters wrong, so read the text with suspicion. Exit code 3 = no text even
   with OCR; 4 = OCR not available (missing commands or language). In both cases
   the PDF stays in inbox: stop and tell me.
2. Read the text. Identify 1-3 topics: NOUNS, Italian, lowercase, singular.
   A topic counts only if it is central, not if mentioned in passing.
3. `zk topics` to see the existing topics. Reuse them. Create a new one only
   if none covers the content. Never plurals or synonyms of existing topics.
4. Show me the title, the chosen topics and why. Proceed after my confirmation.
5. Source note: `cd library && zk new sources --title "<title>" --print-path`
   then write: a 5-line summary, tag `#<topic>`, link `[[topics/<topic>]]`,
   `pdf:` with the path in `raw/pdf/`.
6. For each topic: if `topics/<topic>.md` is missing, create it with
   `zk new topics --title "<topic>" --print-path`.
   Update the synthesis by integrating the new source, cite it with `[[sources/<source-note>]]`
   and flag under "Contraddizioni" (Contradictions) whatever conflicts with what is already written.
7. If two topics appear in the same source, link them under "Vedi anche" (See also).
8. `./wiki log ingest "<title>"`.

A single source may touch 10-15 pages: that's normal.

## Query

Answer by first reading the relevant topics (`zk by <topic>`, `zk list --link-to topics/<topic>.md`),
then the sources. If the answer is a reusable analysis, offer to save it
as a note and record `./wiki log query "<title>"`.

## Lint (on request)

`./wiki lint` (exit 1 if there are warnings): tags without links and vice versa, plurals or
synonyms among topics, orphan notes, topics with a single source. Then manually check
for contradictions between topic pages. Record with `./wiki log lint "<date>"`.

## Rules

- Never modify `raw/`. `wiki extract` refuses to overwrite.
- Don't create thematic folders or symlinks: per-topic views are zk queries.
- Wikilinks ALWAYS with the path: `[[topics/scuola]]`, `[[sources/20261108-titolo]]`.
  zk resolves `[[scuola]]` with a partial match on the file name and might point
  to a source whose name contains "scuola"; the exact path takes precedence.
- `log.md` is excluded from the zk index (see `exclude` in `.zk/config.toml`).
