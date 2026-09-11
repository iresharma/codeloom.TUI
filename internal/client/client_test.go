package client

import (
	"bufio"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/iresharma/codeloom.TUI/internal/protocol"
)

func TestSendAndReceive(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "engine.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	done := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		sc := bufio.NewScanner(c)
		if sc.Scan() {
			done <- sc.Text()
		}
		_, _ = c.Write([]byte(`{"type":"FileContent","path":"a.py","content":"hello"}` + "\n"))
	}()

	conn, err := Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Send(protocol.OpenFile("a.py")); err != nil {
		t.Fatal(err)
	}
	select {
	case line := <-done:
		if line == "" {
			t.Fatal("empty command")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for command")
	}
	evs := conn.Events()
	select {
	case ev := <-evs:
		if ev.Type != "FileContent" || ev.Str("path") != "a.py" {
			t.Fatalf("got %#v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}
