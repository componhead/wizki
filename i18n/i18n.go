// Package i18n è il dizionario di wizki: ogni testo che il programma mostra o
// scrive nella wiki (messaggi, errori, lint, indice, panoramica) sta in una
// Dict, una per lingua. Le voci sono campi di una struct, non chiavi di una
// mappa: una voce sbagliata non compila, e il test controlla che ogni lingua
// le abbia tutte, con gli stessi verbi di formato.
//
// La lingua corrente si sceglie una volta all'avvio (Use) e si legge con T().
// Quella di una wiki è il campo language di .zk/config.toml.
package i18n

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultCode è la lingua usata quando nessuno ne sceglie un'altra.
const DefaultCode = "it"

// Plural è una parola (o una frase) al singolare e al plurale.
type Plural struct{ One, Many string }

// Count: "1 <One>" oppure "<n> <Many>".
func (p Plural) Count(n int) string {
	if n == 1 {
		return "1 " + p.One
	}
	return fmt.Sprintf("%d %s", n, p.Many)
}

// dicts sono le lingue disponibili, per codice.
var dicts = map[string]*Dict{IT.Code: &IT, EN.Code: &EN}

// cur è la lingua corrente.
var cur = &IT

// T restituisce il dizionario della lingua corrente.
func T() *Dict { return cur }

// Use rende d la lingua corrente.
func Use(d *Dict) { cur = d }

// Lookup cerca una lingua per codice (es. "it", "en"; maiuscole e spazi non contano).
func Lookup(code string) (*Dict, bool) {
	d, ok := dicts[strings.ToLower(strings.TrimSpace(code))]
	return d, ok
}

// Codes elenca i codici delle lingue disponibili, in ordine.
func Codes() []string {
	out := make([]string, 0, len(dicts))
	for c := range dicts {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// Err è un errore sentinella (da confrontare con errors.Is) il cui testo
// segue la lingua corrente nel momento in cui viene stampato.
type Err struct{ msg func(*Dict) string }

// NewErr crea un errore sentinella con il testo scelto da msg.
func NewErr(msg func(*Dict) string) *Err { return &Err{msg: msg} }

// Error restituisce il testo dell'errore nella lingua corrente.
func (e *Err) Error() string { return e.msg(T()) }
