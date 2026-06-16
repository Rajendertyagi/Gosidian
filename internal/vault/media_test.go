package vault

import "testing"

const sampleMediaNote = `---
title: Diagramma plancia
type: image
media: proj/attachments/abc.png
tags: [proj, type:image]
---

La plancia in modalità tiling con tre finestre affiancate.
`

func TestMediaRefForNote_FlagGating(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	// The bytes are irrelevant: MediaRefForNote validates the extension and
	// stats the file, it does not sniff content.
	if err := v.Save("proj/attachments/abc.png", []byte("\x89PNG\r\n\x1a\nfake")); err != nil {
		t.Fatal(err)
	}
	if err := v.Save("proj/diagram.md", []byte(sampleMediaNote)); err != nil {
		t.Fatal(err)
	}
	n, err := v.Load("proj/diagram.md")
	if err != nil {
		t.Fatal(err)
	}

	// Flag off (default): a type:image note is treated as ordinary markdown.
	if ref, ok := v.MediaRefForNote(n.Path, n.Content); ok || ref != nil {
		t.Fatalf("flag off: want (nil,false), got (%+v,%v)", ref, ok)
	}

	// Flag on: the pointer resolves to the image attachment.
	v.SetMediaNotes(true)
	ref, ok := v.MediaRefForNote(n.Path, n.Content)
	if !ok {
		t.Fatal("flag on: type:image note should be recognised")
	}
	if ref.Broken {
		t.Fatalf("expected resolvable media, got broken: %+v", ref)
	}
	if ref.Path != "proj/attachments/abc.png" {
		t.Errorf("path = %q, want proj/attachments/abc.png", ref.Path)
	}
	if ref.URL != "/vault-files/proj/attachments/abc.png" {
		t.Errorf("url = %q", ref.URL)
	}
	if ref.MIME != "image/png" {
		t.Errorf("mime = %q, want image/png", ref.MIME)
	}
	if ref.Size == 0 {
		t.Error("size should be non-zero for an existing attachment")
	}
}

func TestMediaRefForNote_NonMedia(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	v.SetMediaNotes(true)

	// Ordinary note with a different type → not a media note.
	if err := v.Save("proj/plain.md", []byte("---\ntitle: plain\ntype: memory\ntags: [proj, type:memory]\n---\n\nbody")); err != nil {
		t.Fatal(err)
	}
	n, _ := v.Load("proj/plain.md")
	if ref, ok := v.MediaRefForNote(n.Path, n.Content); ok || ref != nil {
		t.Fatalf("non-media note: want (nil,false), got (%+v,%v)", ref, ok)
	}

	// Note without any frontmatter → not a media note.
	if err := v.Save("proj/bare.md", []byte("just text, no frontmatter")); err != nil {
		t.Fatal(err)
	}
	nb, _ := v.Load("proj/bare.md")
	if _, ok := v.MediaRefForNote(nb.Path, nb.Content); ok {
		t.Error("note without frontmatter must not be a media note")
	}
}

func TestMediaRefForNote_BrokenPointers(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	v.SetMediaNotes(true)
	// A real non-image file so the "non-image ext" case finds a file but a bad kind.
	if err := v.Save("proj/attachments/note.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}

	cases := []struct{ name, file, body string }{
		{"missing media key", "proj/m1.md", "---\ntitle: x\ntype: image\ntags: [proj, type:image]\n---\n\ncap"},
		{"pointer not found", "proj/m2.md", "---\ntitle: x\ntype: image\nmedia: proj/attachments/nope.png\n---\n\ncap"},
		{"non-image ext", "proj/m3.md", "---\ntitle: x\ntype: image\nmedia: proj/attachments/note.txt\n---\n\ncap"},
		{"path escape", "proj/m4.md", "---\ntitle: x\ntype: image\nmedia: ../../etc/passwd\n---\n\ncap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := v.Save(tc.file, []byte(tc.body)); err != nil {
				t.Fatal(err)
			}
			n, _ := v.Load(tc.file)
			ref, ok := v.MediaRefForNote(n.Path, n.Content)
			if !ok {
				t.Fatal("a type:image note must be recognised (kind set) even with a broken pointer")
			}
			if !ref.Broken {
				t.Errorf("expected Broken=true, got %+v", ref)
			}
		})
	}
}
