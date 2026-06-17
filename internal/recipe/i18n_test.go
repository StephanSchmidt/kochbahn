package recipe

import (
	"strings"
	"testing"
)

func TestLocalizeLanguages(t *testing.T) {
	r := &Recipe{}
	if err := r.Localize("en"); err != nil {
		t.Fatalf("Localize(en): %v", err)
	}
	if r.StartLabel() != "Start" || r.TotalLabel() != "Total time" {
		t.Errorf("en labels = %q/%q", r.StartLabel(), r.TotalLabel())
	}
	if err := r.Localize("de"); err != nil {
		t.Fatalf("Localize(de): %v", err)
	}
	if r.StartLabel() != "Anfang" || r.TotalLabel() != "Gesamtzeit" {
		t.Errorf("de labels = %q/%q", r.StartLabel(), r.TotalLabel())
	}
}

func TestLocalizeOverrides(t *testing.T) {
	r := &Recipe{Labels: map[string]string{"start": "Los geht's"}}
	if err := r.Localize("de"); err != nil {
		t.Fatalf("Localize: %v", err)
	}
	if r.StartLabel() != "Los geht's" {
		t.Errorf("override not applied: %q", r.StartLabel())
	}
	if r.TotalLabel() != "Gesamtzeit" {
		t.Errorf("un-overridden label changed: %q", r.TotalLabel())
	}
}

func TestLocalizeUnknownLang(t *testing.T) {
	r := &Recipe{}
	err := r.Localize("fr")
	if err == nil {
		t.Fatal("expected error for unknown lang")
	}
	if !strings.Contains(err.Error(), "fr") || !strings.Contains(err.Error(), "de") {
		t.Errorf("error = %q, want to name the bad and the available langs", err.Error())
	}
}

func TestStartLabelFallback(t *testing.T) {
	// A hand-built recipe that never called Localize still renders sensible text.
	r := &Recipe{}
	if r.StartLabel() != "Anfang" || r.TotalLabel() != "Gesamtzeit" {
		t.Errorf("fallback labels = %q/%q", r.StartLabel(), r.TotalLabel())
	}
}
