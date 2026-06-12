package index

import (
	"reflect"
	"testing"
)

func TestIndex_Hubs(t *testing.T) {
	idx := openTest(t)
	// hub.md links to four leaves → undirected degree 4; each leaf degree 1.
	upsert(t, idx, "hub.md", "Hub", "[[A]] [[B]] [[C]] [[D]]")
	upsert(t, idx, "a.md", "A", "leaf")
	upsert(t, idx, "b.md", "B", "leaf")
	upsert(t, idx, "c.md", "C", "leaf")
	upsert(t, idx, "d.md", "D", "leaf")
	// orphan.md has no links — must never appear among hubs.
	upsert(t, idx, "orphan.md", "Orphan", "nothing here")

	hubs, err := idx.Hubs("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hubs) != 5 {
		t.Fatalf("expected 5 hubs (orphan excluded), got %d: %+v", len(hubs), hubs)
	}
	if hubs[0].Path != "hub.md" || hubs[0].Degree != 4 {
		t.Errorf("top hub = %+v, want hub.md degree 4", hubs[0])
	}
	for _, h := range hubs {
		if h.Path == "orphan.md" {
			t.Errorf("zero-degree note must be excluded from hubs: %+v", h)
		}
	}

	// limit caps the result, keeping the highest-degree first.
	top, err := idx.Hubs("", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 1 || top[0].Path != "hub.md" {
		t.Errorf("Hubs limit=1 = %+v, want [hub.md]", top)
	}
}

func TestIndex_Hubs_ProjectScoped(t *testing.T) {
	idx := openTest(t)
	// Intra-project: phub links a + b (degree 2 within proj).
	upsert(t, idx, "proj/phub.md", "PHub", "[[proj/a]] [[proj/b]]")
	upsert(t, idx, "proj/a.md", "A", "leaf")
	upsert(t, idx, "proj/b.md", "B", "leaf")
	// Cross-project link from another project: excluded when scoped to proj
	// (GraphData includeCross=false), so it must not inflate phub's degree.
	upsert(t, idx, "other/x.md", "X", "[[proj/phub]]")

	hubs, err := idx.Hubs("proj", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hubs) == 0 || hubs[0].Path != "proj/phub.md" || hubs[0].Degree != 2 {
		t.Fatalf("top proj hub = %+v, want proj/phub.md degree 2", hubs)
	}
	for _, h := range hubs {
		if h.Path == "other/x.md" {
			t.Errorf("foreign note leaked into project-scoped hubs: %+v", h)
		}
	}
}

func TestIndex_BFSPath(t *testing.T) {
	idx := openTest(t)
	// Chain a—b—c—d, plus isolated e.
	upsert(t, idx, "a.md", "A", "[[B]]")
	upsert(t, idx, "b.md", "B", "[[C]]")
	upsert(t, idx, "c.md", "C", "[[D]]")
	upsert(t, idx, "d.md", "D", "leaf")
	upsert(t, idx, "e.md", "E", "isolated")

	// shortest path along the chain, inclusive of both endpoints.
	got, err := idx.BFSPath("a.md", "d.md", 6)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.md", "b.md", "c.md", "d.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("path a→d = %v, want %v", got, want)
	}

	// from == to → single-element path.
	got, err = idx.BFSPath("a.md", "a.md", 6)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a.md"}) {
		t.Errorf("path a→a = %v, want [a.md]", got)
	}

	// maxDepth shorter than the real distance → no path (nil), no error.
	got, err = idx.BFSPath("a.md", "d.md", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("path a→d within 2 hops = %v, want nil", got)
	}

	// existing-but-disconnected node → nil, no error.
	got, err = idx.BFSPath("a.md", "e.md", 6)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("path a→e (disconnected) = %v, want nil", got)
	}

	// missing endpoint → typed error.
	if _, err := idx.BFSPath("a.md", "ghost.md", 6); err == nil {
		t.Error("expected error for missing endpoint")
	}
}

func TestIndex_BFSPath_PrefersShortest(t *testing.T) {
	idx := openTest(t)
	// Long way a—b—c—d AND a direct shortcut a—d. BFS must pick the shortcut.
	upsert(t, idx, "a.md", "A", "[[B]] [[D]]")
	upsert(t, idx, "b.md", "B", "[[C]]")
	upsert(t, idx, "c.md", "C", "[[D]]")
	upsert(t, idx, "d.md", "D", "leaf")

	got, err := idx.BFSPath("a.md", "d.md", 6)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.md", "d.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("path a→d = %v, want %v (direct shortcut)", got, want)
	}
}
