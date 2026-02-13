package dashboard

import "testing"

func TestCache_GetEmpty(t *testing.T) {
	c := NewCache()
	got, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected cache miss on empty cache")
	}
	if got != nil {
		t.Fatal("expected nil on cache miss")
	}
}

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache()
	detail := &BeadDetail{ID: "cap-001", Title: "Test bead"}

	c.Set("cap-001", detail)

	got, ok := c.Get("cap-001")
	if !ok {
		t.Fatal("expected cache hit after Set")
	}
	if got.ID != "cap-001" {
		t.Errorf("got ID %q, want %q", got.ID, "cap-001")
	}
	if got.Title != "Test bead" {
		t.Errorf("got Title %q, want %q", got.Title, "Test bead")
	}
}

func TestCache_Invalidate(t *testing.T) {
	c := NewCache()
	c.Set("cap-001", &BeadDetail{ID: "cap-001"})
	c.Set("cap-002", &BeadDetail{ID: "cap-002"})

	c.Invalidate()

	if _, ok := c.Get("cap-001"); ok {
		t.Fatal("expected cache miss after Invalidate")
	}
	if _, ok := c.Get("cap-002"); ok {
		t.Fatal("expected cache miss after Invalidate")
	}
}

func TestCache_OverwriteExisting(t *testing.T) {
	c := NewCache()
	c.Set("cap-001", &BeadDetail{ID: "cap-001", Title: "v1"})
	c.Set("cap-001", &BeadDetail{ID: "cap-001", Title: "v2"})

	got, ok := c.Get("cap-001")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Title != "v2" {
		t.Errorf("got Title %q, want %q", got.Title, "v2")
	}
}
