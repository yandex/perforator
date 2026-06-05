package profile

import (
	"sort"
	"sync/atomic"
	"time"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/profile/merge"
)

////////////////////////////////////////////////////////////////////////////////

type SpecialMapping = string

const (
	KernelSpecialMapping SpecialMapping = "[kernel]"
	// jit code for which exact runtime is not known
	JITSpecialMapping    SpecialMapping = "[jit]"
	JVMSpecialMapping    SpecialMapping = "[jvm]"
	PHPSpecialMapping    SpecialMapping = "[php]"
	PythonSpecialMapping SpecialMapping = "[python]"
)

var (
	SpecialMappings = map[string]bool{
		string(KernelSpecialMapping): true,
		string(JITSpecialMapping):    true,
		string(JVMSpecialMapping):    true,
		string(PHPSpecialMapping):    true,
		string(PythonSpecialMapping): true,
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

	// Timestamp accumulation for timestamped samples.
	// Zero values mean no timestamped samples have been added yet.
	minTimestamp time.Time
	maxTimestamp time.Time
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
		profile: NewProfile(),
		caches: NewDefaultMap(func(k uint32) *ProcessCache {
			return NewProcessCache(k, ids)
		}, nil),
		ownsCaches: true,
	}
}

func NewBuilderWithCaches(caches *DefaultMap[uint32, ProcessCache]) *Builder {
	return &Builder{
		profile:    NewProfile(),
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

func (b *Builder) resetMinMaxTimestamps() {
	b.minTimestamp = time.Time{}
	b.maxTimestamp = time.Time{}
}

func (b *Builder) finishBase() *Profile {
	defer b.resetMinMaxTimestamps()
	finishedProfile := b.profile

	if b.hasTimestampedSamples() {
		finishedProfile.TimeNanos = b.minTimestamp.UnixNano()
		finishedProfile.DurationNanos = b.maxTimestamp.UnixNano() - b.minTimestamp.UnixNano()
	}

	sampleTypeCopy := make([]*profile.ValueType, 0, len(b.profile.SampleType))
	for _, st := range finishedProfile.SampleType {
		sampleTypeCopy = append(sampleTypeCopy, &profile.ValueType{
			Type: st.Type,
			Unit: st.Unit,
		})
	}
	b.profile = &Profile{Profile: &profile.Profile{SampleType: sampleTypeCopy}}

	if b.ownsCaches {
		b.caches.Clear()
	}

	return finishedProfile
}

// FinishRaw finalizes the profile for yaprof format.
func (b *Builder) FinishRaw() *Profile {
	finishedProfile := b.finishBase()
	populateProfileTables(finishedProfile)
	return finishedProfile
}

func populateProfileTables(p *Profile) {
	locations := make(map[uint64]*profile.Location)
	mappings := make(map[uint64]*profile.Mapping)
	functions := make(map[uint64]*profile.Function)

	for _, sample := range p.Sample {
		for _, location := range sample.Location {
			if location == nil {
				continue
			}
			locations[location.ID] = location
			if location.Mapping != nil {
				mappings[location.Mapping.ID] = location.Mapping
			}
			for _, line := range location.Line {
				if line.Function != nil {
					functions[line.Function.ID] = line.Function
				}
			}
		}
	}

	p.Location = make([]*profile.Location, 0, len(locations))
	for _, location := range locations {
		p.Location = append(p.Location, location)
	}
	sort.Slice(p.Location, func(i, j int) bool {
		return p.Location[i].ID < p.Location[j].ID
	})

	p.Mapping = make([]*profile.Mapping, 0, len(mappings))
	for _, mapping := range mappings {
		p.Mapping = append(p.Mapping, mapping)
	}
	sort.Slice(p.Mapping, func(i, j int) bool {
		return p.Mapping[i].ID < p.Mapping[j].ID
	})

	p.Function = make([]*profile.Function, 0, len(functions))
	for _, function := range functions {
		p.Function = append(p.Function, function)
	}
	sort.Slice(p.Function, func(i, j int) bool {
		return p.Function[i].ID < p.Function[j].ID
	})
}

// Finish finalizes the profile for pprof format.
func (b *Builder) Finish() *Profile {
	finishedProfile := b.finishBase()
	finishedProfile.PeriodType = &profile.ValueType{}

	finishedCompactedProfile, err := merge.Merge([]*profile.Profile{finishedProfile.Profile})
	if err != nil {
		panic(err)
	}

	return &Profile{Profile: finishedCompactedProfile}
}

// Add should not be called if AddTimestampedSample was called.
// User should choose to add samples via Add or AddTimestampedSample.
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

// AddTimestampedSample creates a new sample builder that will accumulate the given
// timestamp for automatic StartTime/EndTime derivation in Builder.Finish().
// User should choose to add samples via Add or AddTimestampedSample.
func (b *Builder) AddTimestampedSample(pid uint32, ts time.Time) *SampleBuilder {
	sb := b.Add(pid)
	sb.timestamp = &ts
	return sb
}

func (b *Builder) hasTimestampedSamples() bool {
	return !b.minTimestamp.IsZero()
}

func (b *Builder) feedSampleTimestamp(ts time.Time) {
	if b.minTimestamp.IsZero() || ts.Before(b.minTimestamp) {
		b.minTimestamp = ts
	}
	if ts.After(b.maxTimestamp) {
		b.maxTimestamp = ts
	}
}

////////////////////////////////////////////////////////////////////////////////

type SampleBuilder struct {
	pid       uint32
	cache     *ProcessCache
	sample    *profile.Sample
	parent    *Builder
	timestamp *time.Time
}

func (b *SampleBuilder) Finish() *Builder {
	// There could be zero locations in a sample (for example,
	// when LBR is enabled but not supported by the hardware).
	//
	// Such samples don't have any value, so we don't collect them.
	if len(b.sample.Location) > 0 {
		b.parent.profile.Sample = append(b.parent.profile.Sample, b.sample)
	}
	if b.timestamp != nil {
		b.parent.feedSampleTimestamp(*b.timestamp)
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
func (b *SampleBuilder) AddNativeLocationUncached(address uint64) *LocationBuilder {
	loc := b.cache.newLocation(address)

	return &LocationBuilder{b.cache, b, loc}
}

// Must be called before all AddNativeLocation calls.
// Shitty but we want to adapt to pprof here until new profile format
// Interpreter frames lay right before all native frames in *profile.Sample.Location
func (b *SampleBuilder) AddInterpreterLocation(key *InterpreterLocationKey) *LocationBuilder {
	loc, isnew := b.cache.GetOrAddInterpreterLocation(key)
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
	nf, isnew := b.cache.GetOrAddFunction(&functionKey{
		name:       f.Name,
		systemName: f.SystemName,
		filename:   f.Filename,
		startLine:  f.StartLine,
	})
	if isnew {
		f.ID = nf.ID
		*nf = *f
	}
	b.line.Function = nf
	b.parent.location.Line = append(b.parent.location.Line, b.line)

	return b.parent
}

////////////////////////////////////////////////////////////////////////////////

type InterpreterLocationKey struct {
	ObjectAddress uint64
	Linestart     int32
}

type functionKey struct {
	name       string
	systemName string
	filename   string
	startLine  int64
}

func isSpecialMapping(mp *profile.Mapping) bool {
	return mp.Start == 0 && SpecialMappings[mp.File]
}

type ProcessCache struct {
	nativeLocations      map[uint64]*profile.Location
	interpreterLocations map[InterpreterLocationKey]*profile.Location
	mappings             map[uint64]*profile.Mapping
	specialMappings      map[string]*profile.Mapping
	functions            map[functionKey]*profile.Function
	ids                  *ids
}

func NewProcessCache(pid uint32, ids *ids) *ProcessCache {
	return &ProcessCache{
		nativeLocations:      make(map[uint64]*profile.Location),
		interpreterLocations: make(map[InterpreterLocationKey]*profile.Location),
		mappings:             make(map[uint64]*profile.Mapping),
		specialMappings:      make(map[string]*profile.Mapping),
		functions:            make(map[functionKey]*profile.Function),
		ids:                  ids,
	}
}

func (c *ProcessCache) newLocation(address uint64) *profile.Location {
	return &profile.Location{
		ID:      c.ids.nextLocation.Add(1),
		Address: address,
	}
}

func (c *ProcessCache) GetOrAddNativeLocation(address uint64) (loc *profile.Location, isnew bool) {
	l, ok := c.nativeLocations[address]
	if ok {
		return l, false
	}

	l = c.newLocation(address)

	c.nativeLocations[address] = l
	return l, true
}

func (c *ProcessCache) GetOrAddInterpreterLocation(key *InterpreterLocationKey) (loc *profile.Location, isnew bool) {
	l, ok := c.interpreterLocations[*key]
	if ok {
		return l, false
	}

	l = &profile.Location{
		ID:      c.ids.nextLocation.Add(1),
		Address: key.ObjectAddress,
	}

	c.interpreterLocations[*key] = l
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

func (c *ProcessCache) GetOrAddFunction(key *functionKey) (loc *profile.Function, isnew bool) {
	f, ok := c.functions[*key]
	if ok {
		return f, false
	}

	f = &profile.Function{
		ID:   c.ids.nextFunction.Add(1),
		Name: key.name,
	}

	c.functions[*key] = f
	return f, true
}

////////////////////////////////////////////////////////////////////////////////

type ids struct {
	nextLocation atomic.Uint64
	nextMapping  atomic.Uint64
	nextFunction atomic.Uint64
}
