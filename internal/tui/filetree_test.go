package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTreeSkipsCachesAndGitignore(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.Mkdir(filepath.Join(root, "src"), 0o755))
	must(os.Mkdir(filepath.Join(root, "node_modules"), 0o755))
	must(os.Mkdir(filepath.Join(root, ".git"), 0o755))
	must(os.WriteFile(filepath.Join(root, "README.md"), []byte("hi"), 0o644))
	must(os.WriteFile(filepath.Join(root, "secret.bin"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(root, ".gitignore"), []byte("secret.bin\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main"), 0o644))

	tree := newFileTree(root)
	names := map[string]bool{}
	for _, n := range tree.flat {
		names[n.name] = true
	}
	if !names["src"] || !names["README.md"] {
		t.Fatalf("missing expected entries: %v", names)
	}
	if names["node_modules"] || names[".git"] || names["secret.bin"] {
		t.Fatalf("should skip caches/gitignore: %v", names)
	}
}

func TestFileTreeTogglePreservesOpen(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "a.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := newFileTree(root)
	found := false
	for _, n := range tree.flat {
		if n.name == "a.go" {
			found = true
		}
	}
	if !found {
		t.Fatal("top-level dirs should start expanded")
	}
	if tree.current() == nil || tree.current().name != "src" {
		t.Fatalf("cursor %#v", tree.current())
	}
	tree.toggle()
	found = false
	for _, n := range tree.flat {
		if n.name == "a.go" {
			found = true
		}
	}
	if found {
		t.Fatal("toggle should collapse")
	}
	tree.toggle()
	tree.reload()
	found = false
	for _, n := range tree.flat {
		if n.name == "a.go" {
			found = true
		}
	}
	if !found {
		t.Fatal("reload should keep directory expanded")
	}
}
