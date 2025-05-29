package server

import (
	"strconv"
	"strings"

	"redis-clone/internal/protocol"
)

func (s *Server) executeCommand(cmd *protocol.RESPValue) *protocol.RESPValue {
	if cmd.Type != protocol.Array || len(cmd.Array) == 0 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR invalid command format",
		}
	}

	command := strings.ToUpper(cmd.Array[0].Str)
	args := make([]string, len(cmd.Array)-1)
	for i, arg := range cmd.Array[1:] {
		args[i] = arg.Str
	}

	// Log command for AOF
	if isWriteCommand(command) {
		cmdStr := command
		for _, arg := range args {
			cmdStr += " " + arg
		}
		s.persistence.WriteAOF(cmdStr)
	}

	switch command {
	case "PING":
		return s.handlePing(args)
	case "SET":
		return s.handleSet(args)
	case "GET":
		return s.handleGet(args)
	case "DEL":
		return s.handleDel(args)
	case "EXISTS":
		return s.handleExists(args)
	case "EXPIRE":
		return s.handleExpire(args)
	case "TTL":
		return s.handleTTL(args)
	case "KEYS":
		return s.handleKeys(args)
	case "HSET":
		return s.handleHSet(args)
	case "HGET":
		return s.handleHGet(args)
	case "HDEL":
		return s.handleHDel(args)
	case "INCR":
		return s.handleIncr(args)
	case "DECR":
		return s.handleDecr(args)
	default:
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR unknown command '" + command + "'",
		}
	}
}

func isWriteCommand(command string) bool {
	writeCommands := map[string]bool{
		"SET":    true,
		"DEL":    true,
		"EXPIRE": true,
		"HSET":   true,
		"HDEL":   true,
		"INCR":   true,
		"DECR":   true,
	}
	return writeCommands[command]
}

func (s *Server) handlePing(args []string) *protocol.RESPValue {
	if len(args) == 0 {
		return &protocol.RESPValue{
			Type: protocol.SimpleString,
			Str:  "PONG",
		}
	}
	return &protocol.RESPValue{
		Type: protocol.BulkString,
		Str:  args[0],
	}
}

func (s *Server) handleSet(args []string) *protocol.RESPValue {
	if len(args) < 2 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'set' command",
		}
	}

	key, value := args[0], args[1]
	s.db.Set(key, value)

	return &protocol.RESPValue{
		Type: protocol.SimpleString,
		Str:  "OK",
	}
}

func (s *Server) handleGet(args []string) *protocol.RESPValue {
	if len(args) != 1 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'get' command",
		}
	}

	key := args[0]
	value, exists := s.db.Get(key)
	if !exists {
		return &protocol.RESPValue{
			Type: protocol.BulkString,
			Null: true,
		}
	}

	return &protocol.RESPValue{
		Type: protocol.BulkString,
		Str:  value,
	}
}

func (s *Server) handleDel(args []string) *protocol.RESPValue {
	if len(args) == 0 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'del' command",
		}
	}

	deleted := 0
	for _, key := range args {
		if s.db.Del(key) {
			deleted++
		}
	}

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  int64(deleted),
	}
}

func (s *Server) handleExists(args []string) *protocol.RESPValue {
	if len(args) == 0 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'exists' command",
		}
	}

	count := 0
	for _, key := range args {
		if s.db.Exists(key) {
			count++
		}
	}

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  int64(count),
	}
}

func (s *Server) handleExpire(args []string) *protocol.RESPValue {
	if len(args) != 2 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'expire' command",
		}
	}

	key := args[0]
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR value is not an integer or out of range",
		}
	}

	if s.db.Expire(key, seconds) {
		return &protocol.RESPValue{
			Type: protocol.Integer,
			Num:  1,
		}
	}

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  0,
	}
}

func (s *Server) handleTTL(args []string) *protocol.RESPValue {
	if len(args) != 1 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'ttl' command",
		}
	}

	key := args[0]
	ttl := s.db.TTL(key)

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  ttl,
	}
}

func (s *Server) handleKeys(args []string) *protocol.RESPValue {
	keys := s.db.Keys()
	result := make([]*protocol.RESPValue, len(keys))

	for i, key := range keys {
		result[i] = &protocol.RESPValue{
			Type: protocol.BulkString,
			Str:  key,
		}
	}

	return &protocol.RESPValue{
		Type:  protocol.Array,
		Array: result,
	}
}

func (s *Server) handleHSet(args []string) *protocol.RESPValue {
	if len(args) != 3 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'hset' command",
		}
	}

	key, field, value := args[0], args[1], args[2]
	s.db.HSet(key, field, value)

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  1,
	}
}

func (s *Server) handleHGet(args []string) *protocol.RESPValue {
	if len(args) != 2 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'hget' command",
		}
	}

	key, field := args[0], args[1]
	value, exists := s.db.HGet(key, field)
	if !exists {
		return &protocol.RESPValue{
			Type: protocol.BulkString,
			Null: true,
		}
	}

	return &protocol.RESPValue{
		Type: protocol.BulkString,
		Str:  value,
	}
}

func (s *Server) handleHDel(args []string) *protocol.RESPValue {
	if len(args) < 2 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'hdel' command",
		}
	}

	key := args[0]
	deleted := 0
	for _, field := range args[1:] {
		if s.db.HDel(key, field) {
			deleted++
		}
	}

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  int64(deleted),
	}
}

func (s *Server) handleIncr(args []string) *protocol.RESPValue {
	if len(args) != 1 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'incr' command",
		}
	}

	key := args[0]
	value, exists := s.db.Get(key)
	var intValue int64 = 0

	if exists {
		num, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return &protocol.RESPValue{
				Type: protocol.Error,
				Str:  "ERR value is not an integer or out of range",
			}
		}
		intValue = num
	}

	intValue++
	s.db.Set(key, strconv.FormatInt(intValue, 10))

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  intValue,
	}
}

func (s *Server) handleDecr(args []string) *protocol.RESPValue {
	if len(args) != 1 {
		return &protocol.RESPValue{
			Type: protocol.Error,
			Str:  "ERR wrong number of arguments for 'decr' command",
		}
	}

	key := args[0]
	value, exists := s.db.Get(key)
	var intValue int64 = 0

	if exists {
		num, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return &protocol.RESPValue{
				Type: protocol.Error,
				Str:  "ERR value is not an integer or out of range",
			}
		}
		intValue = num
	}

	intValue--
	s.db.Set(key, strconv.FormatInt(intValue, 10))

	return &protocol.RESPValue{
		Type: protocol.Integer,
		Num:  intValue,
	}
}
