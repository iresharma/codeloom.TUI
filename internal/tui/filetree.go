package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var skipNames = map[string]struct{}{
	".git": {}, ".engine": {}, ".cursor": {}, "__pycache__": {},
	"node_modules": {}, ".venv": {}, "venv": {}, ".ruff_cache": {},
	"dist": {}, "build": {}, ".next": {}, ".turbo": {},
}

type treeNode struct {
	name     string
	path     string
	isDir    bool
	open     bool
	children []*treeNode
}

type fileTree struct {
	root      string
	nodes     []*treeNode
	flat      []*treeNode
	cursor    int
	gitignore map[string]struct{}
}

func newFileTree(root string) *fileTree {
	t := &fileTree{root: root, gitignore: loadGitignore(root)}
	t.reload()
	t.expandTop()
	return t
}

func (t *fileTree) expandTop() {
	for _, n := range t.nodes {
		if n.isDir && !n.open {
			n.open = true
			n.children = t.readDir(n.path)
		}
	}
	t.flatten()
}

func loadGitignore(root string) map[string]struct{} {
	out := map[string]struct{}{}
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "/")
		line = strings.TrimSuffix(line, "/")
		out[line] = struct{}{}
	}
	return out
}

func (t *fileTree) reload() {
	open := map[string]bool{}
	var collect func([]*treeNode)
	collect = func(nodes []*treeNode) {
		for _, n := range nodes {
			if n.open {
				open[n.path] = true
			}
			collect(n.children)
		}
	}
	collect(t.nodes)
	t.nodes = t.readDir("")
	var restore func([]*treeNode)
	restore = func(nodes []*treeNode) {
		for _, n := range nodes {
			if n.isDir && open[n.path] {
				n.open = true
				n.children = t.readDir(n.path)
				restore(n.children)
			}
		}
	}
	restore(t.nodes)
	t.flatten()
}

func (t *fileTree) readDir(rel string) []*treeNode {
	dir := t.root
	if rel != "" {
		dir = filepath.Join(t.root, rel)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var dirs, files []*treeNode
	for _, e := range entries {
		name := e.Name()
		if _, skip := skipNames[name]; skip {
			continue
		}
		if !e.IsDir() {
			if _, skip := t.gitignore[name]; skip {
				continue
			}
		}
		childRel := name
		if rel != "" {
			childRel = filepath.ToSlash(filepath.Join(rel, name))
		}
		n := &treeNode{name: name, path: childRel, isDir: e.IsDir()}
		if e.IsDir() {
			dirs = append(dirs, n)
		} else {
			files = append(files, n)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	return append(dirs, files...)
}

func (t *fileTree) flatten() {
	t.flat = t.flat[:0]
	var walk func([]*treeNode)
	walk = func(nodes []*treeNode) {
		for _, n := range nodes {
			t.flat = append(t.flat, n)
			if n.isDir && n.open {
				if n.children == nil {
					n.children = t.readDir(n.path)
				}
				walk(n.children)
			}
		}
	}
	walk(t.nodes)
	if t.cursor >= len(t.flat) {
		t.cursor = len(t.flat) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *fileTree) current() *treeNode {
	if t.cursor < 0 || t.cursor >= len(t.flat) {
		return nil
	}
	return t.flat[t.cursor]
}

func (t *fileTree) move(delta int) {
	if len(t.flat) == 0 {
		return
	}
	t.cursor += delta
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor >= len(t.flat) {
		t.cursor = len(t.flat) - 1
	}
}

func (t *fileTree) toggle() {
	n := t.current()
	if n == nil || !n.isDir {
		return
	}
	n.open = !n.open
	if !n.open {
		n.children = nil
	}
	t.flatten()
}

func depthOf(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/") + 1
}
