package events

import "testing"

func TestHub_ReplaySince(t *testing.T) {
	h := New(HubOptions{RingLen: 4})
	for i := 0; i < 5; i++ {
		h.Publish(TopicNote, map[string]any{"path": "p/a.md"})
	}
	h.Publish(TopicTree, map[string]any{"path": "p/a.md"})
	if h.Seq() != 6 {
		t.Fatalf("Seq = %d, want 6", h.Seq())
	}

	// Cursor inside the retention window: exact replay, no resync.
	evs, resync := h.ReplaySince(4)
	if resync || len(evs) != 2 || evs[0].Seq != 5 || evs[1].Seq != 6 {
		t.Fatalf("ReplaySince(4) = %+v resync=%v", evs, resync)
	}

	// Topic filter.
	evs, _ = h.ReplaySince(4, TopicTree)
	if len(evs) != 1 || evs[0].Topic != TopicTree {
		t.Fatalf("topic-filtered replay = %+v", evs)
	}

	// Cursor older than the ring (oldest retained is seq 3): resync.
	if _, resync = h.ReplaySince(1); !resync {
		t.Fatal("cursor 1 must trigger resync (ring starts at 3)")
	}

	// Cursor from a previous hub incarnation (ahead of us): resync.
	if _, resync = h.ReplaySince(99); !resync {
		t.Fatal("cursor ahead of seq must trigger resync")
	}

	// Cursor exactly at head: nothing, no resync.
	if evs, resync = h.ReplaySince(6); resync || len(evs) != 0 {
		t.Fatalf("ReplaySince(head) = %+v resync=%v", evs, resync)
	}
}
