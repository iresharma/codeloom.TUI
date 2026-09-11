package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

type Conn struct {
	nc     net.Conn
	writer *bufio.Writer
	mu     sync.Mutex
}

func Dial(socketPath string) (*Conn, error) {
	nc, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &Conn{nc: nc, writer: bufio.NewWriter(nc)}, nil
}

func (c *Conn) Send(cmd protocol.Command) error {
	raw, err := protocol.Encode(cmd)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.writer.Write(raw); err != nil {
		return err
	}
	return c.writer.Flush()
}

// Events starts a single reader goroutine. Call it once per connection.
func (c *Conn) Events() <-chan protocol.Event {
	out := make(chan protocol.Event, 256)
	go func() {
		defer close(out)
		sc := protocol.NewScanner(c.nc)
		for sc.Scan() {
			ev, err := protocol.DecodeEvent(sc.Bytes())
			if err != nil {
				out <- protocol.Event{
					Type: "ErrorOccurred",
					Raw: map[string]any{
						"type":    "ErrorOccurred",
						"message": err.Error(),
					},
				}
				continue
			}
			out <- ev
		}
		if err := sc.Err(); err != nil {
			out <- protocol.Event{
				Type: "ErrorOccurred",
				Raw: map[string]any{
					"type":    "ErrorOccurred",
					"message": err.Error(),
				},
			}
		}
	}()
	return out
}

func (c *Conn) Close() error {
	if c == nil || c.nc == nil {
		return nil
	}
	return c.nc.Close()
}

func (c *Conn) String() string {
	if c == nil || c.nc == nil {
		return "disconnected"
	}
	return fmt.Sprintf("unix:%s", c.nc.RemoteAddr())
}
