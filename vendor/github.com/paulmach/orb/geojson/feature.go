package geojson

import (
	"bytes"
	"fmt"

	"github.com/paulmach/orb"
	"go.mongodb.org/mongo-driver/bson"
)

// A Feature corresponds to GeoJSON feature object
type Feature struct {
	ID         interface{}  `json:"id,omitempty"`
	Type       string       `json:"type"`
	BBox       BBox         `json:"bbox,omitempty"`
	Geometry   orb.Geometry `json:"geometry"`
	Properties Properties   `json:"properties"`

	// ExtraMembers can be used to encoded/decode extra key/members in
	// the base of the feature object. Note that keys of "id", "type", "bbox"
	// "geometry" and "properties" will not work as those are reserved by the
	// GeoJSON spec.
	ExtraMembers Properties `json:"-"`
}

// NewFeature creates and initializes a GeoJSON feature given the required attributes.
func NewFeature(geometry orb.Geometry) *Feature {
	return &Feature{
		Type:       "Feature",
		Geometry:   geometry,
		Properties: make(map[string]interface{}),
	}
}

// Point implements the orb.Pointer interface so that Features can be used
// with quadtrees. The point returned is the center of the Bound of the geometry.
// To represent the geometry with another point you must create a wrapper type.
func (f *Feature) Point() orb.Point {
	return f.Geometry.Bound().Center()
}

var _ orb.Pointer = &Feature{}

// MarshalJSON converts the feature object into the proper JSON.
// It will handle the encoding of all the child geometries.
// Alternately one can call json.Marshal(f) directly for the same result.
// Items in the ExtraMembers map will be included in the base of the
// feature object.
func (f Feature) MarshalJSON() ([]byte, error) {
	return marshalJSON(newFeatureDoc(&f))
}

// MarshalBSON converts the feature object into the proper JSON.
// It will handle the encoding of all the child geometries.
// Alternately one can call json.Marshal(f) directly for the same result.
// Items in the ExtraMembers map will be included in the base of the
// feature object.
func (f Feature) MarshalBSON() ([]byte, error) {
	return bson.Marshal(newFeatureDoc(&f))
}

func newFeatureDoc(f *Feature) interface{} {
	if len(f.ExtraMembers) == 0 {
		doc := &featureDoc{
			ID:         f.ID,
			Type:       "Feature",
			Properties: f.Properties,
			BBox:       f.BBox,
			Geometry:   NewGeometry(f.Geometry),
		}

		if len(doc.Properties) == 0 {
			doc.Properties = nil
		}

		return doc
	}

	var tmp map[string]interface{}
	if f.ExtraMembers != nil {
		tmp = f.ExtraMembers.Clone()
	} else {
		tmp = make(map[string]interface{}, 3)
	}

	delete(tmp, "id")
	if f.ID != nil {
		tmp["id"] = f.ID
	}
	tmp["type"] = "Feature"

	delete(tmp, "bbox")
	if f.BBox != nil {
		tmp["bbox"] = f.BBox
	}

	tmp["geometry"] = NewGeometry(f.Geometry)

	if len(f.Properties) == 0 {
		tmp["properties"] = nil
	} else {
		tmp["properties"] = f.Properties
	}

	return tmp
}

// UnmarshalFeature decodes the data into a GeoJSON feature.
// Alternately one can call json.Unmarshal(f) directly for the same result.
func UnmarshalFeature(data []byte) (*Feature, error) {
	f := &Feature{}
	err := f.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// UnmarshalJSON handles the correct unmarshalling of the data
// into the orb.Geometry types.
func (f *Feature) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte(`null`)) {
		*f = Feature{}
		return nil
	}

	tmp := make(map[string]nocopyRawMessage, 4)

	err := unmarshalJSON(data, &tmp)
	if err != nil {
		return err
	}

	*f = Feature{}
	for key, value := range tmp {
		switch key {
		case "id":
			err := unmarshalJSON(value, &f.ID)
			if err != nil {
				return err
			}
		case "type":
			err := unmarshalJSON(value, &f.Type)
			if err != nil {
				return err
			}
		case "bbox":
			err := unmarshalJSON(value, &f.BBox)
			if err != nil {
				return err
			}
		case "geometry":
			g := &Geometry{}
			err := unmarshalJSON(value, &g)
			if err != nil {
				return err
			}

			if g != nil {
				f.Geometry = g.Geometry()
			}
		case "properties":
			err := unmarshalJSON(value, &f.Properties)
			if err != nil {
				return err
			}
		default:
			if f.ExtraMembers == nil {
				f.ExtraMembers = Properties{}
			}

			var val interface{}
			err := unmarshalJSON(value, &val)
			if err != nil {
				return err
			}
			f.ExtraMembers[key] = val
		}
	}

	if f.Type != "Feature" {
		return fmt.Errorf("geojson: not a feature: type=%s", f.Type)
	}

	return nil
}

// UnmarshalBSON will unmarshal a BSON document created with bson.Marshal.
func (f *Feature) UnmarshalBSON(data []byte) error {
	tmp := make(map[string]bson.RawValue, 4)

	err := bson.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	*f = Feature{}
	for key, value := range tmp {
		switch key {
		case "id":
			err := value.Unmarshal(&f.ID)
			if err != nil {
				return err
			}
		case "type":
			f.Type, _ = bson.RawValue(value).StringValueOK()
		case "bbox":
			err := value.Unmarshal(&f.BBox)
			if err != nil {
				return err
			}
		case "geometry":
			g := &Geometry{}
			err := value.Unmarshal(&g)
			if err != nil {
				return err
			}

			if g != nil {
				f.Geometry = g.Geometry()
			}
		case "properties":
			err := value.Unmarshal(&f.Properties)
			if err != nil {
				return err
			}
		default:
			if f.ExtraMembers == nil {
				f.ExtraMembers = Properties{}
			}

			var val interface{}
			err := value.Unmarshal(&val)
			if err != nil {
				return err
			}
			f.ExtraMembers[key] = val
		}
	}

	if f.Type != "Feature" {
		return fmt.Errorf("geojson: not a feature: type=%s", f.Type)
	}

	return nil
}

type featureDoc struct {
	ID         interface{} `json:"id,omitempty" bson:"id"`
	Type       string      `json:"type" bson:"type"`
	BBox       BBox        `json:"bbox,omitempty" bson:"bbox,omitempty"`
	Geometry   *Geometry   `json:"geometry" bson:"geometry"`
	Properties Properties  `json:"properties" bson:"properties"`
}
