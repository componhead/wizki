package core

import (
	"reflect"
	"strings"
	"testing"
)

// TestRawMarker: il marcatore porta il nome della wiki e si rilegge.
func TestRawMarker(t *testing.T) {
	if got := RawMarkerWiki(string(RawMarkerContent("scuola"))); got != "scuola" {
		t.Errorf("RawMarkerWiki = %q", got)
	}
	if RawMarkerWiki("altro") != "" {
		t.Error("marcatore senza wiki: atteso vuoto")
	}
}

// TestPlanRawDaLocale: copia verificata dei file nuovi (salta quelli già
// uguali), backup della raw locale, link, marcatore.
func TestPlanRawDaLocale(t *testing.T) {
	st := RawState{
		Root: "w", Target: "/nas/raw",
		Local: []string{"pdf/b.pdf", "doc/a.md", "pdf/c.pdf"},
		Same:  map[string]bool{"pdf/c.pdf": true},
	}
	got, err := PlanRaw(st, "w", "S")
	if err != nil {
		t.Fatal(err)
	}
	want := []Step{
		Mkdir{"/nas/raw/doc"},
		CopyVerify{From: "w/raw/doc/a.md", To: "/nas/raw/doc/a.md"},
		Mkdir{"/nas/raw/pdf"},
		CopyVerify{From: "w/raw/pdf/b.pdf", To: "/nas/raw/pdf/b.pdf"},
		Move{From: "w/raw", To: "w/raw.local-S"},
		Symlink{Target: "/nas/raw", Link: "w/raw"},
		WriteFile{Path: "/nas/raw/.wizki-raw", Data: RawMarkerContent("w")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %#v\nwant %#v", got, want)
	}
}

// TestPlanRawCasi: senza raw locale solo link e marcatore; già collegata,
// niente (o solo il marcatore mancante); rifiuti per conflitti, link diverso,
// marcatore di un'altra wiki, path relativo.
func TestPlanRawCasi(t *testing.T) {
	got, _ := PlanRaw(RawState{Root: "w", Target: "/nas"}, "w", "S")
	if len(got) != 2 {
		t.Errorf("senza raw locale: %#v", got)
	}
	if got, _ := PlanRaw(RawState{Root: "w", Target: "/nas", LinkTo: "/nas/", MarkerWiki: "w"}, "w", "S"); got != nil {
		t.Errorf("già collegata: %#v", got)
	}
	if got, _ := PlanRaw(RawState{Root: "w", Target: "/nas", LinkTo: "/nas"}, "w", "S"); len(got) != 1 {
		t.Errorf("collegata senza marcatore: %#v", got)
	}
	bad := map[string]RawState{
		"contenuto diverso": {Root: "w", Target: "/nas", Local: []string{"pdf/a.pdf"}, Conflict: map[string]bool{"pdf/a.pdf": true}},
		"punta già":         {Root: "w", Target: "/nas", LinkTo: "/altro"},
		"della wiki":        {Root: "w", Target: "/nas", MarkerWiki: "altra"},
		"path assoluto":     {Root: "w", Target: "nas"},
	}
	for want, st := range bad {
		if _, err := PlanRaw(st, "w", "S"); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", want, err)
		}
	}
}

// TestHashCache: formato a righe, righe malformate ignorate, hash valido solo
// se dimensione e data coincidono.
func TestHashCache(t *testing.T) {
	es := map[string]HashEntry{
		"raw/pdf/b.pdf": {Path: "raw/pdf/b.pdf", Size: 2, MTime: 20, Sum: "bb"},
		"raw/pdf/a.pdf": {Path: "raw/pdf/a.pdf", Size: 1, MTime: 10, Sum: "aa"},
	}
	s := FormatHashCache(es)
	if s != "raw/pdf/a.pdf\t1\t10\taa\nraw/pdf/b.pdf\t2\t20\tbb\n" {
		t.Errorf("Format = %q", s)
	}
	back := ParseHashCache(s + "rotta\nx\tno\t1\tcc\n")
	if !reflect.DeepEqual(back, es) {
		t.Errorf("Parse = %#v", back)
	}
	if sum, ok := CachedSum(back, "raw/pdf/a.pdf", 1, 10); !ok || sum != "aa" {
		t.Error("hash in cache non trovato")
	}
	if _, ok := CachedSum(back, "raw/pdf/a.pdf", 1, 11); ok {
		t.Error("data cambiata: la cache non vale")
	}
}
