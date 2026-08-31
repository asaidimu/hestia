package http

import (
        "fmt"
        "strings"
)

// @note #perf-20260821-003 issue status=open priority=P3 tags=#performance,#struct-packing : trieNode uses slice for children
//
// trieNode struct (line 8) uses a slice for children:
//
// - segment: string (16 bytes)
// - children: []*trieNode (24 bytes - slice)
// - paramChild: *trieNode (8 bytes)
// - paramName: string (16 bytes)
// - handlers: map (8 bytes)
//
// Most trie nodes have 1-2 children. Using a slice causes:
// 1. Heap allocation for the slice header
// 2. Indirection for each child access
// 3. GC pressure from small allocations
//
// For IoT/HFT: Route lookup is on the hot path for every request.
//
// Resolution:
// 1. Use [2]*trieNode for small child counts with fallback to slice
// 2. Or use a fixed-size array with linear search for 1-2 children
// 3. Consider embedding the first child inline
type trieNode struct {
        segment    string
        children   []*trieNode
        paramChild *trieNode
        paramName  string
        handlers   map[string]routeEntry
}

// routeEntry pairs the route handler with per-route transport behavior
// decided at Handle() time — before the body is ever touched.
type routeEntry struct {
        handler       Handler
        streamingBody bool
}

type pathTrie struct {
        root *trieNode
}

func newPathTrie() *pathTrie {
        return &pathTrie{root: &trieNode{}}
}

func (t *pathTrie) insert(method, path string, entry routeEntry) {
        segments := splitPath(path)
        node := t.root
        for _, seg := range segments {
                node = node.findOrCreate(seg)
        }
        if node.handlers == nil {
                node.handlers = make(map[string]routeEntry)
        }
        if _, exists := node.handlers[method]; exists {
                panic(fmt.Sprintf("duplicate route: %s %s", method, path))
        }
        node.handlers[method] = entry
}

func (t *pathTrie) lookup(method, path string) (routeEntry, map[string]string, bool) {
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
                return routeEntry{}, nil, false
        }

        e, ok := node.handlers[method]
        if !ok {
                return routeEntry{}, nil, false
        }
        return e, params, true
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
