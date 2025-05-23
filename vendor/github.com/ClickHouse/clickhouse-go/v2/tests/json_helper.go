package tests

import (
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/chcol"
)

var JSONTestDate, _ = time.Parse(time.RFC3339, "2024-12-13T02:09:30.123Z")

type TestStructAddress struct {
	Street  string `chType:"String"`
	City    string `chType:"String"`
	Country string `chType:"String"`
}

type TestStruct struct {
	Name   string
	Age    int64
	Active bool
	Score  float64

	Tags    []string
	Numbers []int64

	Address TestStructAddress

	KeysNumbers map[string]int64
	Metadata    map[string]interface{}

	Timestamp time.Time `chType:"DateTime64(3)"`

	DynamicString chcol.Dynamic
	DynamicInt    chcol.Dynamic
	DynamicMap    chcol.Dynamic
}

// FastTestStruct is a distinctly separate type that implements clickhouse.JSONSerializer and clickhouse.JSONDeserializer
// The struct must be a separate type since the JSON column is unable to ignore the interface implementation.
type FastTestStruct struct {
	ts TestStruct
}

// SerializeClickHouseJSON implements clickhouse.JSONSerializer for faster struct appending
func (fts *FastTestStruct) SerializeClickHouseJSON() (*clickhouse.JSON, error) {
	obj := chcol.NewJSON()
	obj.SetValueAtPath("Name", fts.ts.Name)
	obj.SetValueAtPath("Age", fts.ts.Age)
	obj.SetValueAtPath("Active", fts.ts.Active)
	obj.SetValueAtPath("Score", fts.ts.Score)
	obj.SetValueAtPath("Tags", fts.ts.Tags)
	obj.SetValueAtPath("Numbers", fts.ts.Numbers)
	obj.SetValueAtPath("Address.Street", fts.ts.Address.Street)
	obj.SetValueAtPath("Address.City", fts.ts.Address.City)
	obj.SetValueAtPath("Address.Country", fts.ts.Address.Country)
	obj.SetValueAtPath("KeysNumbers", fts.ts.KeysNumbers)
	obj.SetValueAtPath("Metadata.FieldA", fts.ts.Metadata["FieldA"])
	obj.SetValueAtPath("Metadata.FieldB", fts.ts.Metadata["FieldB"])
	obj.SetValueAtPath("Metadata.FieldC.FieldD", fts.ts.Metadata["FieldC"].(map[string]any)["FieldD"])
	obj.SetValueAtPath("Timestamp", fts.ts.Timestamp)
	obj.SetValueAtPath("DynamicString", fts.ts.DynamicString)
	obj.SetValueAtPath("DynamicInt", fts.ts.DynamicInt)
	obj.SetValueAtPath("DynamicMap", fts.ts.DynamicMap)

	return obj, nil
}

// DeserializeClickHouseJSON implements clickhouse.JSONDeserializer for faster struct scanning
func (fts *FastTestStruct) DeserializeClickHouseJSON(obj *clickhouse.JSON) error {
	fts.ts.Name, _ = clickhouse.ExtractJSONPathAs[string](obj, "Name")
	fts.ts.Age, _ = clickhouse.ExtractJSONPathAs[int64](obj, "Age")
	fts.ts.Active, _ = clickhouse.ExtractJSONPathAs[bool](obj, "Active")
	fts.ts.Score, _ = clickhouse.ExtractJSONPathAs[float64](obj, "Score")
	fts.ts.Tags, _ = clickhouse.ExtractJSONPathAs[[]string](obj, "Tags")
	fts.ts.Numbers, _ = clickhouse.ExtractJSONPathAs[[]int64](obj, "Numbers")
	fts.ts.Address.Street, _ = clickhouse.ExtractJSONPathAs[string](obj, "Address.Street")
	fts.ts.Address.City, _ = clickhouse.ExtractJSONPathAs[string](obj, "Address.City")
	fts.ts.Address.Country, _ = clickhouse.ExtractJSONPathAs[string](obj, "Address.Country")
	fts.ts.KeysNumbers, _ = clickhouse.ExtractJSONPathAs[map[string]int64](obj, "KeysNumbers")
	fts.ts.Metadata = make(map[string]any)
	fts.ts.Metadata["FieldA"], _ = clickhouse.ExtractJSONPathAs[string](obj, "Metadata.FieldA")
	fts.ts.Metadata["FieldB"], _ = clickhouse.ExtractJSONPathAs[int64](obj, "Metadata.FieldB")
	fts.ts.Metadata["FieldC"] = make(map[string]any)
	fts.ts.Metadata["FieldC"].(map[string]any)["FieldD"], _ = clickhouse.ExtractJSONPathAs[string](obj, "Metadata.FieldC.FieldD")
	fts.ts.Timestamp, _ = clickhouse.ExtractJSONPathAs[time.Time](obj, "Timestamp")
	fts.ts.DynamicString, _ = clickhouse.ExtractJSONPathAs[clickhouse.Dynamic](obj, "DynamicString")
	fts.ts.DynamicInt, _ = clickhouse.ExtractJSONPathAs[clickhouse.Dynamic](obj, "DynamicInt")
	fts.ts.DynamicMap, _ = clickhouse.ExtractJSONPathAs[clickhouse.Dynamic](obj, "DynamicMap")

	return nil
}

func BuildTestJSONPaths() *chcol.JSON {
	ts := BuildTestJSONStruct()
	fts := FastTestStruct{ts: ts}
	jsonObj, _ := fts.SerializeClickHouseJSON()
	return jsonObj
}

func BuildTestJSONStruct() TestStruct {
	return TestStruct{
		Name:    "JSON",
		Age:     42,
		Active:  true,
		Score:   3.14,
		Tags:    []string{"a", "b"},
		Numbers: []int64{20, 40},
		Address: TestStructAddress{
			Street:  "Street",
			City:    "City",
			Country: "Country",
		},
		KeysNumbers: map[string]int64{"FieldA": 42, "FieldB": 32},
		Metadata: map[string]interface{}{
			"FieldA": "a",
			"FieldB": "b",
			"FieldC": map[string]interface{}{
				"FieldD": "d",
			},
		},
		Timestamp:     JSONTestDate,
		DynamicString: chcol.NewDynamic("str").WithType("String"),
		DynamicInt:    chcol.NewDynamic(int64(48)).WithType("Int64"),
		DynamicMap:    chcol.NewDynamic(map[string]string{"a": "a", "b": "b"}).WithType("Map(String, String)"),
	}
}

func BuildFastTestJSONStruct() FastTestStruct {
	ts := BuildTestJSONStruct()
	return FastTestStruct{ts: ts}
}
