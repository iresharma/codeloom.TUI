package host

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type Engine struct {
	Cmd       *exec.Cmd
	Owned     bool
	Sock      string
	Workspace string
}

func SocketPath(workspace string) string {
	return filepath.Join(workspace, ".engine", "engine.sock")
}

func AttachOrSpawn(workspace string) (*Engine, error) {
	ws, err := filepath.Abs(workspace)
	if err != nil {
		return nil, err
	}
	sock := SocketPath(ws)
	if canDial(sock) {
		return &Engine{Sock: sock, Workspace: ws, Owned: false}, nil
	}
	bin, args, dir := LookupEngine(ws)
	if bin == "" {
		return nil, fmt.Errorf("no engine binary or python app.py found; start the engine or set CODELOOM_ENGINE")
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("spawn engine: %w", err)
	}
	eng := &Engine{Cmd: cmd, Owned: true, Sock: sock, Workspace: ws}
	if err := waitSock(sock, 8*time.Second); err != nil {
		eng.Kill()
		return nil, err
	}
	return eng, nil
}

func LookupEngine(workspace string) (bin string, args []string, dir string) {
	if env := os.Getenv("CODELOOM_ENGINE"); env != "" {
		return env, []string{workspace}, filepath.Dir(env)
	}
	if p, err := exec.LookPath("engine"); err == nil {
		return p, []string{workspace}, workspace
	}
	var starts []string
	if wd, err := os.Getwd(); err == nil {
		starts = append(starts, wd)
	}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	starts = append(starts, workspace, filepath.Dir(workspace))
	seen := map[string]struct{}{}
	for _, start := range starts {
		cur := start
		for i := 0; i < 8; i++ {
			if _, ok := seen[cur]; ok {
				break
			}
			seen[cur] = struct{}{}
			for _, rel := range []string{
				filepath.Join("engine", "engine"),
				filepath.Join("workspace", "engine", "engine"),
			} {
				c := filepath.Join(cur, rel)
				if st, err := os.Stat(c); err == nil && !st.IsDir() {
					return c, []string{workspace}, filepath.Dir(c)
				}
			}
			for _, rel := range []string{
				filepath.Join("engine", "app.py"),
				filepath.Join("workspace", "engine", "app.py"),
			} {
				app := filepath.Join(cur, rel)
				if _, err := os.Stat(app); err == nil {
					py, err := exec.LookPath("python3")
					if err != nil {
						py = "python3"
					}
					return py, []string{app, workspace}, filepath.Dir(app)
				}
			}
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}
	return "", nil, ""
}

func canDial(sock string) bool {
	c, err := net.DialTimeout("unix", sock, 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func waitSock(sock string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if canDial(sock) {
			return nil
		}
		time.Sleep(80 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", sock)
}

func (e *Engine) Kill() {
	if e == nil || !e.Owned || e.Cmd == nil || e.Cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(e.Cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = e.Cmd.Process.Signal(syscall.SIGTERM)
	}
	done := make(chan struct{})
	go func() {
		_, _ = e.Cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		if err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = e.Cmd.Process.Kill()
		}
	}
}
