package server

import (
	"bufio"
	"net"

	"redis-clone/internal/protocol"
)

type Client struct {
	conn   net.Conn
	writer *bufio.Writer
	server *Server
}

func NewClient(conn net.Conn, server *Server) *Client {
	return &Client{
		conn:   conn,
		writer: bufio.NewWriter(conn),
		server: server,
	}
}

func (c *Client) WriteResponse(resp *protocol.RESPValue) {
	data := protocol.Serialize(resp)
	c.writer.Write(data)
	c.writer.Flush()
}

func (c *Client) WriteError(msg string) {
	resp := &protocol.RESPValue{
		Type: protocol.Error,
		Str:  msg,
	}
	c.WriteResponse(resp)
}

func (c *Client) WriteOK() {
	resp := &protocol.RESPValue{
		Type: protocol.SimpleString,
		Str:  "OK",
	}
	c.WriteResponse(resp)
}
