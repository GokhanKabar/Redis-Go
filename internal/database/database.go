package database

import (
	"sync"
	"time"
)

type Database struct {
	data     map[string]*Value
	expiry   map[string]time.Time
	mu       sync.RWMutex
	shutdown chan bool
}

type ValueType string

const (
	StringType ValueType = "string"
	HashType   ValueType = "hash"
	ListType   ValueType = "list"
	SetType    ValueType = "set"
)

type Value struct {
	Type     ValueType
	StrVal   string
	HashVal  map[string]string
	ListVal  []string
	SetVal   map[string]struct{}
	ExpireAt *time.Time
}

func NewDatabase() *Database {
	return &Database{
		data:     make(map[string]*Value),
		expiry:   make(map[string]time.Time),
		shutdown: make(chan bool),
	}
}

func (db *Database) Set(key, value string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.data[key] = &Value{
		Type:   StringType,
		StrVal: value,
	}
	delete(db.expiry, key)
}

func (db *Database) Get(key string) (string, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.isExpired(key) {
		delete(db.data, key)
		delete(db.expiry, key)
		return "", false
	}

	val, exists := db.data[key]
	if !exists || val.Type != StringType {
		return "", false
	}

	return val.StrVal, true
}

func (db *Database) Del(key string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, exists := db.data[key]
	if exists {
		delete(db.data, key)
		delete(db.expiry, key)
		return true
	}
	return false
}

func (db *Database) Exists(key string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.isExpired(key) {
		delete(db.data, key)
		delete(db.expiry, key)
		return false
	}

	_, exists := db.data[key]
	return exists
}

func (db *Database) Expire(key string, seconds int) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.data[key]; !exists {
		return false
	}

	db.expiry[key] = time.Now().Add(time.Duration(seconds) * time.Second)
	return true
}

func (db *Database) TTL(key string) int64 {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Vérifier si la clé existe
	if _, exists := db.data[key]; !exists {
		return -2 // Key doesn't exist
	}

	// Vérifier si elle a une expiration
	expiry, hasExpiry := db.expiry[key]
	if !hasExpiry {
		return -1 // Key exists but no expiry
	}

	// Calculer le temps restant
	remaining := expiry.Sub(time.Now()).Seconds()
	if remaining <= 0 {
		// La clé a expiré, la supprimer
		delete(db.data, key)
		delete(db.expiry, key)
		return -2 // Key expired
	}

	return int64(remaining)
}

func (db *Database) Keys() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	keys := make([]string, 0, len(db.data))
	for key := range db.data {
		if !db.isExpired(key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (db *Database) isExpired(key string) bool {
	expiry, exists := db.expiry[key]
	if !exists {
		return false
	}
	return time.Now().After(expiry)
}

func (db *Database) StartExpirationManager() {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				db.cleanupExpired()
			case <-db.shutdown:
				return
			}
		}
	}()
}

func (db *Database) cleanupExpired() {
	db.mu.Lock()
	defer db.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, expiry := range db.expiry {
		if now.After(expiry) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(db.data, key)
		delete(db.expiry, key)
	}
}

// Hash operations
func (db *Database) HSet(key, field, value string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	val, exists := db.data[key]
	if !exists {
		val = &Value{
			Type:    HashType,
			HashVal: make(map[string]string),
		}
		db.data[key] = val
	} else if val.Type != HashType {
		val.Type = HashType
		val.HashVal = make(map[string]string)
	}

	val.HashVal[field] = value
}

func (db *Database) HGet(key, field string) (string, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.isExpired(key) {
		return "", false
	}

	val, exists := db.data[key]
	if !exists || val.Type != HashType {
		return "", false
	}

	value, exists := val.HashVal[field]
	return value, exists
}

func (db *Database) HDel(key, field string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	val, exists := db.data[key]
	if !exists || val.Type != HashType {
		return false
	}

	_, exists = val.HashVal[field]
	if exists {
		delete(val.HashVal, field)
		return true
	}
	return false
}
