package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: redis-cli <host:port>")
		fmt.Println("Example: redis-cli localhost:6379")
		os.Exit(1)
	}

	address := os.Args[1]
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Printf("Failed to connect to %s: %v\n", address, err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Connected to Redis server at %s\n", address)
	fmt.Println("Type 'quit' to exit")
	fmt.Println("DEBUG MODE: Showing raw responses")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("redis> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			break
		}

		// Send command
		command := formatCommand(input)
		_, err := conn.Write([]byte(command))
		if err != nil {
			fmt.Printf("Error sending command: %v\n", err)
			continue
		}

		rawData := make([]byte, 1024)
		n, err := conn.Read(rawData)
		if err != nil && err != io.EOF {
			fmt.Printf("Error reading response: %v\n", err)
			continue
		}
		parseResponse(rawData[:n])
	}

	fmt.Println("Goodbye!")
}

func formatCommand(input string) string {
	parts := parseCommandLine(input)
	if len(parts) == 0 {
		return ""
	}

	result := fmt.Sprintf("*%d\r\n", len(parts))
	for _, part := range parts {
		result += fmt.Sprintf("$%d\r\n%s\r\n", len(part), part)
	}
	return result
}

func parseCommandLine(input string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i, char := range input {
		switch {
		case escaped:
			// Handle escaped characters
			current.WriteRune(char)
			escaped = false
		case char == '\\':
			// Next character is escaped
			escaped = true
		case char == '"':
			// Toggle quote state
			inQuotes = !inQuotes
		case char == ' ' && !inQuotes:
			// Space outside quotes - end current part
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			// Skip multiple spaces
			for i+1 < len(input) && input[i+1] == ' ' {
				i++
			}
		default:
			// Regular character
			current.WriteRune(char)
		}
	}

	// Add the last part if there is one
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func parseResponse(data []byte) {
	reader := bufio.NewReader(strings.NewReader(string(data)))
	err := readAndPrintResponse(reader)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		// If it's not valid RESP, just print as plain text
		fmt.Printf("Plain text response: %s\n", string(data))
	}
}

func readAndPrintResponse(reader *bufio.Reader) error {
	// Read the first byte to determine the type
	typeByte, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read type byte: %w", err)
	}

	switch typeByte {
	case '+': // Simple string
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read simple string: %w", err)
		}
		fmt.Printf("OK: %s\n", strings.TrimSuffix(line, "\r\n"))

	case '-': // Error
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read error: %w", err)
		}
		fmt.Printf("(error) %s\n", strings.TrimSuffix(line, "\r\n"))

	case ':': // Integer
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read integer: %w", err)
		}
		fmt.Printf("(integer) %s\n", strings.TrimSuffix(line, "\r\n"))

	case '$': // Bulk string
		// Read the length
		lengthLine, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read bulk string length: %w", err)
		}
		lengthStr := strings.TrimSuffix(lengthLine, "\r\n")
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return fmt.Errorf("invalid bulk string length '%s': %w", lengthStr, err)
		}

		if length == -1 {
			fmt.Println("(nil)")
		} else {
			// Read the actual string data
			data := make([]byte, length)
			_, err = io.ReadFull(reader, data)
			if err != nil {
				return fmt.Errorf("failed to read bulk string data: %w", err)
			}
			// Read the trailing \r\n
			_, err = reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read bulk string terminator: %w", err)
			}
			fmt.Printf("\"%s\"\n", string(data))
		}

	case '*': // Array
		// Read the count
		countLine, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read array count: %w", err)
		}
		countStr := strings.TrimSuffix(countLine, "\r\n")
		count, err := strconv.Atoi(countStr)
		if err != nil {
			return fmt.Errorf("invalid array count '%s': %w", countStr, err)
		}

		if count == 0 {
			fmt.Println("(empty array)")
		} else if count == -1 {
			fmt.Println("(nil)")
		} else {
			fmt.Printf("Array with %d elements:\n", count)
			// Read each element
			for i := 0; i < count; i++ {
				fmt.Printf("  [%d] ", i)
				err = readAndPrintResponse(reader)
				if err != nil {
					return fmt.Errorf("failed to read array element %d: %w", i, err)
				}
			}
		}

	default:
		return fmt.Errorf("unknown RESP type: %c (0x%02x)", typeByte, typeByte)
	}

	return nil
}
