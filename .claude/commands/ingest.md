---
description: Acquisisce una fonte dalla inbox nella wiki zk
argument-hint: "[@library/inbox/<file>]  (Tab dopo @ per completare; vuoto = scegli dalla inbox)"
allowed-tools: Bash(wizki inbox:*)
---

Argomento ricevuto: `$ARGUMENTS`

Fonti in inbox, nell'ordine di acquisizione (`wizki inbox`):

!`wizki inbox 2>&1`

Se l'argomento è vuoto, proponimi le fonti elencate sopra come scelta
(AskUserQuestion, la prima marcata come consigliata; se sono più di 4, le prime 4)
e procedi con quella che scelgo. Se la inbox è vuota, dimmelo e fermati.

Se l'argomento c'è, ricava il file: togli un eventuale `@` iniziale e, se non
inizia già con `library/inbox/`, anteponilo. Se il file non esiste, dimmelo e
proponimi le fonti elencate sopra. Non leggere il file direttamente (anche se
allegato con `@`): il testo si ottiene solo con `wizki extract`.

Poi esegui l'ingestion del file seguendo la sezione "Ingestion di una fonte" di
CLAUDE.md. Fermati al punto 5 e attendi la mia conferma su punti chiave, topic
ed entità prima di creare note.
