package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"redis-clone/internal/database"
	"redis-clone/internal/persistence"
	"redis-clone/internal/protocol"
)

type Server struct {
	listener    net.Listener
	db          *database.Database
	persistence *persistence.Manager
	clients     map[string]*Client
	clientsMu   sync.RWMutex
	shutdown    chan bool
	config      *Config
}

type Config struct {
	AOFEnabled     bool
	RDBEnabled     bool
	SaveInterval   time.Duration
	AOFSyncPolicy  string
	MaxMemory      int64
	EvictionPolicy string
}

func NewServer(configPath string) *Server {
	config := &Config{
		AOFEnabled:     true,
		RDBEnabled:     true,
		SaveInterval:   300 * time.Second,
		AOFSyncPolicy:  "everysec",
		MaxMemory:      100 * 1024 * 1024, // 100MB
		EvictionPolicy: "allkeys-lru",
	}

	db := database.NewDatabase()
	persistence := persistence.NewManager(db, config.AOFEnabled, config.RDBEnabled)

	return &Server{
		db:          db,
		persistence: persistence,
		clients:     make(map[string]*Client),
		shutdown:    make(chan bool),
		config:      config,
	}
}

func (s *Server) Start(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	s.listener = listener

	// Start background processes
	go s.db.StartExpirationManager()
	go s.persistence.StartBackgroundSave(s.config.SaveInterval)

	// Load existing data
	if err := s.persistence.LoadRDB(); err != nil {
		fmt.Printf("Warning: Could not load RDB file: %v\n", err)
	}

	if err := s.persistence.LoadAOF(); err != nil {
		fmt.Printf("Warning: Could not load AOF file: %v\n", err)
	}

	fmt.Printf("Redis server listening on port %s\n", port)

	for {
		select {
		case <-s.shutdown:
			return nil
		default:
			conn, err := listener.Accept()
			if err != nil {
				// Check if we're shutting down
				select {
				case <-s.shutdown:
					return nil
				default:
					fmt.Printf("Error accepting connection: %v\n", err)
					continue
				}
			}
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("New client connected: %s\n", conn.RemoteAddr())

	client := NewClient(conn, s)
	clientID := conn.RemoteAddr().String()

	s.clientsMu.Lock()
	s.clients[clientID] = client
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientID)
		s.clientsMu.Unlock()
		fmt.Printf("Client disconnected: %s\n", conn.RemoteAddr())
	}()

	reader := bufio.NewReader(conn)

	for {
		// Set read timeout
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Read RESP command
		cmd, err := s.readRESPArray(reader)
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Printf("Error reading command: %v\n", err)
			client.WriteError("ERR " + err.Error())
			continue
		}

		if len(cmd) == 0 {
			client.WriteError("ERR empty command")
			continue
		}

		fmt.Printf("Received command: %v\n", cmd)

		// Convert string array to RESPValue for executeCommand
		respArray := make([]*protocol.RESPValue, len(cmd))
		for i, part := range cmd {
			respArray[i] = &protocol.RESPValue{
				Type: protocol.BulkString,
				Str:  part,
			}
		}

		respCmd := &protocol.RESPValue{
			Type:  protocol.Array,
			Array: respArray,
		}

		response := s.executeCommand(respCmd)
		client.WriteResponse(response)
	}
}

func (s *Server) readRESPArray(reader *bufio.Reader) ([]string, error) {
	// Read the type byte
	typeByte, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	if typeByte != '*' {
		return nil, fmt.Errorf("expected array type '*', got '%c'", typeByte)
	}

	// Read array length
	lengthLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read array length: %w", err)
	}

	lengthStr := strings.TrimSpace(lengthLine)
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array length '%s': %w", lengthStr, err)
	}

	if length <= 0 {
		return []string{}, nil
	}

	// Read each bulk string
	result := make([]string, length)
	for i := 0; i < length; i++ {
		bulkString, err := s.readBulkString(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read bulk string %d: %w", i, err)
		}
		result[i] = bulkString
	}

	return result, nil
}

func (s *Server) readBulkString(reader *bufio.Reader) (string, error) {
	// Read the type byte
	typeByte, err := reader.ReadByte()
	if err != nil {
		return "", err
	}

	if typeByte != '$' {
		return "", fmt.Errorf("expected bulk string type '$', got '%c'", typeByte)
	}

	// Read string length
	lengthLine, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read bulk string length: %w", err)
	}

	lengthStr := strings.TrimSpace(lengthLine)
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", fmt.Errorf("invalid bulk string length '%s': %w", lengthStr, err)
	}

	if length == -1 {
		return "", nil // NULL bulk string
	}

	if length == 0 {
		// Read the trailing \r\n
		_, err = reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read empty bulk string terminator: %w", err)
		}
		return "", nil
	}

	// Read the actual string data
	data := make([]byte, length)
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return "", fmt.Errorf("failed to read bulk string data: %w", err)
	}

	// Read the trailing \r\n
	_, err = reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read bulk string terminator: %w", err)
	}

	return string(data), nil
}

func (s *Server) Shutdown() {
	close(s.shutdown)
	if s.listener != nil {
		s.listener.Close()
	}
	s.persistence.SaveRDB()
	s.persistence.Close()
}
