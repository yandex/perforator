package profilingevent

// Type is just a handle to some underlying event type,
// either perf event or some custom one.
// Particular value is assigned at runtime and should be retrieved
// upon registration or via call to `GetTypeByNameOrAlias()`.
type Type interface {
	String() string
	Name() string
	Unit() string

	IsProfilingEventType()
}
