package i18n

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"testing"
)

// verbRe riconosce un verbo di formato (es. %s, %q, %-*s, %%).
var verbRe = regexp.MustCompile(`%[-+# 0]*\d*(\.\d+)?[a-zA-Z%]`)

// verbs restituisce i verbi di formato di s, in ordine, esclusi i "%%".
func verbs(s string) []string {
	var out []string
	for _, v := range verbRe.FindAllString(s, -1) {
		if v != "%%" {
			out = append(out, v)
		}
	}
	return out
}

// texts appiattisce un Dict: nome della voce -> testo (i Plural danno due voci).
func texts(d Dict) map[string]string {
	out := map[string]string{}
	v := reflect.ValueOf(d)
	for i := range v.NumField() {
		name, f := v.Type().Field(i).Name, v.Field(i)
		switch x := f.Interface().(type) {
		case string:
			out[name] = x
		case Plural:
			out[name+".One"], out[name+".Many"] = x.One, x.Many
		default:
			panic(fmt.Sprintf("tipo non previsto per %s: %T", name, x))
		}
	}
	return out
}

// TestDizionariCompleti: ogni lingua ha tutte le voci, e le stesse voci hanno
// gli stessi verbi di formato in tutte le lingue (altrimenti fmt stamperebbe
// %!s(MISSING) o argomenti in più).
func TestDizionariCompleti(t *testing.T) {
	ref := texts(IT)
	for _, code := range Codes() {
		d, _ := Lookup(code)
		got := texts(*d)
		for name, s := range got {
			if s == "" {
				t.Errorf("%s: voce %s vuota", code, name)
			}
			if !slices.Equal(verbs(s), verbs(ref[name])) {
				t.Errorf("%s: voce %s con verbi %v, in it %v", code, name, verbs(s), verbs(ref[name]))
			}
		}
		if d.Code != code {
			t.Errorf("Lookup(%q).Code = %q", code, d.Code)
		}
	}
}

// TestLookup: codici noti (anche con maiuscole e spazi) e sconosciuti.
func TestLookup(t *testing.T) {
	if d, ok := Lookup(" EN "); !ok || d != &EN {
		t.Error("Lookup(\" EN \") non trova l'inglese")
	}
	if _, ok := Lookup("fr"); ok {
		t.Error("Lookup(\"fr\") trova una lingua che non c'è")
	}
	if got := Codes(); !slices.Equal(got, []string{"en", "it"}) {
		t.Errorf("Codes() = %v", got)
	}
}

// TestPluralCount: singolare e plurale.
func TestPluralCount(t *testing.T) {
	p := Plural{"a", "b"}
	if p.Count(1) != "1 a" || p.Count(3) != "3 b" || p.Count(0) != "0 b" {
		t.Error("Plural.Count sbagliato")
	}
}

// TestErrSegueLaLingua: il testo di un Err cambia con la lingua corrente,
// ma errors.Is lo riconosce comunque, anche avvolto.
func TestErrSegueLaLingua(t *testing.T) {
	defer Use(T())
	e := NewErr(func(d *Dict) string { return d.ErrExists })
	wrapped := fmt.Errorf("x: %w", e)
	Use(&IT)
	if e.Error() != "esiste già" {
		t.Errorf("it: %q", e.Error())
	}
	Use(&EN)
	if e.Error() != "already exists" || !errors.Is(wrapped, e) {
		t.Errorf("en: %q", e.Error())
	}
}
