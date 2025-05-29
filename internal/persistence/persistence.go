package persistence

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"redis-clone/internal/database"
)

type Manager struct {
	db         *database.Database
	aofEnabled bool
	rdbEnabled bool
	aofFile    *os.File
	aofWriter  *bufio.Writer
}

func NewManager(db *database.Database, aofEnabled, rdbEnabled bool) *Manager {
	return &Manager{
		db:         db,
		aofEnabled: aofEnabled,
		rdbEnabled: rdbEnabled,
	}
}

func (m *Manager) StartBackgroundSave(interval time.Duration) {
	if !m.rdbEnabled {
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			m.SaveRDB()
		}
	}()
}

func (m *Manager) SaveRDB() error {
	if !m.rdbEnabled {
		return nil
	}

	file, err := os.Create("dump.rdb")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)

	// Save metadata
	metadata := map[string]interface{}{
		"version":   "1.0",
		"timestamp": time.Now().Unix(),
	}

	if err := encoder.Encode(metadata); err != nil {
		return err
	}

	// Save database data
	keys := m.db.Keys()
	data := make(map[string]interface{})

	for _, key := range keys {
		if val, exists := m.db.Get(key); exists {
			data[key] = val
		}

		// Save TTL information
		if ttl := m.db.TTL(key); ttl > 0 {
			data[key+"__ttl__"] = ttl
		}
	}

	return encoder.Encode(data)
}

func (m *Manager) LoadRDB() error {
	if !m.rdbEnabled {
		return nil
	}

	file, err := os.Open("dump.rdb")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)

	// Load metadata
	var metadata map[string]interface{}
	if err := decoder.Decode(&metadata); err != nil {
		return err
	}

	// Load data
	var data map[string]interface{}
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	for key, value := range data {
		if strVal, ok := value.(string); ok {
			m.db.Set(key, strVal)
		}
	}

	return nil
}

func (m *Manager) WriteAOF(command string) error {
	if !m.aofEnabled {
		return nil
	}

	if m.aofFile == nil {
		file, err := os.OpenFile("appendonly.aof", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		m.aofFile = file
		m.aofWriter = bufio.NewWriter(file)
	}

	_, err := m.aofWriter.WriteString(command + "\n")
	if err != nil {
		return err
	}

	return m.aofWriter.Flush()
}

func (m *Manager) LoadAOF() error {
	if !m.aofEnabled {
		return nil
	}

	file, err := os.Open("appendonly.aof")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := scanner.Text()
		// Here you would replay the command
		// This is a simplified version
		fmt.Printf("Replaying command: %s\n", command)
	}

	return scanner.Err()
}

func (m *Manager) Close() {
	if m.aofWriter != nil {
		m.aofWriter.Flush()
	}
	if m.aofFile != nil {
		m.aofFile.Close()
	}
}
