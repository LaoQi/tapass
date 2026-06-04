package model

import (
	"sort"
	"strings"

	"github.com/tapass/tapass-tools/vault"
)

func normalizePath(path string) string {
	if path == "" || path == "/" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

type Node struct {
	ID       string
	Path     string
	Children []*Node
	IsGroup  bool
	Attrs    map[string][]byte
	Expanded bool
}

func NewNode(id, path string) *Node {
	return &Node{
		ID:       id,
		Path:     path,
		Children: make([]*Node, 0),
		Attrs:    make(map[string][]byte),
		Expanded: true,
	}
}

func (n *Node) AddChild(child *Node) {
	n.Children = append(n.Children, child)
	n.IsGroup = true
}

func (n *Node) FindChild(id string) *Node {
	for _, c := range n.Children {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (n *Node) SortChildren() {
	sort.Slice(n.Children, func(i, j int) bool {
		if n.Children[i].IsGroup != n.Children[j].IsGroup {
			return n.Children[i].IsGroup
		}
		return n.Children[i].ID < n.Children[j].ID
	})
	for _, c := range n.Children {
		c.SortChildren()
	}
}

func (n *Node) GetEntries() []*Node {
	var result []*Node
	n.collectEntries(&result)
	return result
}

func (n *Node) DirectEntries() []*Node {
	var result []*Node
	for _, child := range n.Children {
		if !child.IsGroup && len(child.Attrs) > 0 {
			result = append(result, child)
		}
	}
	return result
}

func (n *Node) collectEntries(result *[]*Node) {
	if !n.IsGroup && len(n.Attrs) > 0 {
		*result = append(*result, n)
	}
	for _, c := range n.Children {
		c.collectEntries(result)
	}
}

func BuildTree(entries map[string]vault.Entry) *Node {
	root := NewNode("root", "")

	paths := make([]string, 0, len(entries))
	for k := range entries {
		paths = append(paths, k)
	}
	sort.Strings(paths)

	for _, key := range paths {
		e := entries[key]
		parts := splitPath(key)
		if len(parts) == 0 {
			continue
		}

		node := root
		for i, part := range parts[:len(parts)-1] {
			child := node.FindChild(part)
			if child == nil {
				child = NewNode(part, normalizePath(strings.Join(parts[:i+1], "/")))
				node.AddChild(child)
			}
			node = child
		}

		attrName := parts[len(parts)-1]
		node.Attrs[attrName] = e.Value
	}

	root.SortChildren()
	return root
}

func splitPath(key string) []string {
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return nil
	}
	return strings.Split(key, "/")
}

func GetNodeByPath(root *Node, path string) *Node {
	path = normalizePath(path)
	if path == "" {
		return root
	}
	parts := splitPath(path)
	node := root
	for _, part := range parts {
		child := node.FindChild(part)
		if child == nil {
			return nil
		}
		node = child
	}
	return node
}

func GetAllGroups(root *Node) []*Node {
	var groups []*Node
	root.collectGroups(&groups)
	return groups
}

func (n *Node) collectGroups(result *[]*Node) {
	if n.IsGroup {
		*result = append(*result, n)
	}
	for _, c := range n.Children {
		c.collectGroups(result)
	}
}

func EntryPath(groupPath, entryID string) string {
	groupPath = normalizePath(groupPath)
	if groupPath == "" {
		return "/" + entryID
	}
	return groupPath + "/" + entryID
}
