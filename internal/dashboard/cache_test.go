package dashboard

import "testing"

func TestCache_GetEmpty(t *testing.T) {
	// Given: an empty cache
	c := NewCache()

	// When: a nonexistent key is retrieved
	got, ok := c.Get("nonexistent")

	// Then: it returns a miss with nil value
	if ok {
		t.Fatal("expected cache miss on empty cache")
	}
	if got != nil {
		t.Fatal("expected nil on cache miss")
	}
}

func TestCache_SetAndGet(t *testing.T) {
	// Given: a cache with one entry stored
	c := NewCache()
	detail := &BeadDetail{ID: "cap-001", Title: "Test bead"}
	c.Set("cap-001", detail)

	// When: the same key is retrieved
	got, ok := c.Get("cap-001")

	// Then: the stored detail is returned
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
	// Given: a cache with two entries
	c := NewCache()
	c.Set("cap-001", &BeadDetail{ID: "cap-001"})
	c.Set("cap-002", &BeadDetail{ID: "cap-002"})

	// When: the cache is invalidated
	c.Invalidate()

	// Then: all entries are removed
	if _, ok := c.Get("cap-001"); ok {
		t.Fatal("expected cache miss after Invalidate")
	}
	if _, ok := c.Get("cap-002"); ok {
		t.Fatal("expected cache miss after Invalidate")
	}
}

func TestCache_OverwriteExisting(t *testing.T) {
	// Given: a cache with the same key set twice
	c := NewCache()
	c.Set("cap-001", &BeadDetail{ID: "cap-001", Title: "v1"})
	c.Set("cap-001", &BeadDetail{ID: "cap-001", Title: "v2"})

	// When: the key is retrieved
	got, ok := c.Get("cap-001")

	// Then: the latest value is returned
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Title != "v2" {
		t.Errorf("got Title %q, want %q", got.Title, "v2")
	}
}
