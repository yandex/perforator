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

// TODO: while deleting pprof make sure that this struct contains only bytes (Bundle),
// and all other metadata is extracted from them instead of keeping *gprofile.Profile
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
