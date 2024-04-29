package largecache

// Stats stores cache statistics
type Stats struct {
	// Hits is a number of successfully found keys
	Hits int64 `json:"hits"`
	// Misses is a number of not found keys
	Misses int64 `json:"messes"`
	// DelHits is number of successfully deleted keys
	DelHits int64 `json:"delete_hits"`
	// DelMissed is a number of not deleted keys
	DelMissed int64 `json:"delete_misses"`
	// Collisions is a number of happend key-collisions
	Collision int64 `json:"collisions"`
}
