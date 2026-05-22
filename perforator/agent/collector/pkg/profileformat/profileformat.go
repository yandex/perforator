package profileformat

// ProfileFormat describes which profile format(s) the agent sends to storage.
type ProfileFormat string

const (
	Pprof  ProfileFormat = "pprof"
	Yaprof ProfileFormat = "yaprof"
)
