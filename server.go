// Package memcached provides an interface for building your
// own memcached ascii protocol servers.
package memcached

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
)

const VERSION = "0.0.0"

var (
	crlf    = []byte("\r\n")
	noreply = []byte("noreply")
)

type conn struct {
	server *Server
	conn   net.Conn
	rwc    *bufio.ReadWriter
	ctx    *context.Context
	cancel context.CancelFunc
}

type Server struct {
	Addr    string
	Handler RequestHandler
	Stats   Stats
}

type StorageCmd struct {
	Key     string
	Flags   int
	Exptime int64
	Length  int
	Noreply bool
}

func (s *Server) newConn(rwc net.Conn) (c *conn, err error) {
	c = new(conn)
	c.server = s
	c.conn = rwc
	c.rwc = bufio.NewReadWriter(bufio.NewReaderSize(rwc, 1048576), bufio.NewWriter(rwc))
	return c, nil
}

// Start listening and accepting requests to this server.
func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":11211"
	}
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return e
	}
	return s.Serve(l)
}

func (s *Server) Serve(l net.Listener) error {
	serverCtx, cancel := context.WithCancel(context.Background())
	defer l.Close()
	defer cancel()
	//push stats into the handler
	handler, _ := s.Handler.(StatsHandler)
	handler.Stats(s.Stats)

	for {
		rw, e := l.Accept()
		if e != nil {
			return e
		}
		c, err := s.newConn(rw)
		if err != nil {
			continue
		}
		go c.serve(&serverCtx)
	}
}

func (c *conn) serve(serverCtx *context.Context) {
	defer func() {
		c.server.Stats["curr_connections"].(*CounterStat).Decrement(1)
		c.Close()
	}()

	ctx, cancel := context.WithCancel(*serverCtx)
	defer cancel()

	c.server.Stats["total_connections"].(*CounterStat).Increment(1)
	c.server.Stats["curr_connections"].(*CounterStat).Increment(1)

	for {
		err := c.handleRequest(&ctx)
		if err != nil {
			if err == io.EOF {
				return
			}
			c.rwc.WriteString(err.Error())
			c.end()
		}
	}
}

func (c *conn) end() {
	c.rwc.Flush()
}

func (c *conn) handleRequest(ctx *context.Context) error {
    
    var byt int 
	line, err := c.ReadLine()
	if err != nil || len(line) == 0 {
		return io.EOF
	}
	if len(line) < 4 {
		return Error
	}
	switch line[0] {
	case 'g':
		key := string(line[4:]) // get
		getter, ok := c.server.Handler.(Getter)
		if !ok {
			return Error
		}
		c.server.Stats["cmd_get"].(*CounterStat).Increment(1)
		//response := getter.Get(key)
		response := getter.GetWithContext(ctx, key)
		if response != nil {
			c.server.Stats["get_hits"].(*CounterStat).Increment(1)
			byt = response.WriteResponse(c.rwc)
		} else {
			c.server.Stats["get_misses"].(*CounterStat).Increment(1)
		}
		byt,_ := c.rwc.WriteString(StatusEnd)
		c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
		c.end()
	case 's':
		switch line[1] {
		case 'e':
			if len(line) < 11 {
				return Error
			}
			setter, ok := c.server.Handler.(Setter)
			if !ok {
				return Error
			}
			item := &Item{}
			cmd := parseStorageLine(line)
			item.Key = cmd.Key
			item.Flags = cmd.Flags
			item.SetExpires(cmd.Exptime)

			value := make([]byte, cmd.Length+2)
			n, err := c.Read(value)
			
			if err != nil {
				return Error
			}

			// Didn't provide the correct number of bytes
			if n != cmd.Length+2 {
				response := &ClientErrorResponse{"bad chunk data"}
				byt = response.WriteResponse(c.rwc)
				c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
				c.ReadLine() // Read out the rest of the line
				return Error
			}

			// Doesn't end with \r\n
			if !bytes.HasSuffix(value, crlf) {
				response := &ClientErrorResponse{"bad chunk data"}
				byt = response.WriteResponse(c.rwc)
				c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
				c.ReadLine() // Read out the rest of the line
				return Error
			}

			// Copy the value into the *Item
			item.Value = make([]byte, len(value)-2)
			copy(item.Value, value)

			c.server.Stats["cmd_set"].(*CounterStat).Increment(1)
			if cmd.Noreply {
				go setter.Set(item)
			} else {
				//response := setter.Set(item)
				response := setter.SetWithContext(ctx, item)
				if response != nil {
					byt = response.WriteResponse(c.rwc)
					c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
					c.end()
				} else {
					byt,_ = c.rwc.WriteString(StatusStored)
					c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
					c.end()
				}
			}
		case 't':
			if len(line) != 5 {
				return Error
			}
			for key, value := range c.server.Stats {
				fmt.Fprintf(c.rwc, StatusStat, key, value)
			}
			byt,_ = c.rwc.WriteString(StatusEnd)
			c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
			c.end()
		default:
			return Error
		}
	case 'd':
		if len(line) < 8 {
			return Error
		}
		key := string(line[7:]) // delete
		deleter, ok := c.server.Handler.(Deleter)
		if !ok {
			return Error
		}
		//err := deleter.Delete(key)
		err := deleter.DeleteWithContext(ctx, key)
		if err != nil {
			byt,_ = c.rwc.WriteString(StatusNotFound)
			c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
			c.end()
		} else {
			byt,_ = c.rwc.WriteString(StatusDeleted)
			c.server.Stats["bytes_written"].(*CounterStat).Increment(byt)
			c.end()
		}
	case 'q':
		if len(line) == 4 {
			return io.EOF
		}
		return Error
	default:
		return Error
	}
	return nil
}

func (c *conn) Close() {
	c.conn.Close()
}

func (c *conn) ReadLine() (line []byte, err error) {
	
	line, _, err = c.rwc.ReadLine()
	
	c.server.Stats["bytes_read"].(*CounterStat).Increment(len(line))
	
	return
}

func (c *conn) Read(p []byte) (n int, err error) {
	n,err = io.ReadFull(c.rwc, p)
	
	c.server.Stats["bytes_read"].(*CounterStat).Increment(n)
	
	return n,err
}

func ListenAndServe(addr string) error {
	s := &Server{
		Addr: addr,
	}
	return s.ListenAndServe()
}

func parseStorageLine(line []byte) *StorageCmd {
	pieces := bytes.Fields(line[4:]) // Skip the actual "set "
	cmd := &StorageCmd{}
	// lol, no error handling here
	cmd.Key = string(pieces[0])
	cmd.Flags, _ = strconv.Atoi(string(pieces[1]))
	cmd.Exptime, _ = strconv.ParseInt(string(pieces[2]), 10, 64)
	cmd.Length, _ = strconv.Atoi(string(pieces[3]))
	cmd.Noreply = len(pieces) == 5 && bytes.Equal(pieces[4], noreply)
	return cmd
}

// Initialize a new memcached Server
func NewServer(listen string, handler RequestHandler) *Server {
	return &Server{listen, handler, NewStats()}
}
