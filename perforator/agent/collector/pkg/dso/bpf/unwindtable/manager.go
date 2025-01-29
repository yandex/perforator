package unwindtable

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

////////////////////////////////////////////////////////////////////////////////

type PageID = unwinder.PageId

const InvalidPageID = PageID(unwinder.UnwindTableInvalidPageId)

////////////////////////////////////////////////////////////////////////////////

type Allocation struct {
	// Allocation state machine.
	state   AllocationState
	statemu sync.Mutex
	// Index of the allocation in the cache.
	// Allows to remove engaged items from the cache efficiently.
	cacheid int
	// Index of the binary.
	id uint64

	BuildID   string
	RowCount  int
	NodeCount int
	Pages     []PageID
}

type AllocationState = int32

const (
	// The allocation is used now. It should not be evicted.
	AllocationStateEngaged AllocationState = iota
	// The allocation is cached. It can be evicted. It can be engaged again.
	AllocationStateCached
	// The allocation is evicting. This state is transient.
	AllocationStateEvicting
	// The allocation is done. It cannot be repaired.
	AllocationStateReleased
)

// So, the allocation lifecycle is:
// None --[BPFManager.New]-> Engaged --[BPFManager.MoveToCache]-> Cached
//                              ^                                  |  |
//                              |                                  |  |
//                              +------[BPFManager.MoveFromCache]--+  |
//                                                                    |
//                     Evicting <------[BPFManager.evict]------------+

////////////////////////////////////////////////////////////////////////////////

type BPFManager struct {
	l        log.Logger
	bpf      *machine.BPF
	freelist *freelist[PageID]
	cache    cache

	metrics struct {
		liveallocs metrics.IntGauge
		liverows   metrics.IntGauge

		cachedallocs metrics.IntGauge
		cachedpages  metrics.IntGauge
		cachedrows   metrics.IntGauge

		rowslost metrics.Counter
		rowsused metrics.Counter

		tablesbuilt    metrics.Counter
		tablesfailed   metrics.Counter
		tablesreleased metrics.Counter

		leafcount metrics.IntGauge
		nodecount metrics.IntGauge
	}
}

func NewBPFManager(l log.Logger, m metrics.Registry, bpf *machine.BPF) (*BPFManager, error) {
	mgr := &BPFManager{
		l:   l.WithName("UnwindManager"),
		bpf: bpf,
	}

	if err := mgr.init(); err != nil {
		return nil, err
	}
	if err := mgr.instrument(m); err != nil {
		return nil, err
	}

	return mgr, nil
}

func (m *BPFManager) instrument(metrics metrics.Registry) error {
	metrics = metrics.WithPrefix("unwind")

	metrics.FuncIntGauge("page.free.count", func() int64 {
		return int64(m.freelist.FreeItems())
	})
	metrics.FuncIntGauge("page.total.count", func() int64 {
		return int64(m.freelist.TotalItems())
	})
	metrics.FuncGauge("page.usage", func() float64 {
		free := float64(m.freelist.FreeItems())
		total := float64(m.freelist.TotalItems())
		return (total - free) / total
	})
	m.metrics.leafcount = metrics.WithTags(map[string]string{"kind": "leaf"}).IntGauge("page.used.count")
	m.metrics.nodecount = metrics.WithTags(map[string]string{"kind": "node"}).IntGauge("page.used.count")

	m.metrics.rowsused = metrics.Counter("row.used.count")
	m.metrics.rowslost = metrics.Counter("row.lost.count")
	m.metrics.tablesbuilt = metrics.Counter("tables.built.count")
	m.metrics.tablesfailed = metrics.Counter("tables.failed.count")
	m.metrics.tablesreleased = metrics.Counter("tables.released.count")
	m.metrics.liveallocs = metrics.IntGauge("alloc.live.count")
	m.metrics.liverows = metrics.IntGauge("row.live.count")
	m.metrics.cachedallocs = metrics.IntGauge("alloc.cached.count")
	m.metrics.cachedpages = metrics.IntGauge("page.cached.count")
	m.metrics.cachedrows = metrics.IntGauge("row.cached.count")

	return nil
}

func (m *BPFManager) init() error {
	npages := int(m.bpf.UnwindTablePartCount()) * int(unwinder.UnwindPageTableNumPagesPerPart)

	m.freelist = newFreelist[PageID](int(npages))
	for id := PageID(0); id < PageID(npages); id++ {
		m.freelist.put(id)
	}

	m.l.Debug("Initialized unwind tables manager", log.Int("pages", m.freelist.FreeItems()))

	return nil
}

func (m *BPFManager) Add(buildID string, id uint64, table *unwind.UnwindTable) (a *Allocation, err error) {
	l := log.With(m.l, log.String("buildid", buildID))
	defer func() {
		if err != nil {
			m.metrics.tablesfailed.Inc()
			l.Warn("Failed to allocate unwind table",
				log.Error(err),
			)
		} else {
			m.metrics.tablesbuilt.Inc()
			l.Debug("Allocated unwind table",
				log.UInt64("id", a.id),
				log.Int("npages", len(a.Pages)),
				log.Int("nrows", a.RowCount),
				log.Int("nnodes", a.NodeCount),
			)
		}
	}()

	res, err := m.registerTable(id, table)
	if err != nil {
		return nil, fmt.Errorf("failed to register unwind table for binary buildID=%s: %w", buildID, err)
	}
	a = &Allocation{
		state:     AllocationStateEngaged,
		id:        id,
		BuildID:   buildID,
		RowCount:  len(table.GetStartPc()),
		NodeCount: res.nodes,
		Pages:     res.pages,
	}

	m.metrics.liveallocs.Add(1)
	m.metrics.liverows.Add(int64(a.RowCount))

	pagecount := int64(len(a.Pages))
	nodecount := int64(a.NodeCount)
	m.metrics.leafcount.Add(pagecount - nodecount)
	m.metrics.nodecount.Add(nodecount)

	return a, nil
}

func (m *BPFManager) Release(a *Allocation) {
	a.statemu.Lock()
	defer a.statemu.Unlock()
	m.release(a)
}

func (m *BPFManager) release(a *Allocation) {
	if a.state == AllocationStateReleased {
		return
	}

	l := log.With(m.l,
		log.String("buildid", a.BuildID),
		log.UInt64("id", a.id),
		log.Int("npages", len(a.Pages)),
	)

	l.Debug("Releasing allocation",
		log.Int32("allocstate", a.state))

	m.uncache(a)
	a.state = AllocationStateReleased

	for _, page := range a.Pages {
		m.freelist.put(page)
	}

	err := m.delRoot(a.id)
	if err != nil {
		l.Error("Failed to delete unwind table root", log.Error(err))
	}

	m.metrics.liverows.Add(-int64(a.RowCount))
	m.metrics.liveallocs.Add(-1)
	m.metrics.tablesreleased.Inc()

	pagecount := int64(len(a.Pages))
	nodecount := int64(a.NodeCount)
	m.metrics.leafcount.Add(-(pagecount - nodecount))
	m.metrics.nodecount.Add(-nodecount)
}

type registeredTable struct {
	pages []PageID
	nodes int
}

func (m *BPFManager) registerTable(id uint64, table *unwind.UnwindTable) (res *registeredTable, err error) {
	b := pageTableBuilder{
		m:     m,
		id:    id,
		table: table,
	}
	pages, err := b.do()
	if err != nil {
		return nil, fmt.Errorf("failed to build page table: %w", err)
	}

	return &registeredTable{pages, len(b.nodes)}, nil
}

func (m *BPFManager) putPage(id PageID, page *unwinder.UnwindTablePage) error {
	if m.bpf == nil {
		return nil
	}
	page.Id = id
	err := m.bpf.PutUnwindTablePage(id, page)
	if err != nil {
		return fmt.Errorf("failed to add root page into unwind table: %w", err)
	}
	return nil
}

func (m *BPFManager) putRoot(id uint64, root PageID) error {
	if m.bpf == nil {
		return nil
	}
	return m.bpf.PutBinaryUnwindTable(unwinder.BinaryId(id), root)
}

func (m *BPFManager) delRoot(id uint64) error {
	if m.bpf == nil {
		return nil
	}
	return m.bpf.DeleteBinaryUnwindTable(unwinder.BinaryId(id))
}

type pageTableBuilder struct {
	m     *BPFManager
	id    uint64
	table *unwind.UnwindTable
	pages []PageID

	page   *unwinder.UnwindTablePageLeaf
	pageid int
	rowid  int

	startpc uint64
	nextpc  uint64

	// page table intermediate pages.
	nodes []*node
	root  *node
}

type link struct {
	leaf PageID
	node *node
}

func (l *link) empty() bool {
	return l.leaf == 0 && l.node == nil
}

type node struct {
	id       PageID
	children [1 << int(unwinder.UnwindPageTableLevel0Width)]link
}

func (b *pageTableBuilder) do() (pg []PageID, err error) {
	defer func() {
		if err != nil {
			b.releasePages()
		}
	}()

	if err := b.allocPages(); err != nil {
		return nil, err
	}

	b.page = &unwinder.UnwindTablePageLeaf{}
	b.pageid = 0
	b.rowid = 0
	b.startpc = 0
	b.nextpc = 0

	b.root, err = b.grabNode()
	if err != nil {
		return nil, err
	}

	for i := range b.table.GetStartPc() {
		if b.rowid >= len(b.page.Ranges) {
			if err := b.flushPage(); err != nil {
				return nil, err
			}
		}

		r := row{b.table, i}
		if b.startpc == 0 {
			b.startpc = r.StartPC()
		}
		b.nextpc = r.StartPC() + r.PCRange()

		fillRule(r, b.page, b.rowid)
		b.rowid++
	}

	if b.rowid > 0 {
		if err := b.flushPage(); err != nil {
			return nil, err
		}
	}

	if err := b.flushNodes(); err != nil {
		return nil, err
	}

	if err := b.m.putRoot(b.id, b.root.id); err != nil {
		return nil, fmt.Errorf("failed to generate root page: %w", err)
	}

	return b.pages, nil
}

type bufwriter struct {
	buf []byte
	pos int
}

func (w *bufwriter) Write(part []byte) (int, error) {
	if w.pos >= len(w.buf) {
		return 0, io.EOF
	}
	n := copy(w.buf[w.pos:], part)
	w.pos += n
	return n, nil
}

func (b *pageTableBuilder) marshalPage(page *unwinder.UnwindTablePage, nested any) error {
	w := bufwriter{buf: page.Kind.UnionBuf[:]}
	err := binary.Write(&w, binary.LittleEndian, nested)
	if err != nil {
		return fmt.Errorf("failed to marshal page: %w", err)
	}
	return nil
}

func (b *pageTableBuilder) allocPages() error {
	nrows := len(b.table.GetStartPc())
	pagesize := len(unwinder.UnwindTablePageLeaf{}.Rules)
	npages := nrows / pagesize
	if nrows%pagesize != 0 {
		npages++
	}

	b.pages = make([]PageID, 0, npages)
	for i := 0; i < npages; i++ {
		if _, err := b.grabPage(); err != nil {
			return fmt.Errorf("failed to allocate %d unwind table pages, have %d", npages, len(b.pages))
		}
	}

	return nil
}

var (
	ErrNoPage = errors.New("no pages available")
)

func (b *pageTableBuilder) grabPage() (PageID, error) {
	for {
		if pageid, ok := b.tryGrabPage(); ok {
			b.pages = append(b.pages, pageid)
			return pageid, nil
		}

		if err := b.tryEvictFromCache(); err != nil {
			return 0, err
		}
	}
}

func (b *pageTableBuilder) tryGrabPage() (PageID, bool) {
	return b.m.freelist.get()
}

func (b *pageTableBuilder) tryEvictFromCache() error {
	top := b.m.cache.pop()
	if top == nil {
		return ErrNoPage
	}
	_ = b.m.evict(top)
	return nil
}

func (b *pageTableBuilder) grabNode() (*node, error) {
	id, err := b.grabPage()
	if err != nil {
		return nil, fmt.Errorf("failed to grab node, have %d pages: %w", len(b.pages), err)
	}

	n := &node{id: id}
	b.nodes = append(b.nodes, n)

	return n, nil
}

func (b *pageTableBuilder) releasePages() {
	for _, page := range b.pages {
		b.m.freelist.put(page)
	}
}

func (b *pageTableBuilder) flushPage() error {
	nextpage := InvalidPageID
	if b.pageid+1 < len(b.pages) {
		nextpage = PageID(b.pages[b.pageid+1])
	}

	pagewrapper := &unwinder.UnwindTablePage{
		Type:         unwinder.UnwindTablePageTypeLeaf,
		NextPage:     nextpage,
		BeginAddress: b.startpc,
		EndAddress:   b.nextpc,
	}
	b.page.Length = uint32(b.rowid)

	if err := b.marshalPage(pagewrapper, b.page); err != nil {
		return err
	}
	if err := b.m.putPage(b.pages[b.pageid], pagewrapper); err != nil {
		return err
	}
	if err := b.populatePageTable(b.startpc, b.nextpc, b.pages[b.pageid]); err != nil {
		return err
	}

	b.m.metrics.rowsused.Add(int64(b.rowid))
	b.m.metrics.rowslost.Add(int64(len(b.page.Ranges) - b.rowid))

	b.pageid++
	b.page = &unwinder.UnwindTablePageLeaf{}
	b.rowid = 0
	b.startpc = 0
	return nil
}

const (
	PageLeafSize uint64 = 256
	PageNodeSize uint64 = 1024
	Mask0        uint64 = ^uint64(PageLeafSize - 1)
	Mask1        uint64 = ^uint64((PageNodeSize-1)<<8) & Mask0
	Mask2        uint64 = ^uint64((PageNodeSize-1)<<18) & Mask1
	Mask3        uint64 = ^uint64((PageNodeSize-1)<<28) & Mask2
)

func (b *pageTableBuilder) populatePageTable(from, to uint64, page PageID) error {
	from = from & Mask0
	to = to & Mask0

	for pc := from; pc <= to; pc += PageLeafSize {
		if err := b.insertPageTable(pc, page); err != nil {
			return err
		}
	}

	return nil
}

// pc = 0000000000000000000000000044444444443333333333222222222211111111
//
//	|page2 id |page1 id |leaf id  |row id
//
// pc0 = (pc >> 28) & PageNodeSize
// pc1 = (pc >> 18) & PageNodeSize
// pc2 = (pc >> 8) & PageNodeSize
// pc3 = pc & PageLeafSize
func (b *pageTableBuilder) insertPageTable(pc uint64, page PageID) (err error) {
	pc0 := (pc >> 28) & (PageNodeSize - 1)
	pc1 := (pc >> 18) & (PageNodeSize - 1)
	pc2 := (pc >> 8) & (PageNodeSize - 1)

	node := b.root
	level := 0

	advance := func(idx uint64) error {
		level++
		if node.children[idx].empty() {
			node.children[idx].node, err = b.grabNode()
			if err != nil {
				return err
			}
		}
		node = node.children[idx].node
		return nil
	}

	if err := advance(pc0); err != nil {
		return err
	}
	if err := advance(pc1); err != nil {
		return err
	}
	if node.children[pc2].leaf == 0 {
		node.children[pc2].leaf = page
	}

	return nil
}

func (b *pageTableBuilder) flushNodes() error {
	b.m.l.Debug("Allocated intermediate nodes", log.Int("count", len(b.nodes)))
	for _, node := range b.nodes {
		page := unwinder.UnwindTablePageNode{}
		for i, l := range node.children {
			if l.node != nil {
				page.Children[i] = l.node.id
			} else {
				page.Children[i] = l.leaf
			}
		}

		wrapper := &unwinder.UnwindTablePage{
			Type: unwinder.UnwindTablePageTypeNode,
		}

		if err := b.marshalPage(wrapper, page); err != nil {
			return err
		}
		if err := b.m.putPage(node.id, wrapper); err != nil {
			return err
		}
	}

	return nil
}

// FIXME(sskvor): Generate this from the BTF
const dwarfUnwindRBPRuleUndefined = 0x7f

func fillRule(row row, page *unwinder.UnwindTablePageLeaf, idx int) {
	rule := unwinder.UnwindRule{}

	// Fill RBP rule
	if rbp := row.RBP(); rbp != nil && rbp.GetCfaPlusOffset() != nil {
		offset := rbp.GetCfaPlusOffset().GetOffset()
		rule.Rbp = unwinder.RbpUnwindRule{Offset: int8(offset)}
	} else {
		rule.Rbp = unwinder.RbpUnwindRule{Offset: dwarfUnwindRBPRuleUndefined}
	}

	// Fill CFA rule
	if cfa := row.CFA(); cfa != nil && cfa.GetRegisterOffset() != nil {
		rule.Cfa.Kind = unwinder.UnwindRuleRegisterOffset
		rule.Cfa.Regno = uint8(cfa.GetRegisterOffset().GetRegister())
		rule.Cfa.Offset = cfa.GetRegisterOffset().GetOffset()
	} else {
		rule.Cfa.Kind = unwinder.UnwindRuleUnsupported
	}

	page.Pc[idx] = uint32(row.StartPC())
	page.Ranges[idx] = uint32(row.PCRange())
	page.Rules[idx] = rule
}

func (m *BPFManager) MoveFromCache(a *Allocation) bool {
	a.statemu.Lock()
	defer a.statemu.Unlock()

	switch a.state {
	case AllocationStateEngaged:
		return true

	case AllocationStateCached:
		m.l.Debug("Moving allocation from cache",
			log.String("buildid", a.BuildID),
			log.Int32("allocstate", a.state),
			log.Int("cacheid", a.cacheid),
		)

		m.uncache(a)
		a.state = AllocationStateEngaged
		return true

	default:
		return false
	}
}

func (m *BPFManager) MoveToCache(a *Allocation) bool {
	a.statemu.Lock()
	defer a.statemu.Unlock()

	switch a.state {
	case AllocationStateEngaged:
		m.l.Debug("Moving allocation to cache",
			log.String("buildid", a.BuildID),
			log.Int32("allocstate", a.state),
		)

		a.state = AllocationStateCached
		m.metrics.cachedallocs.Add(1)
		m.metrics.cachedpages.Add(int64(len(a.Pages)))
		m.metrics.cachedrows.Add(int64(a.RowCount))
		m.cache.put(a)
		return true

	case AllocationStateCached:
		return true

	default:
		return false
	}
}

func (m *BPFManager) uncache(a *Allocation) {
	if a.state == AllocationStateCached || a.state == AllocationStateEvicting {
		m.metrics.cachedallocs.Add(-1)
		m.metrics.cachedpages.Add(-int64(len(a.Pages)))
		m.metrics.cachedrows.Add(-int64(a.RowCount))
	}
	if a.state == AllocationStateCached {
		m.cache.remove(a)
	}
}

func (m *BPFManager) evict(a *Allocation) bool {
	a.statemu.Lock()
	defer a.statemu.Unlock()

	switch a.state {
	case AllocationStateCached:
		m.l.Debug("Evicting allocation from cache",
			log.String("buildid", a.BuildID),
			log.Int32("allocstate", a.state),
			log.Int("npages", len(a.Pages)),
		)
		a.state = AllocationStateEvicting
		m.release(a)
		return true

	default:
		return false
	}
}
