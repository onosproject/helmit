package task

import (
	"github.com/gosuri/uilive"
	"sort"
	"sync"
)

type Node interface {
	write(writer *uilive.Writer, depth int)
}

type Parent interface {
	add(child Node) (index int)
	remove(index int) bool
	list() []Node
}

type Root interface {
	Parent
	write(writer *uilive.Writer)
}

func newRoot() Root {
	return &root{
		Parent: newParent(),
	}
}

type root struct {
	Parent
}

func (r *root) write(writer *uilive.Writer) {
	for _, child := range r.list() {
		child.write(writer, 0)
	}
}

func newParent() Parent {
	return &parent{
		children: make(map[int]Node),
	}
}

type parent struct {
	children map[int]Node
	index    int
	mu       sync.RWMutex
}

func (p *parent) add(child Node) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	index := p.index
	p.children[index] = child
	p.index++
	return index
}

func (p *parent) remove(index int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.children[index]; ok {
		delete(p.children, index)
		return true
	}
	return false
}

func (p *parent) list() []Node {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ids := make([]int, 0, len(p.children))
	for sid := range p.children {
		ids = append(ids, sid)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	children := make([]Node, 0, len(p.children))
	for _, sid := range ids {
		children = append(children, p.children[sid])
	}
	return children
}
