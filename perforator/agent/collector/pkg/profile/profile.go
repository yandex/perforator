package profile

import (
	"bytes"

	gprofile "github.com/google/pprof/profile"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
	"github.com/yandex/perforator/perforator/proto/pprofprofile"
)

type (
	Sample    = gprofile.Sample
	ValueType = gprofile.ValueType
	Location  = gprofile.Location
	Function  = gprofile.Function
	Mapping   = gprofile.Mapping
	Line      = gprofile.Line
)

// Profile is the mutable model used by the Go builder and its consumers.
// ToResult adapts a completed profile to the serialized Result sent to storage.
type Profile struct {
	*gprofile.Profile
	Bundle *bundle.ProfileBundle
}

func NewProfile() *Profile {
	return &Profile{Profile: &gprofile.Profile{}}
}

func GProfToProfileProto(prof *Profile) (*pprofprofile.Profile, error) {
	var buffer bytes.Buffer
	err := prof.WriteUncompressed(&buffer)
	if err != nil {
		return nil, err
	}

	profileProto := &pprofprofile.Profile{}
	err = proto.Unmarshal(buffer.Bytes(), profileProto)
	if err != nil {
		return nil, err
	}

	return profileProto, nil
}
