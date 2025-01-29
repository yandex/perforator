package postgrestaskservice

import (
	"errors"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/yandex/perforator/perforator/internal/asynctask"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type TaskRow struct {
	ID             string  `db:"id"`
	IdempotencyKey *string `db:"idempotency_key"`
	Meta           []byte  `db:"meta"`
	Spec           []byte  `db:"spec"`
	Status         []byte  `db:"status"`
	Result         []byte  `db:"result"`
}

func RowsToTasks(rows []*TaskRow) ([]asynctask.Task, error) {
	res := make([]asynctask.Task, 0, len(rows))
	for _, row := range rows {
		task, err := RowToTask(row)
		if err != nil {
			return nil, err
		}
		res = append(res, *task)
	}

	return res, nil
}

func RowToTask(row *TaskRow) (*asynctask.Task, error) {
	t := asynctask.Task{}
	t.ID = asynctask.TaskID(row.ID)
	t.Meta = new(perforator.TaskMeta)
	t.Spec = new(perforator.TaskSpec)
	t.Status = new(perforator.TaskStatus)
	t.Result = new(perforator.TaskResult)

	err := errors.Join(
		json2proto(row.Meta, t.Meta),
		json2proto(row.Spec, t.Spec),
		json2proto(row.Status, t.Status),
		json2proto(row.Result, t.Result),
	)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func TaskToRow(t *asynctask.Task) (*TaskRow, error) {
	var row TaskRow
	var err error

	row.ID = string(t.ID)
	if t.Meta.IdempotencyKey != "" {
		row.IdempotencyKey = &t.Meta.IdempotencyKey
	}

	if row.Meta, err = proto2json(t.Meta); err != nil {
		return nil, err
	}
	if row.Spec, err = proto2json(t.Spec); err != nil {
		return nil, err
	}
	if row.Status, err = proto2json(t.Status); err != nil {
		return nil, err
	}
	if row.Result, err = proto2json(t.Result); err != nil {
		return nil, err
	}

	return &row, nil
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
