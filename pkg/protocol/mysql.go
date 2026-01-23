// Package protocol provides MySQL protocol implementation for admin interface.
package protocol

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

// CommandHandler handles SQL-like commands.
type CommandHandler interface {
	Execute(ctx context.Context, command string) (string, error)
}

// MySQLServer implements a basic MySQL protocol server.
type MySQLServer struct {
	listener net.Listener
	handler  CommandHandler
	wg       sync.WaitGroup
	quit     chan struct{}
}

// NewMySQLServer creates a new MySQL protocol server.
func NewMySQLServer(handler CommandHandler) *MySQLServer {
	return &MySQLServer{
		handler: handler,
		quit:    make(chan struct{}),
	}
}

// Start starts the MySQL server on the given address.
func (s *MySQLServer) Start(address string) error {
	var err error
	s.listener, err = net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop stops the MySQL server.
func (s *MySQLServer) Stop() error {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *MySQLServer) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.quit:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *MySQLServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	client := &mysqlClient{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		handler: s.handler,
		seq:     0,
	}

	if err := client.sendHandshake(); err != nil {
		return
	}

	if err := client.readHandshakeResponse(); err != nil {
		return
	}

	if err := client.sendOK(); err != nil {
		return
	}

	// Command loop
	for {
		select {
		case <-s.quit:
			return
		default:
		}

		if err := client.handleCommand(); err != nil {
			return
		}
	}
}

type mysqlClient struct {
	conn    net.Conn
	reader  *bufio.Reader
	handler CommandHandler
	seq     byte
}

const (
	comQuit  = 0x01
	comQuery = 0x03
	comPing  = 0x0e
)

func (c *mysqlClient) sendHandshake() error {
	// Simple handshake packet
	packet := []byte{
		10,                                        // Protocol version
		'5', '.', '7', '.', '0', '-', 'M', 'i', 'n', 'd', 'B', 'a', 'l', 'a', 'n', 'c', 'e', 'r', 0, // Server version
		1, 0, 0, 0, // Connection ID
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', // Auth plugin data part 1
		0,          // Filler
		0xff, 0xf7, // Capability flags (lower 2 bytes)
		0x21,       // Character set (utf8)
		0x02, 0x00, // Status flags
		0x00, 0x00, // Capability flags (upper 2 bytes)
		0x15, // Length of auth plugin data
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // Reserved
		'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 0, // Auth plugin data part 2
		'm', 'y', 's', 'q', 'l', '_', 'n', 'a', 't', 'i', 'v', 'e', '_', 'p', 'a', 's', 's', 'w', 'o', 'r', 'd', 0, // Auth plugin name
	}

	return c.writePacket(packet)
}

func (c *mysqlClient) readHandshakeResponse() error {
	_, err := c.readPacket()
	return err
}

func (c *mysqlClient) sendOK() error {
	packet := []byte{
		0x00,       // OK header
		0x00,       // Affected rows
		0x00,       // Last insert ID
		0x02, 0x00, // Status flags
		0x00, 0x00, // Warnings
	}
	return c.writePacket(packet)
}

func (c *mysqlClient) sendError(code uint16, message string) error {
	packet := make([]byte, 0, 9+len(message))
	packet = append(packet, 0xff) // Error header
	packet = append(packet, byte(code), byte(code>>8))
	packet = append(packet, '#')
	packet = append(packet, '4', '2', '0', '0', '0') // SQL state
	packet = append(packet, message...)
	return c.writePacket(packet)
}

func (c *mysqlClient) sendResult(text string) error {
	// Simple result set: field count, field, EOF, row, EOF
	lines := strings.Split(strings.TrimSpace(text), "\n")

	// Column count = 1
	if err := c.writePacket([]byte{0x01}); err != nil {
		return err
	}

	// Column definition
	colDef := c.makeColumnDefinition("result", "result")
	if err := c.writePacket(colDef); err != nil {
		return err
	}

	// EOF packet (after columns)
	if err := c.writePacket([]byte{0xfe, 0x00, 0x00, 0x02, 0x00}); err != nil {
		return err
	}

	// Rows
	for _, line := range lines {
		row := c.makeLengthEncodedString(line)
		if err := c.writePacket(row); err != nil {
			return err
		}
	}

	// EOF packet (after rows)
	return c.writePacket([]byte{0xfe, 0x00, 0x00, 0x02, 0x00})
}

func (c *mysqlClient) makeColumnDefinition(name, table string) []byte {
	// Simplified column definition packet
	packet := make([]byte, 0, 128)

	// Catalog
	packet = append(packet, c.makeLengthEncodedString("def")...)
	// Schema
	packet = append(packet, c.makeLengthEncodedString("")...)
	// Table
	packet = append(packet, c.makeLengthEncodedString(table)...)
	// Org table
	packet = append(packet, c.makeLengthEncodedString(table)...)
	// Name
	packet = append(packet, c.makeLengthEncodedString(name)...)
	// Org name
	packet = append(packet, c.makeLengthEncodedString(name)...)
	// Filler
	packet = append(packet, 0x0c)
	// Character set
	packet = append(packet, 0x21, 0x00) // utf8
	// Column length
	packet = append(packet, 0xff, 0xff, 0xff, 0xff)
	// Column type
	packet = append(packet, 0xfd) // VARCHAR
	// Flags
	packet = append(packet, 0x00, 0x00)
	// Decimals
	packet = append(packet, 0x00)
	// Filler
	packet = append(packet, 0x00, 0x00)

	return packet
}

func (c *mysqlClient) makeLengthEncodedString(s string) []byte {
	length := len(s)
	if length < 251 {
		result := make([]byte, 1+length)
		result[0] = byte(length)
		copy(result[1:], s)
		return result
	}
	// For longer strings, use 2-byte length prefix
	result := make([]byte, 3+length)
	result[0] = 0xfc
	result[1] = byte(length)
	result[2] = byte(length >> 8)
	copy(result[3:], s)
	return result
}

func (c *mysqlClient) handleCommand() error {
	data, err := c.readPacket()
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return fmt.Errorf("empty packet")
	}

	cmd := data[0]
	switch cmd {
	case comQuit:
		return fmt.Errorf("client quit")

	case comPing:
		return c.sendOK()

	case comQuery:
		query := string(data[1:])
		return c.handleQuery(query)

	default:
		return c.sendError(1047, "Unknown command")
	}
}

func (c *mysqlClient) handleQuery(query string) error {
	query = strings.TrimSpace(query)

	// Handle some MySQL-specific queries
	upper := strings.ToUpper(query)

	if strings.HasPrefix(upper, "SELECT @@") || strings.HasPrefix(upper, "SET NAMES") ||
		strings.HasPrefix(upper, "SET CHARACTER") || strings.HasPrefix(upper, "SET AUTOCOMMIT") {
		return c.sendOK()
	}

	// Execute through handler
	ctx := context.Background()
	result, err := c.handler.Execute(ctx, query)
	if err != nil {
		return c.sendError(1064, err.Error())
	}

	if result == "" {
		return c.sendOK()
	}

	return c.sendResult(result)
}

func (c *mysqlClient) readPacket() ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return nil, err
	}

	length := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
	c.seq = header[3]

	data := make([]byte, length)
	if _, err := io.ReadFull(c.reader, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (c *mysqlClient) writePacket(data []byte) error {
	c.seq++

	header := make([]byte, 4)
	header[0] = byte(len(data))
	header[1] = byte(len(data) >> 8)
	header[2] = byte(len(data) >> 16)
	header[3] = c.seq

	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(data)
	return err
}

// Helper function for little-endian encoding
func putUint16(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}
