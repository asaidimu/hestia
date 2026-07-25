package http

import (
	"fmt"
	"strings"
)

type trieNode struct {
	segment    string
	children   []*trieNode
	paramChild *trieNode
	paramName  string
	handlers   map[string]Handler
}

type pathTrie struct {
	root *trieNode
}

func newPathTrie() *pathTrie {
	return &pathTrie{root: &trieNode{}}
}

func (t *pathTrie) insert(method, path string, handler Handler) {
	segments := splitPath(path)
	node := t.root
	for _, seg := range segments {
		node = node.findOrCreate(seg)
	}
	if node.handlers == nil {
		node.handlers = make(map[string]Handler)
	}
	if _, exists := node.handlers[method]; exists {
		panic(fmt.Sprintf("duplicate route: %s %s", method, path))
	}
	node.handlers[method] = handler
}

func (t *pathTrie) lookup(method, path string) (Handler, map[string]string, bool) {
	segments := splitPath(path)
	node := t.root
	params := make(map[string]string)

	for _, seg := range segments {
		found := false
		for _, child := range node.children {
			if child.segment == seg {
				node = child
				found = true
				break
			}
		}
		if found {
			continue
		}
		if node.paramChild != nil {
			node = node.paramChild
			params[node.paramName] = seg
			continue
		}
		return nil, nil, false
	}

	h, ok := node.handlers[method]
	if !ok {
		return nil, nil, false
	}
	return h, params, true
}

func (n *trieNode) findOrCreate(segment string) *trieNode {
	if isParam(segment) {
		if n.paramChild == nil {
			n.paramChild = &trieNode{
				paramName: strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}"),
			}
		}
		return n.paramChild
	}
	for _, child := range n.children {
		if child.segment == segment {
			return child
		}
	}
	child := &trieNode{segment: segment}
	n.children = append(n.children, child)
	return child
}

func splitPath(path string) []string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func isParam(seg string) bool {
	return len(seg) > 2 && seg[0] == '{' && seg[len(seg)-1] == '}'
}
