// Package protocol provides MySQL protocol implementation for admin interface.
package protocol

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"io"
	"log"
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
	username string
	password string // plaintext credential; empty => loopback-only access
	wg       sync.WaitGroup
	quit     chan struct{}
}

// NewMySQLServer creates a new MySQL protocol server. The username/password are
// verified against the client's mysql_native_password handshake response. If
// password is empty, connections are accepted only from loopback (fail-closed
// for anything remote).
func NewMySQLServer(handler CommandHandler, username, password string) *MySQLServer {
	return &MySQLServer{
		handler:  handler,
		username: username,
		password: password,
		quit:     make(chan struct{}),
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

// authorize decides whether a connection may proceed. When a password is
// configured, the client's mysql_native_password response is verified against
// it. When no password is configured, only loopback connections are allowed.
func (s *MySQLServer) authorize(conn net.Conn, username string, authResp, scramble []byte) bool {
	if s.username != "" && username != s.username {
		return false
	}

	if s.password == "" {
		// No verifiable credential: restrict to loopback and warn.
		if !isLoopbackConn(conn) {
			log.Printf("admin(mysql): rejected non-loopback connection from %s (no admin_password configured)", conn.RemoteAddr())
			return false
		}
		return true
	}

	return verifyNativePassword(s.password, scramble, authResp)
}

func isLoopbackConn(conn net.Conn) bool {
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// verifyNativePassword verifies a mysql_native_password auth response:
//
//	response = SHA1(password) XOR SHA1(scramble + SHA1(SHA1(password)))
//
// The server recovers SHA1(password) and checks SHA1 of it equals the stored
// double hash, in constant time.
func verifyNativePassword(password string, scramble, authResp []byte) bool {
	if len(authResp) == 0 {
		return password == ""
	}
	if len(authResp) != sha1.Size || len(scramble) == 0 {
		return false
	}

	stage1 := sha1.Sum([]byte(password)) // SHA1(password)
	stage2 := sha1.Sum(stage1[:])        // SHA1(SHA1(password))

	h := sha1.New()
	h.Write(scramble)
	h.Write(stage2[:])
	token := h.Sum(nil) // SHA1(scramble + stage2)

	recovered := make([]byte, sha1.Size)
	for i := range recovered {
		recovered[i] = authResp[i] ^ token[i]
	}
	check := sha1.Sum(recovered) // should equal stage2
	return subtle.ConstantTimeCompare(check[:], stage2[:]) == 1
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

	scramble := make([]byte, 20)
	if _, err := rand.Read(scramble); err != nil {
		return
	}

	client := &mysqlClient{
		conn:     conn,
		reader:   bufio.NewReader(conn),
		handler:  s.handler,
		seq:      0,
		scramble: scramble,
	}

	if err := client.sendHandshake(); err != nil {
		return
	}

	username, authResp, err := client.readHandshakeResponse()
	if err != nil {
		return
	}

	if !s.authorize(conn, username, authResp, client.scramble) {
		_ = client.sendError(1045, "Access denied for user")
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
	conn     net.Conn
	reader   *bufio.Reader
	handler  CommandHandler
	seq      byte
	seqSet   bool   // track if we've read a packet to know the sequence
	scramble []byte // 20-byte auth challenge sent in the handshake
}

const (
	comQuit  = 0x01
	comQuery = 0x03
	comPing  = 0x0e
)

func (c *mysqlClient) sendHandshake() error {
	// Handshake v10. The 20-byte scramble is split into an 8-byte part 1 and a
	// 12-byte part 2 (per the protocol), and the client re-joins them.
	part1 := c.scramble[:8]
	part2 := c.scramble[8:20]

	packet := make([]byte, 0, 128)
	packet = append(packet, 10) // Protocol version
	packet = append(packet, []byte("5.7.0-MindBalancer")...)
	packet = append(packet, 0)                   // NUL-terminated server version
	packet = append(packet, 1, 0, 0, 0)          // Connection ID
	packet = append(packet, part1...)            // Auth plugin data part 1 (8 bytes)
	packet = append(packet, 0)                   // Filler
	packet = append(packet, 0xff, 0xf7)          // Capability flags (lower 2 bytes)
	packet = append(packet, 0x21)                // Character set (utf8)
	packet = append(packet, 0x02, 0x00)          // Status flags
	packet = append(packet, 0x00, 0x00)          // Capability flags (upper 2 bytes)
	packet = append(packet, 0x15)                // Length of auth plugin data (21)
	packet = append(packet, make([]byte, 10)...) // Reserved
	packet = append(packet, part2...)            // Auth plugin data part 2 (12 bytes)
	packet = append(packet, 0)                   // NUL terminator for part 2
	packet = append(packet, []byte("mysql_native_password")...)
	packet = append(packet, 0) // NUL-terminated auth plugin name

	return c.writePacket(packet)
}

// readHandshakeResponse reads the client's handshake response and returns the
// username and the mysql_native_password auth response (scrambled password).
func (c *mysqlClient) readHandshakeResponse() (string, []byte, error) {
	data, err := c.readPacket()
	if err != nil {
		return "", nil, err
	}
	return parseHandshakeResponse(data)
}

// parseHandshakeResponse parses a protocol-41 HandshakeResponse packet.
// Layout: 4 bytes capabilities, 4 bytes max packet, 1 byte charset, 23 bytes
// filler, NUL-terminated username, then a 1-byte-length-prefixed auth response
// (CLIENT_SECURE_CONNECTION form, which go-sql-driver uses here).
func parseHandshakeResponse(data []byte) (string, []byte, error) {
	const headerLen = 4 + 4 + 1 + 23
	if len(data) < headerLen {
		return "", nil, fmt.Errorf("handshake response too short")
	}
	pos := headerLen

	nul := bytes.IndexByte(data[pos:], 0)
	if nul < 0 {
		return "", nil, fmt.Errorf("missing username terminator")
	}
	username := string(data[pos : pos+nul])
	pos += nul + 1

	var authResp []byte
	if pos < len(data) {
		l := int(data[pos])
		pos++
		if pos+l > len(data) {
			return "", nil, fmt.Errorf("truncated auth response")
		}
		authResp = data[pos : pos+l]
	}
	return username, authResp, nil
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
	// Reset sequence for each new command
	c.seq = 0

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
	c.seq = header[3] + 1 // Next packet we send should be seq+1

	data := make([]byte, length)
	if _, err := io.ReadFull(c.reader, data); err != nil {
		return nil, err
	}

	return data, nil
}

func (c *mysqlClient) writePacket(data []byte) error {
	header := make([]byte, 4)
	header[0] = byte(len(data))
	header[1] = byte(len(data) >> 8)
	header[2] = byte(len(data) >> 16)
	header[3] = c.seq

	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	if _, err := c.conn.Write(data); err != nil {
		return err
	}

	c.seq++
	return nil
}

// Helper function for little-endian encoding
func putUint16(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}
