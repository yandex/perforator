package profile

import (
	"sync/atomic"
	"time"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/internal/linguist/python/models"
)

////////////////////////////////////////////////////////////////////////////////

type SpecialMapping = string

const (
	KernelSpecialMapping SpecialMapping = "[kernel]"
)

var (
	SpecialMappings = map[string]bool{
		models.PythonSpecialMapping:  true,
		string(KernelSpecialMapping): true,
	}
)

////////////////////////////////////////////////////////////////////////////////

type SampleType struct {
	Kind string
	Unit string
}

func (s *SampleType) String() string {
	return s.Kind + "." + s.Unit
}

////////////////////////////////////////////////////////////////////////////////

type Builder struct {
	profile     *Profile
	caches      *DefaultMap[uint32, ProcessCache]
	ownsCaches  bool
	sampleTypes map[string]bool
}

func NewProcessCaches() *DefaultMap[uint32, ProcessCache] {
	ids := &ids{}
	return NewDefaultMap(func(k uint32) *ProcessCache {
		return NewProcessCache(k, ids)
	}, nil)
}

func NewBuilder() *Builder {
	ids := &ids{}
	return &Builder{
		profile: &profile.Profile{},
		caches: NewDefaultMap(func(k uint32) *ProcessCache {
			return NewProcessCache(k, ids)
		}, nil),
		ownsCaches: true,
	}
}

func NewBuilderWithCaches(caches *DefaultMap[uint32, ProcessCache]) *Builder {
	return &Builder{
		profile:    &profile.Profile{},
		caches:     caches,
		ownsCaches: false,
	}
}

func (b *Builder) AddSampleType(kind, unit string) *Builder {
	b.profile.SampleType = append(b.profile.SampleType, &profile.ValueType{
		Type: kind,
		Unit: unit,
	})
	return b
}

func (b *Builder) SetDefaultSampleType(name string) *Builder {
	b.profile.DefaultSampleType = name
	return b
}

func (b *Builder) AddComment(comment string) *Builder {
	b.profile.Comments = append(b.profile.Comments, comment)
	return b
}

func (b *Builder) SetStartTime(ts time.Time) *Builder {
	b.profile.TimeNanos = ts.UnixNano()
	return b
}

func (b *Builder) SetEndTime(ts time.Time) *Builder {
	b.profile.DurationNanos = ts.UnixNano() - b.profile.TimeNanos
	return b
}

func (b *Builder) GetStartTime() time.Time {
	return time.Unix(0, b.profile.TimeNanos)
}

func (b *Builder) Finish() *Profile {
	sampleType := make([]*profile.ValueType, len(b.profile.SampleType))
	copy(sampleType, b.profile.SampleType)

	b.profile.PeriodType = &profile.ValueType{}

	// Compactify profile.
	res, err := profile.Merge([]*profile.Profile{b.profile})
	if err != nil {
		panic(err)
	}
	b.profile = &Profile{SampleType: sampleType}

	if b.ownsCaches {
		b.caches.Clear()
	}

	return res
}

func (b *Builder) Add(pid uint32) *SampleBuilder {
	bb := &SampleBuilder{
		pid:   pid,
		cache: b.caches.Get(pid),
		sample: &profile.Sample{
			Label:    make(map[string][]string),
			NumLabel: make(map[string][]int64),
			NumUnit:  make(map[string][]string),
		},
		parent: b,
	}

	return bb
}

////////////////////////////////////////////////////////////////////////////////

type SampleBuilder struct {
	pid    uint32
	cache  *ProcessCache
	sample *profile.Sample
	parent *Builder
}

func (b *SampleBuilder) Finish() *Builder {
	// There could be zero locations in a sample (for example,
	// when LBR is enabled but not supported by the hardware).
	//
	// Such samples don't have any value, so we don't collect them.
	if len(b.sample.Location) > 0 {
		b.parent.profile.Sample = append(b.parent.profile.Sample, b.sample)
	}
	return b.parent
}

func (b *SampleBuilder) AddValue(value int64) *SampleBuilder {
	b.sample.Value = append(b.sample.Value, value)
	return b
}

func (b *SampleBuilder) AddStringLabel(key, value string) *SampleBuilder {
	b.sample.Label[key] = append(b.sample.Label[key], value)
	return b
}

func (b *SampleBuilder) AddIntLabel(key string, value int64, unit string) *SampleBuilder {
	b.sample.NumLabel[key] = append(b.sample.NumLabel[key], value)
	b.sample.NumUnit[key] = append(b.sample.NumUnit[key], unit)
	return b
}

func (b *SampleBuilder) AddNativeLocation(address uint64) *LocationBuilder {
	loc, isnew := b.cache.GetOrAddNativeLocation(address)
	if !isnew {
		b.sample.Location = append(b.sample.Location, loc)
		return &LocationBuilder{b.cache, b, nil}
	}

	return &LocationBuilder{b.cache, b, loc}
}

// Must be called before all AddNativeLocation calls.
// Shitty but we want to adapt to pprof here until new profile format
// Python frames lay right before all native frames in *profile.Sample.Location
func (b *SampleBuilder) AddPythonLocation(key *PythonLocationKey) *LocationBuilder {
	loc, isnew := b.cache.GetOrAddPythonLocation(key)
	if !isnew {
		b.sample.Location = append(b.sample.Location, loc)
		return &LocationBuilder{b.cache, b, nil}
	}

	return &LocationBuilder{b.cache, b, loc}
}

////////////////////////////////////////////////////////////////////////////////

type LocationBuilder struct {
	cache    *ProcessCache
	parent   *SampleBuilder
	location *profile.Location
}

func (b *LocationBuilder) SetMapping() *MappingBuilder {
	return &MappingBuilder{b.cache, b, &profile.Mapping{}}
}

func (b *LocationBuilder) AddFrame() *FrameBuilder {
	return &FrameBuilder{b.cache, b, profile.Line{Function: &profile.Function{}}}
}

func (b *LocationBuilder) Finish() *SampleBuilder {
	if b.location != nil {
		b.parent.sample.Location = append(b.parent.sample.Location, b.location)
	}
	return b.parent
}

////////////////////////////////////////////////////////////////////////////////

type MappingBuilder struct {
	cache   *ProcessCache
	parent  *LocationBuilder
	mapping *profile.Mapping
}

func (b *MappingBuilder) SetBegin(address uint64) *MappingBuilder {
	b.mapping.Start = address
	return b
}

func (b *MappingBuilder) SetEnd(address uint64) *MappingBuilder {
	b.mapping.Limit = address
	return b
}

func (b *MappingBuilder) SetOffset(address uint64) *MappingBuilder {
	b.mapping.Offset = address
	return b
}

func (b *MappingBuilder) SetPath(path string) *MappingBuilder {
	b.mapping.File = path
	return b
}

func (b *MappingBuilder) SetBuildID(id string) *MappingBuilder {
	b.mapping.BuildID = id
	return b
}

func (b *MappingBuilder) Finish() *LocationBuilder {
	if b.parent.location == nil {
		return b.parent
	}

	m, isnew := b.cache.GetOrAddMapping(b.mapping)
	if isnew {
		b.mapping.ID = m.ID
		*m = *b.mapping
	} else {
		*b.mapping = *m
	}
	b.parent.location.Mapping = m

	return b.parent
}

////////////////////////////////////////////////////////////////////////////////

type FrameBuilder struct {
	cache  *ProcessCache
	parent *LocationBuilder
	line   profile.Line
}

func (b *FrameBuilder) SetName(name string) *FrameBuilder {
	b.line.Function.Name = name
	if b.line.Function.SystemName == "" {
		b.line.Function.SystemName = name
	}
	return b
}

func (b *FrameBuilder) SetMangledName(name string) *FrameBuilder {
	b.line.Function.SystemName = name
	if b.line.Function.Name == "" {
		b.line.Function.Name = name
	}
	return b
}

func (b *FrameBuilder) SetFilename(path string) *FrameBuilder {
	b.line.Function.Filename = path
	return b
}

func (b *FrameBuilder) SetStartLine(line int64) *FrameBuilder {
	b.line.Function.StartLine = line
	return b
}

func (b *FrameBuilder) SetLine(line int64) *FrameBuilder {
	b.line.Line = line
	return b
}

func (b *FrameBuilder) SetColumn(column int64) *FrameBuilder {
	b.line.Column = column
	return b
}

func (b *FrameBuilder) Finish() *LocationBuilder {
	if b.parent.location == nil {
		return b.parent
	}

	f := b.line.Function
	nf, isnew := b.cache.GetOrAddFunction(b.line.Function.Name)
	if isnew {
		f.ID = nf.ID
		*nf = *f
	} else {
		*f = *nf
	}
	b.parent.location.Line = append(b.parent.location.Line, b.line)

	return b.parent
}

////////////////////////////////////////////////////////////////////////////////

type PythonLocationKey struct {
	CodeObjectAddress     uint64
	CodeObjectFirstLineNo int32
}

func isSpecialMapping(mp *profile.Mapping) bool {
	return mp.Start == 0 && SpecialMappings[mp.File]
}

type ProcessCache struct {
	nativeLocations map[uint64]*profile.Location
	pythonLocations map[PythonLocationKey]*profile.Location
	mappings        map[uint64]*profile.Mapping
	specialMappings map[string]*profile.Mapping
	functions       map[string]*profile.Function
	ids             *ids
}

func NewProcessCache(pid uint32, ids *ids) *ProcessCache {
	return &ProcessCache{
		nativeLocations: make(map[uint64]*profile.Location),
		pythonLocations: make(map[PythonLocationKey]*profile.Location),
		mappings:        make(map[uint64]*profile.Mapping),
		specialMappings: make(map[string]*profile.Mapping),
		functions:       make(map[string]*profile.Function),
		ids:             ids,
	}
}

func (c *ProcessCache) GetOrAddNativeLocation(address uint64) (loc *profile.Location, isnew bool) {
	l, ok := c.nativeLocations[address]
	if ok {
		return l, false
	}

	l = &profile.Location{
		ID:      c.ids.nextLocation.Add(1),
		Address: address,
	}

	c.nativeLocations[address] = l
	return l, true
}

func (c *ProcessCache) GetOrAddPythonLocation(key *PythonLocationKey) (loc *profile.Location, isnew bool) {
	l, ok := c.pythonLocations[*key]
	if ok {
		return l, false
	}

	l = &profile.Location{
		ID:      c.ids.nextLocation.Add(1),
		Address: key.CodeObjectAddress,
	}

	c.pythonLocations[*key] = l
	return l, true
}

func (c *ProcessCache) getOrAddSpecialMapping(name string) (loc *profile.Mapping, isnew bool) {
	m, ok := c.specialMappings[name]
	if ok {
		return m, false
	}

	m = &profile.Mapping{
		ID: c.ids.nextMapping.Add(1),
	}
	c.specialMappings[name] = m

	return m, true
}

func (c *ProcessCache) GetOrAddMapping(mp *profile.Mapping) (loc *profile.Mapping, isnew bool) {
	if isSpecialMapping(mp) {
		return c.getOrAddSpecialMapping(mp.File)
	}

	m, ok := c.mappings[mp.Start]
	if ok {
		return m, false
	}

	m = &profile.Mapping{
		ID:    c.ids.nextMapping.Add(1),
		Start: mp.Start,
	}

	c.mappings[mp.Start] = m
	return m, true
}

func (c *ProcessCache) GetOrAddFunction(name string) (loc *profile.Function, isnew bool) {
	f, ok := c.functions[name]
	if ok {
		return f, false
	}

	f = &profile.Function{
		ID:   c.ids.nextFunction.Add(1),
		Name: name,
	}

	c.functions[name] = f
	return f, true
}

////////////////////////////////////////////////////////////////////////////////

type ids struct {
	nextLocation atomic.Uint64
	nextMapping  atomic.Uint64
	nextFunction atomic.Uint64
}
