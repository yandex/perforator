package perfevent

// ID is a unique identifier of a single perf event.
type ID uint64

type TypeMeta struct {
	Name string
	Unit string
}

type Type struct {
	Type   uint32
	Config uint64
	Meta   TypeMeta
}

// IsProfilingEventType implements profilingevent.Type
func (t *Type) IsProfilingEventType() {}

// Name implements profilingevent.Type
func (t *Type) Name() string {
	return t.Meta.Name
}

// Unit implements profilingevent.Type
func (t *Type) Unit() string {
	return t.Meta.Unit
}

// String implements profilingevent.Type
func (t *Type) String() string {
	return t.Meta.Name + "." + t.Meta.Unit
}
