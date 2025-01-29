package asynctask

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/yandex/perforator/perforator/proto/perforator"
)

var (
	ErrAttemptsLimitReached = errors.New("task too many times")
	ErrTaskAlreadyFinished  = errors.New("task is already finished")
)

////////////////////////////////////////////////////////////////////////////////

type TaskService interface {
	GetTask(ctx context.Context, id TaskID) (*Task, error)
	ListTasks(ctx context.Context, filter *TaskFilter, limit uint64, offset uint64) ([]Task, error)
	AddTask(ctx context.Context, meta *perforator.TaskMeta, spec *perforator.TaskSpec) (TaskID, error)
	FinishTask(ctx context.Context, id TaskID, result *perforator.TaskResult) error
	FailTask(ctx context.Context, id TaskID, err string) error
	PickTask(ctx context.Context) (task *Task, stop func(), err error)
}

////////////////////////////////////////////////////////////////////////////////

type TaskFilter struct {
	Author string
	From   time.Time
	To     time.Time
}

////////////////////////////////////////////////////////////////////////////////

type TaskID string

type TaskKey struct {
	ID TaskID `yson:",key"`
}

type Task struct {
	ID     TaskID `yson:",key"`
	Meta   *perforator.TaskMeta
	Spec   *perforator.TaskSpec
	Status *perforator.TaskStatus
	Result *perforator.TaskResult
}

type TaskRow struct {
	ID     TaskID `yson:",key"`
	Meta   any
	Spec   any
	Status any
	Result any
}

type TaskIdempotencyIndexRow struct {
	IdempotencyKey string `yson:",key"`
	ID             TaskID
}

type TaskPendingIndexRow struct {
	ID     TaskID `yson:",key"`
	Meta   any
	Status any
}

type TaskIdempotencyIndexRowKey struct {
	IdempotencyKey string `yson:",key"`
}

////////////////////////////////////////////////////////////////////////////////

func (t *Task) Unmarshal(row *TaskRow) error {
	t.ID = row.ID
	t.Meta = new(perforator.TaskMeta)
	t.Spec = new(perforator.TaskSpec)
	t.Status = new(perforator.TaskStatus)
	t.Result = new(perforator.TaskResult)
	return errors.Join(
		json2proto(row.Meta, t.Meta),
		json2proto(row.Spec, t.Spec),
		json2proto(row.Status, t.Status),
		json2proto(row.Result, t.Result),
	)
}

func (t *Task) Marshal() (*TaskRow, error) {
	var row TaskRow

	row.ID = t.ID
	err := errors.Join(
		proto2json(t.Meta, &row.Meta),
		proto2json(t.Spec, &row.Spec),
		proto2json(t.Status, &row.Status),
		proto2json(t.Result, &row.Result),
	)

	if err != nil {
		return nil, err
	}
	return &row, nil
}

////////////////////////////////////////////////////////////////////////////////

func json2proto(jsonmsg any, protomsg protoreflect.ProtoMessage) error {
	if jsonmsg == nil {
		return nil
	}

	buf, err := json.Marshal(jsonmsg)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(buf, protomsg)
}

func proto2json(protomsg protoreflect.ProtoMessage, jsonmsg any) error {
	if protomsg == nil {
		return nil
	}

	buf, err := protojson.Marshal(protomsg)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, jsonmsg)
}

////////////////////////////////////////////////////////////////////////////////

func (t *Task) GetMeta() *perforator.TaskMeta {
	if t.Meta == nil {
		t.Meta = &perforator.TaskMeta{}
	}
	return t.Meta
}

func (t *Task) GetSpec() *perforator.TaskSpec {
	if t.Spec == nil {
		t.Spec = &perforator.TaskSpec{}
	}
	return t.Spec
}

func (t *Task) GetStatus() *perforator.TaskStatus {
	if t.Status == nil {
		t.Status = &perforator.TaskStatus{}
	}
	return t.Status
}

func (t *Task) GetResult() *perforator.TaskResult {
	if t.Result == nil {
		t.Result = &perforator.TaskResult{}
	}
	return t.Result
}

////////////////////////////////////////////////////////////////////////////////

func IsFinalState(state perforator.TaskState) bool {
	switch state {
	case perforator.TaskState_Failed, perforator.TaskState_Finished:
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////
