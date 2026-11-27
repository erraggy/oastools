package mcpserver

// reset clears all cached entries.
func (c *specCacheStore) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// size returns the number of cached entries.
func (c *specCacheStore) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
