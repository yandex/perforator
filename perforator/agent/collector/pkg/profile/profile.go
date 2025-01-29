package profile

import (
	"bytes"

	gprofile "github.com/google/pprof/profile"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/proto/pprofprofile"
)

type (
	Profile   = gprofile.Profile
	Sample    = gprofile.Sample
	ValueType = gprofile.ValueType
	Location  = gprofile.Location
	Function  = gprofile.Function
	Mapping   = gprofile.Mapping
	Line      = gprofile.Line
)

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
