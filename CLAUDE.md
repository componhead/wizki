# Wiki personale (notebook zk in `library/`)

Pattern "LLM Wiki": le fonti grezze sono immutabili, la wiki è markdown
che mantieni tu, questo file è lo schema. Strumenti: `zk` (note, tag, link)
e `wiki` (binario Go di questo repo: estrazione PDF, log, lint).

    go build -o wiki .        # una volta; poi ./wiki ...

## Struttura

    library/
    ├── inbox/            PDF da elaborare
    ├── raw/pdf/          PDF originali (mai modificare)
    ├── raw/text/         testo estratto (mai modificare)
    ├── sources/          una nota per fonte: riassunto + tag + [[topic]]
    ├── topics/           un hub per topic: sintesi cumulativa
    └── log.md            registro in sola aggiunta

## Ingestion di un PDF

1. `./wiki extract library/inbox/<file>.pdf` -> stampa il path del testo.
   Se il PDF è una scansione, `wiki extract` fa da solo l'OCR (pdftoppm + tesseract,
   lingua `-ocr-lang`, default `ita`) e lo segnala su stderr: l'OCR sbaglia qualche
   carattere, quindi leggi il testo con sospetto. Exit code 3 = nessun testo nemmeno
   con l'OCR; 4 = OCR non disponibile (mancano comandi o lingua). In entrambi i casi
   il PDF resta in inbox: fermati e dimmelo.
2. Leggi il testo. Individua 1-3 topic: NOMI, italiano, minuscolo, singolare.
   Un topic conta solo se centrale, non se citato di passaggio.
3. `zk topics` per vedere i topic esistenti. Riusali. Creane uno nuovo solo
   se nessuno copre il contenuto. Mai plurali o sinonimi di topic esistenti.
4. Mostrami titolo, topic scelti e perché. Procedi dopo la mia conferma.
5. Nota-fonte: `cd library && zk new sources --title "<titolo>" --print-path`
   poi scrivi: riassunto di 5 righe, tag `#<topic>`, link `[[topics/<topic>]]`,
   `pdf:` con il path in `raw/pdf/`.
6. Per ogni topic: se manca `topics/<topic>.md` crealo con
   `zk new topics --title "<topic>" --print-path`.
   Aggiorna la sintesi integrando la nuova fonte, citala con `[[sources/<nota-fonte>]]`
   e segnala sotto "Contraddizioni" ciò che confligge con quanto già scritto.
7. Se due topic compaiono nella stessa fonte, linkali in "Vedi anche".
8. `./wiki log ingest "<titolo>"`.

Una fonte può toccare 10-15 pagine: è normale.

## Query

Rispondi leggendo prima i topic rilevanti (`zk by <topic>`, `zk list --link-to topics/<topic>.md`),
poi le fonti. Se la risposta è un'analisi riutilizzabile, proponi di salvarla
come nota e registra `./wiki log query "<titolo>"`.

## Lint (su richiesta)

`./wiki lint` (exit 1 se ci sono warn): tag senza link e viceversa, plurali o
sinonimi tra topic, note orfane, topic con una sola fonte. Poi controlla a mano
le contraddizioni tra le pagine topic. Registra con `./wiki log lint "<data>"`.

## Regole

- Non modificare mai `raw/`. `wiki extract` rifiuta di sovrascrivere.
- Non creare cartelle tematiche né symlink: le viste per argomento sono query zk.
- Wikilink SEMPRE con percorso: `[[topics/scuola]]`, `[[sources/20261108-titolo]]`.
  zk risolve `[[scuola]]` con un match parziale sul nome file e potrebbe puntare
  a una fonte il cui nome contiene "scuola"; il percorso esatto ha la precedenza.
- `log.md` è escluso dall'indice di zk (vedi `exclude` in `.zk/config.toml`).
