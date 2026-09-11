package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/iresharma/codeloom.TUI/internal/client"
	"github.com/iresharma/codeloom.TUI/internal/host"
	"github.com/iresharma/codeloom.TUI/internal/tui"
)

func main() {
	workspace := "."
	if len(os.Args) > 1 {
		workspace = os.Args[1]
	}
	workspace, err := filepath.Abs(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workspace: %v\n", err)
		os.Exit(1)
	}
	eng, err := host.AttachOrSpawn(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine: %v\n", err)
		os.Exit(1)
	}
	conn, err := client.Dial(eng.Sock)
	if err != nil {
		eng.Kill()
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	model := tui.New(workspace, eng, conn)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	final, err := p.Run()
	if m, ok := final.(tui.Model); ok {
		m.Close()
	} else {
		_ = conn.Close()
		eng.Kill()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui: %v\n", err)
		os.Exit(1)
	}
}
