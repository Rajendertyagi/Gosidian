package qrsvg

import (
	"strings"
	"testing"
)

const sampleURI = "otpauth://totp/gosidian:alice?secret=JBSWY3DPEHPK3PXP&issuer=gosidian"

func TestSVG(t *testing.T) {
	out, err := SVG(sampleURI)
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	if !strings.HasPrefix(out, "<svg ") || !strings.HasSuffix(out, "</svg>") {
		t.Errorf("SVG output is not a well-formed svg document: %.60q…", out)
	}
	if !strings.Contains(out, "viewBox=") {
		t.Error("SVG output missing viewBox")
	}
	if !strings.Contains(out, `fill="#000000"`) {
		t.Error("SVG output has no dark modules")
	}
	// No script and no inline style attribute — must stay CSP-safe.
	if strings.Contains(out, "<script") || strings.Contains(out, "style=") {
		t.Error("SVG output contains script/style — would need CSP relaxation")
	}
}

func TestSVG_Deterministic(t *testing.T) {
	a, err := SVG(sampleURI)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SVG(sampleURI)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Error("SVG is not deterministic for identical input")
	}
}

func TestSVG_Empty(t *testing.T) {
	if _, err := SVG(""); err == nil {
		t.Error("expected error on empty text")
	}
}

func TestTerminal(t *testing.T) {
	out, err := Terminal(sampleURI)
	if err != nil {
		t.Fatalf("Terminal: %v", err)
	}
	if !strings.Contains(out, "\x1b[40m") || !strings.Contains(out, "\x1b[47m") {
		t.Error("Terminal output missing ANSI background colours")
	}
	if _, err := Terminal(""); err == nil {
		t.Error("expected error on empty text")
	}
}
