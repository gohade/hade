package contract

import "time"

const CacheKey = "hade:cache"

type Cache interface {
	// Get value by key
	Get(key string) []byte
	// Pull get and delete
	Pull(key string) []byte
	// Check key exists
	Has(key string) bool
	// Set value to key
	Put(key string, val []byte, duration time.Duration) error
	// Forever Put
	Forever(key string, val []byte) error
	// Delete key
	Delete(key string) error
	// increment
	Increment(key string) int
	// decrement
	Decrement(key string) int
}
