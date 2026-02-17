package dashboard

// Cache stores resolved BeadDetail entries keyed by bead ID.
// It is not safe for concurrent use; callers must synchronize externally
// or confine access to a single goroutine (e.g., the Bubble Tea update loop).
type Cache struct {
	entries map[string]*BeadDetail
}

// NewCache creates an empty cache.
func NewCache() *Cache {
	return &Cache{entries: make(map[string]*BeadDetail)}
}

// Get returns the cached detail for the given ID, or nil and false on miss.
func (c *Cache) Get(id string) (*BeadDetail, bool) {
	d, ok := c.entries[id]
	return d, ok
}

// Set stores a detail entry in the cache, replacing any existing entry.
func (c *Cache) Set(id string, detail *BeadDetail) {
	c.entries[id] = detail
}

// Invalidate clears all cached entries.
func (c *Cache) Invalidate() {
	c.entries = make(map[string]*BeadDetail)
}
