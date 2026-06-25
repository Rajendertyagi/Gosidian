// Package qrsvg renders a QR code as a self-contained SVG document (for the web
// UI) or as an ANSI block grid (for the CLI). It exists so the TOTP enrolment
// flow can hand the SPA a scannable QR inline — no <img> data: URI, hence no
// img-src CSP relaxation — without pulling a new dependency: boombuler/barcode
// is already in the module graph via github.com/pquerna/otp.
package qrsvg

import (
	"fmt"
	"strings"

	"github.com/boombuler/barcode/qr"
)

// quietZone is the mandatory light border around a QR symbol, in modules. The
// spec recommends 4; scanners are unreliable without it.
const quietZone = 4

// encode produces the unscaled QR symbol (1 pixel == 1 module) for text. Medium
// error-correction balances density against scan resilience for short otpauth
// URIs; Auto picks the smallest fitting version.
func encode(text string) (matrix [][]bool, err error) {
	if text == "" {
		return nil, fmt.Errorf("qrsvg: empty text")
	}
	code, err := qr.Encode(text, qr.M, qr.Auto)
	if err != nil {
		return nil, fmt.Errorf("qrsvg: encode: %w", err)
	}
	n := code.Bounds().Dx() // square symbol: Dx == Dy
	m := make([][]bool, n)
	for y := 0; y < n; y++ {
		m[y] = make([]bool, n)
		for x := 0; x < n; x++ {
			// boombuler renders dark modules as black (R==0); light as white.
			r, _, _, _ := code.At(x, y).RGBA()
			m[y][x] = r == 0
		}
	}
	return m, nil
}

// SVG encodes text as a QR code and returns a standalone SVG document. The grid
// is laid out at 1 user-unit per module plus a quiet zone, with a viewBox but no
// fixed width/height — the caller sizes it via CSS. shape-rendering=crispEdges
// keeps the modules sharp when scaled. The markup contains no <script> and no
// inline style attributes, so it is safe to inject under a strict CSP.
func SVG(text string) (string, error) {
	m, err := encode(text)
	if err != nil {
		return "", err
	}
	n := len(m)
	dim := n + quietZone*2

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges" role="img">`, dim, dim)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#ffffff"/>`, dim, dim)
	// One <rect> per dark module. Coalescing horizontal runs would shrink the
	// output, but TOTP QRs are tiny (~25-33 modules) so per-module rects keep
	// the renderer obvious and the payload well under a kilobyte.
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if m[y][x] {
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="1" height="1" fill="#000000"/>`, x+quietZone, y+quietZone)
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.String(), nil
}

// Terminal encodes text as a QR code rendered with ANSI background colours: two
// spaces per module so the grid is roughly square, dark modules on a black
// background and light modules (plus the quiet zone) on white. Setting explicit
// background colours makes it scannable regardless of the terminal's own theme.
func Terminal(text string) (string, error) {
	m, err := encode(text)
	if err != nil {
		return "", err
	}
	n := len(m)
	const (
		white = "\x1b[47m  \x1b[0m"
		black = "\x1b[40m  \x1b[0m"
	)
	dim := n + quietZone*2
	whiteRow := strings.Repeat(white, dim) + "\n"

	var b strings.Builder
	for i := 0; i < quietZone; i++ {
		b.WriteString(whiteRow)
	}
	for y := 0; y < n; y++ {
		b.WriteString(strings.Repeat(white, quietZone))
		for x := 0; x < n; x++ {
			if m[y][x] {
				b.WriteString(black)
			} else {
				b.WriteString(white)
			}
		}
		b.WriteString(strings.Repeat(white, quietZone))
		b.WriteByte('\n')
	}
	for i := 0; i < quietZone; i++ {
		b.WriteString(whiteRow)
	}
	return b.String(), nil
}
