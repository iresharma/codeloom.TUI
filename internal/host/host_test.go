package host

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestSocketPath(t *testing.T) {
	got := SocketPath("/tmp/ws")
	if got != filepath.Join("/tmp/ws", ".engine", "engine.sock") {
		t.Fatalf("got %s", got)
	}
}

func TestAttachDoesNotSpawn(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "clt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockDir := filepath.Join(dir, ".engine")
	if err := os.MkdirAll(sockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sock := filepath.Join(sockDir, "engine.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	time.Sleep(50 * time.Millisecond)
	eng, err := AttachOrSpawn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if eng.Owned {
		t.Fatal("expected attach, not spawn")
	}
	if eng.Sock != sock {
		t.Fatalf("sock %s", eng.Sock)
	}
	eng.Kill()
}

func TestSpawnOwnsChild(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "clt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	script := filepath.Join(dir, "fake-engine")
	body := `#!/usr/bin/env python3
import os, socket, sys, time
ws = sys.argv[1]
p = os.path.join(ws, ".engine", "engine.sock")
os.makedirs(os.path.dirname(p), exist_ok=True)
if os.path.exists(p):
    os.unlink(p)
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.bind(p)
s.listen(8)
while True:
    try:
        c, _ = s.accept()
        c.close()
    except Exception:
        time.sleep(0.05)
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(dir, "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODELOOM_ENGINE", script)
	eng, err := AttachOrSpawn(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Kill()
	if !eng.Owned {
		t.Fatal("expected owned spawn")
	}
	if eng.Cmd == nil || eng.Cmd.Process == nil {
		t.Fatal("missing child process")
	}
}

func TestLookupHonorsEnv(t *testing.T) {
	t.Setenv("CODELOOM_ENGINE", "/opt/codeloom/engine")
	bin, args, dir := LookupEngine("/tmp/ws")
	if bin != "/opt/codeloom/engine" {
		t.Fatalf("bin %s", bin)
	}
	if len(args) != 1 || args[0] != "/tmp/ws" {
		t.Fatalf("args %v", args)
	}
	if dir != "/opt/codeloom" {
		t.Fatalf("dir %s", dir)
	}
}

func TestLookupFindsSiblingApp(t *testing.T) {
	t.Setenv("CODELOOM_ENGINE", "")
	_ = os.Unsetenv("CODELOOM_ENGINE")
	if p, err := exec.LookPath("engine"); err == nil {
		t.Skip("engine on PATH:", p)
	}
	bin, args, dir := LookupEngine("/tmp/ws")
	if bin == "" || len(args) < 2 || filepath.Base(args[0]) != "app.py" {
		t.Fatalf("expected python app.py fallback, got bin=%s args=%v dir=%s", bin, args, dir)
	}
	if _, err := os.Stat(args[0]); err != nil {
		t.Fatal(err)
	}
}
