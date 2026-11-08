package core

import (
	"reflect"
	"testing"
	"time"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Scuola":                   "scuola",
		"  Circolare N° 12/2026 ":  "circolare-n-12-2026",
		"Perché l'Università è...": "perche-l-universita-e",
		"già--fatto":               "gia-fatto",
		"///":                      "",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlanExtract(t *testing.T) {
	steps, err := PlanExtract("library", "inbox/Circolare 2026.PDF")
	if err != nil {
		t.Fatal(err)
	}
	want := []Step{
		Mkdir{"library/raw/pdf"},
		Mkdir{"library/raw/text"},
		Move{From: "inbox/Circolare 2026.PDF", To: "library/raw/pdf/circolare-2026.pdf"},
		Extract{PDF: "library/raw/pdf/circolare-2026.pdf", Out: "library/raw/text/circolare-2026.md"},
	}
	if !reflect.DeepEqual(steps, want) {
		t.Errorf("got %#v", steps)
	}
	if _, err := PlanExtract("library", "note.txt"); err == nil {
		t.Error("atteso errore per file non PDF")
	}
}

func TestLogLine(t *testing.T) {
	day := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	got, err := LogLine(day, "ingest", "  Circolare \n 2026 ")
	if err != nil || got != "## [2026-11-08] ingest | Circolare 2026\n" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := LogLine(day, "boh", "x"); err == nil {
		t.Error("atteso errore per tipo sconosciuto")
	}
	if _, err := LogLine(day, "ingest", "   "); err == nil {
		t.Error("atteso errore per titolo vuoto")
	}
}

func TestDiff(t *testing.T) {
	got := Diff([]string{"c", "a", "b", "a"}, []string{"b"})
	if !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Errorf("got %v", got)
	}
}

func TestTopicFindings(t *testing.T) {
	fs := TopicFindings(TopicView{
		Topic:  "scuola",
		Tagged: []string{"sources/a.md", "sources/b.md"},
		Linked: []string{"sources/a.md", "sources/c.md"},
	})
	if len(fs) != 2 || !HasWarn(fs) {
		t.Fatalf("got %v", fs)
	}
	one := TopicFindings(TopicView{Topic: "x", Tagged: []string{"s/a.md"}, Linked: []string{"s/a.md"}})
	if len(one) != 1 || one[0].Level != "info" || HasWarn(one) {
		t.Errorf("got %v", one)
	}
	none := TopicFindings(TopicView{Topic: "y"})
	if len(none) != 1 || none[0].Msg != "topic senza fonti" {
		t.Errorf("got %v", none)
	}
}

func TestNearDuplicates(t *testing.T) {
	fs := NearDuplicates([]string{"scuole", "economia", "scuola", "salute"})
	if len(fs) != 1 || fs[0].Subject != "scuola ~ scuole" {
		t.Errorf("got %v", fs)
	}
}

func TestUnderDir(t *testing.T) {
	p := UnderDir("sources")
	if !p("sources/a.md") || p("topics/sources.md") || p("sourcesx/a.md") {
		t.Error("predicato errato")
	}
}

func TestSortPages(t *testing.T) {
	got := SortPages([]string{"/t/p-10.png", "/t/p-2.png", "/t/p-1.png"})
	want := []string{"/t/p-1.png", "/t/p-2.png", "/t/p-10.png"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	if got := SortPages([]string{"/t/p-02.png", "/t/p-01.png", "/t/p-10.png"}); got[0] != "/t/p-01.png" || got[2] != "/t/p-10.png" {
		t.Errorf("zero-padded: %v", got)
	}
}

func TestJoinPages(t *testing.T) {
	if got := JoinPages([]string{" a \n", "", "  ", "b"}); got != "a\n\nb\n" {
		t.Errorf("got %q", got)
	}
	if JoinPages([]string{" ", "\n"}) != "" {
		t.Error("solo spazi: atteso vuoto")
	}
}

func TestLangs(t *testing.T) {
	av := ParseLangs("List of available languages (3):\neng\nita\nosd\n")
	if !reflect.DeepEqual(av, []string{"eng", "ita", "osd"}) {
		t.Errorf("langs = %v", av)
	}
	if m := MissingLangs(av, "ita+eng"); len(m) != 0 {
		t.Errorf("missing = %v", m)
	}
	if m := MissingLangs(av, "ita+deu"); !reflect.DeepEqual(m, []string{"deu"}) {
		t.Errorf("missing = %v", m)
	}
}
