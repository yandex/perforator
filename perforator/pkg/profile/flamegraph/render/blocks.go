package render

import (
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

////////////////////////////////////////////////////////////////////////////////

type counts struct {
	count  int64
	events float64
}

func (w *counts) add(sum float64, count int64) {
	w.events += sum
	w.count += count
}

type FrameOrigin string

const (
	FrameOriginUnknown FrameOrigin = ""
	FrameOriginNative  FrameOrigin = "native"
	FrameOriginKernel  FrameOrigin = "kernel"
	FrameOriginPython  FrameOrigin = "python"
)

////////////////////////////////////////////////////////////////////////////////

type block struct {
	parent      *block
	key         string
	name        string
	kind        string
	level       int
	frameOrigin FrameOrigin
	file        string
	inlined     bool

	nextCount counts
	prevCount counts

	truncated bool

	offset float64
	weight float64

	children map[string]*block
}

func (b *block) add(sum float64, count int64) {
	b.nextCount.add(sum, count)
}

func (b *block) sub(sum float64, count int64) {
	b.prevCount.add(sum, count)
}

type blocksBuilder struct {
	root   *block
	blocks []*block
}

func newBlocksBuilder() *blocksBuilder {
	res := &blocksBuilder{}
	res.root = res.newBlock(nil, "", "all", "", 0)
	return res
}

func (b *blocksBuilder) child(block *block, name, path string) *block {
	fullPath := name + path
	res, found := block.children[fullPath]
	if !found {
		res = b.newBlock(block, fullPath, name, path, block.level+1)
		block.children[fullPath] = res
	}
	return res
}

func (b *blocksBuilder) newBlock(parent *block, key, name, file string, level int) *block {
	res := &block{
		key:      key,
		parent:   parent,
		name:     name,
		level:    level,
		file:     file,
		children: make(map[string]*block, 0),
	}
	b.blocks = append(b.blocks, res)
	return res
}

func (b *blocksBuilder) Finish(minWeight float64) []*block {
	b.trimBlocks(minWeight)
	b.pushDownOffsets(b.root, b.root.nextCount.events, 0.0)
	return b.blocks
}

func (b *blocksBuilder) trimBlocks(minWeight float64) {
	if minWeight < 1e-6 {
		return
	}
	minSamples := minWeight * b.root.nextCount.events

	oldBlocks := b.blocks
	b.blocks = make([]*block, 0, len(oldBlocks))

	for _, blk := range oldBlocks {
		if blk.parent != nil && blk.parent.truncated {
			blk.truncated = true
			continue
		}
		if blk.nextCount.events < minSamples {
			delete(blk.parent.children, blk.key)
			blk.truncated = true
			child := b.child(blk.parent, truncatedStack, "")
			child.frameOrigin = blk.frameOrigin
			child.add(blk.nextCount.events, blk.nextCount.count)
			child.sub(blk.prevCount.events, blk.prevCount.count)
		} else {
			b.blocks = append(b.blocks, blk)
		}
	}
}

func (b *blocksBuilder) pushDownOffsets(block *block, total, offset float64) {
	block.offset = offset
	block.weight = block.nextCount.events / total

	keys := maps.Keys(block.children)
	slices.Sort(keys)
	for _, name := range keys {
		child := block.children[name]
		b.pushDownOffsets(child, total, offset)
		offset += child.nextCount.events / total
	}
}

////////////////////////////////////////////////////////////////////////////////

type blocksIterator struct {
	inverted bool
	sum      float64
	block    *block
	builder  *blocksBuilder
	depth    int
	minus    bool
}

func (b *blocksBuilder) MakeIterator(sum float64, minus bool) *blocksIterator {
	i := &blocksIterator{
		sum:     sum,
		block:   b.root,
		builder: b,
		depth:   0,
		minus:   minus,
	}
	i.add(sum, 1)
	return i
}

func (i *blocksIterator) Advance(name, path string) blockBuilder {
	i.block = i.builder.child(i.block, name, path)
	i.add(i.sum, 1)
	i.depth += 1
	return blockBuilder{i.block}
}

func (i *blocksIterator) Depth() int {
	return i.depth
}

func (i *blocksIterator) add(sum float64, count int64) {
	if i.minus {
		i.block.sub(sum, count)
	} else {
		i.block.add(sum, count)
	}
}

////////////////////////////////////////////////////////////////////////////////

type blockBuilder struct {
	block *block
}

func (b blockBuilder) SetKind(name string) blockBuilder {
	b.block.kind = name
	return b
}

func (b blockBuilder) SetFrameOrigin(origin FrameOrigin) blockBuilder {
	b.block.frameOrigin = origin
	return b
}

func (b blockBuilder) SetInlined(inlined bool) blockBuilder {
	b.block.inlined = inlined
	return b
}

////////////////////////////////////////////////////////////////////////////////
