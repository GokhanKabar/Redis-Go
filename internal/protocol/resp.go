package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type RESPType byte

const (
	SimpleString RESPType = '+'
	Error        RESPType = '-'
	Integer      RESPType = ':'
	BulkString   RESPType = '$'
	Array        RESPType = '*'
)

type RESPValue struct {
	Type  RESPType
	Str   string
	Num   int64
	Array []*RESPValue
	Null  bool
}

type RESPParser struct{}

func NewRESPParser() *RESPParser {
	return &RESPParser{}
}

func (p *RESPParser) Parse(input string) (*RESPValue, error) {
	lines := strings.Split(strings.TrimSpace(input), "\r\n")
	if len(lines) == 0 {
		return nil, errors.New("empty input")
	}

	value, _ := p.parseValue(lines, 0)
	return value, nil
}

func (p *RESPParser) parseValue(lines []string, index int) (*RESPValue, int) {
	if index >= len(lines) {
		return nil, index
	}

	line := lines[index]
	if len(line) == 0 {
		return nil, index
	}

	switch RESPType(line[0]) {
	case Array:
		return p.parseArray(lines, index)
	case BulkString:
		return p.parseBulkString(lines, index)
	case SimpleString:
		return &RESPValue{
			Type: SimpleString,
			Str:  line[1:],
		}, index + 1
	case Error:
		return &RESPValue{
			Type: Error,
			Str:  line[1:],
		}, index + 1
	case Integer:
		num, err := strconv.ParseInt(line[1:], 10, 64)
		if err != nil {
			return nil, index
		}
		return &RESPValue{
			Type: Integer,
			Num:  num,
		}, index + 1
	default:
		return nil, index
	}
}

func (p *RESPParser) parseArray(lines []string, index int) (*RESPValue, int) {
	line := lines[index]
	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, index
	}

	array := make([]*RESPValue, count)
	currentIndex := index + 1

	for i := 0; i < count; i++ {
		value, nextIndex := p.parseValue(lines, currentIndex)
		if value == nil {
			return nil, currentIndex
		}
		array[i] = value
		currentIndex = nextIndex
	}

	return &RESPValue{
		Type:  Array,
		Array: array,
	}, currentIndex
}

func (p *RESPParser) parseBulkString(lines []string, index int) (*RESPValue, int) {
	line := lines[index]
	length, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, index
	}

	if length == -1 {
		return &RESPValue{
			Type: BulkString,
			Null: true,
		}, index + 1
	}

	if index+1 >= len(lines) {
		return nil, index
	}

	return &RESPValue{
		Type: BulkString,
		Str:  lines[index+1],
	}, index + 2
}

func Serialize(value *RESPValue) []byte {
	switch value.Type {
	case SimpleString:
		return []byte(fmt.Sprintf("+%s\r\n", value.Str))
	case Error:
		return []byte(fmt.Sprintf("-%s\r\n", value.Str))
	case Integer:
		return []byte(fmt.Sprintf(":%d\r\n", value.Num))
	case BulkString:
		if value.Null {
			return []byte("$-1\r\n")
		}
		return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(value.Str), value.Str))
	case Array:
		result := fmt.Sprintf("*%d\r\n", len(value.Array))
		for _, item := range value.Array {
			result += string(Serialize(item))
		}
		return []byte(result)
	default:
		return []byte("-ERR unknown type\r\n")
	}
}
