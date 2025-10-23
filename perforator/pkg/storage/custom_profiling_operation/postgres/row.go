package postgres

import (
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

type operationRow struct {
	ID          string  `db:"id"`
	Meta        []byte  `db:"meta"`
	Spec        []byte  `db:"spec"`
	Status      *[]byte `db:"status"`
	TargetState *[]byte `db:"target_state"`
}

func rowsToOperations(rows []*operationRow) ([]*cpo_proto.Operation, error) {
	res := make([]*cpo_proto.Operation, 0, len(rows))
	for _, row := range rows {
		operation, err := rowToOperation(row)
		if err != nil {
			return nil, err
		}
		res = append(res, operation)
	}

	return res, nil
}

func rowToOperation(row *operationRow) (*cpo_proto.Operation, error) {
	if row == nil {
		return nil, nil
	}

	op := cpo_proto.Operation{}
	op.ID = row.ID

	op.Meta = new(cpo_proto.OperationMeta)
	op.Spec = new(cpo_proto.OperationSpec)
	err := errors.Join(
		json2proto(row.Meta, op.Meta),
		json2proto(row.Spec, op.Spec),
	)

	if row.Status != nil {
		op.Status = new(cpo_proto.OperationStatus)
		err = errors.Join(err, json2proto(*row.Status, op.Status))
	}
	if row.TargetState != nil {
		op.TargetState = new(cpo_proto.OperationTargetState)
		err = errors.Join(err, json2proto(*row.TargetState, op.TargetState))
	}
	if err != nil {
		return nil, err
	}

	return &op, nil
}

func operationToRow(op *cpo_proto.Operation) (*operationRow, error) {
	if op == nil {
		return nil, nil
	}

	var row operationRow
	var err error

	row.ID = op.ID

	if row.Meta, err = proto2json(op.Meta); err != nil {
		return nil, err
	}
	if row.Spec, err = proto2json(op.Spec); err != nil {
		return nil, err
	}
	if op.Status != nil {
		status, err := proto2json(op.Status)
		if err != nil {
			return nil, err
		}
		row.Status = &status
	}
	if op.TargetState != nil {
		targetState, err := proto2json(op.TargetState)
		if err != nil {
			return nil, err
		}
		row.TargetState = &targetState
	}

	return &row, nil
}

func stateToString(state cpo_proto.OperationState) string {
	if state == cpo_proto.OperationState_Unknown {
		// this means the state is not set at all
		return ""
	}

	return state.String()
}

func json2proto(buf []byte, protomsg protoreflect.ProtoMessage) error {
	if buf == nil {
		return nil
	}

	return protojson.Unmarshal(buf, protomsg)
}

func proto2json(protomsg protoreflect.ProtoMessage) ([]byte, error) {
	if protomsg == nil {
		return nil, nil
	}

	buf, err := protojson.Marshal(protomsg)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
