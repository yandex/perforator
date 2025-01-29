package format

type ProfileData struct {
	Nodes   [][]RenderingNode `json:"rows"`
	Strings []string          `json:"stringTable"`
	Meta    ProfileMeta       `json:"meta"`
}

type StringIndex = int

type ProfileMeta struct {
	EventType StringIndex `json:"eventType"`
	FrameType StringIndex `json:"frameType"`
	Version   int         `json:"version"`
}

type RenderingNode struct {
	ParentIndex     int         `json:"parentIndex"`
	TextID          StringIndex `json:"textId"`
	SampleCount     int64       `json:"sampleCount"`
	EventCount      float64     `json:"eventCount"`
	BaseEventCount  float64     `json:"baseEventCount,omitempty"`
	BaseSampleCount int64       `json:"baseSampleCount,omitempty"`
	FrameOrigin     StringIndex `json:"frameOrigin"`
	Kind            StringIndex `json:"kind"`
	File            StringIndex `json:"file"`
	Inlined         bool        `json:"inlined,omitempty"`
}
