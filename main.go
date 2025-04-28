package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

// Structure pour stocker les données en mémoire
type RedisStore struct {
	data map[string]string
	mu   sync.RWMutex
}

// Créer une nouvelle instance du store
func NewRedisStore() *RedisStore {
	return &RedisStore{
		data: make(map[string]string),
	}
}

// Gérer une connexion client
func handleConnection(conn net.Conn, store *RedisStore) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		// Pour cette première étape, on utilise un protocole simple:
		// chaque commande est une ligne de texte
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Erreur de lecture: %v", err)
			return
		}

		// Traiter la commande
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)

		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])
		args := parts[1:]

		// Exécuter la commande et obtenir la réponse
		response := executeCommand(cmd, args, store)

		// Envoyer la réponse au client
		_, err = conn.Write([]byte(response + "\n"))
		if err != nil {
			log.Printf("Erreur d'écriture: %v", err)
			return
		}
	}
}

// Exécuter une commande Redis
func executeCommand(cmd string, args []string, store *RedisStore) string {
	switch cmd {
	case "PING":
		return "PONG"

	case "SET":
		if len(args) < 2 {
			return "ERR wrong number of arguments for SET command"
		}

		key, value := args[0], args[1]

		store.mu.Lock()
		store.data[key] = value
		store.mu.Unlock()

		return "OK"

	case "GET":
		if len(args) != 1 {
			return "ERR wrong number of arguments for GET command"
		}

		key := args[0]

		store.mu.RLock()
		defer store.mu.RUnlock()

		if value, exists := store.data[key]; exists {
			return value
		}

		return "(nil)"

	case "DEL":
		if len(args) < 1 {
			return "ERR wrong number of arguments for DEL command"
		}

		count := 0
		store.mu.Lock()
		for _, key := range args {
			if _, exists := store.data[key]; exists {
				delete(store.data, key)
				count++
			}
		}
		store.mu.Unlock()

		return fmt.Sprintf("%d", count)

	case "EXISTS":
		if len(args) < 1 {
			return "ERR wrong number of arguments for EXISTS command"
		}

		count := 0
		store.mu.RLock()
		for _, key := range args {
			if _, exists := store.data[key]; exists {
				count++
			}
		}
		store.mu.RUnlock()

		return fmt.Sprintf("%d", count)

	default:
		return fmt.Sprintf("ERR unknown command '%s'", cmd)
	}
}

func main() {
	// Créer l'instance de stockage
	store := NewRedisStore()

	// Démarrer le serveur sur le port standard Redis
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatalf("Impossible de démarrer le serveur: %v", err)
	}
	defer listener.Close()

	log.Println("Serveur Redis en Go démarré sur le port :6379")

	// Accepter et gérer les connexions
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Erreur lors de l'acceptation de la connexion: %v", err)
			continue
		}

		// Gérer chaque client dans une goroutine séparée
		go handleConnection(conn, store)
	}
}
