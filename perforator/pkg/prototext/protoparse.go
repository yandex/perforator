package prototext

import (
	"io/ioutil"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func ParseFile(path string, conf protoreflect.ProtoMessage) error {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = prototext.Unmarshal(body, conf)
	if err != nil {
		return err
	}
	return nil
}
