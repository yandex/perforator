package inmemorytaskservice

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

func TestCreateServiceAndAddTasks(t *testing.T) {
	logger, err := xlog.TryNew(zap.NewDeployLogger(log.DebugLevel))
	require.NoError(t, err)
	taskService, err := NewInMemoryTaskService(&Config{}, logger, nop.Registry{})
	require.NoError(t, err)

	ctx := context.Background()
	ids := make([]asynctask.TaskID, 0, 3)
	metas := []*perforator.TaskMeta{
		{},
		{},
		{},
	}

	for _, meta := range metas {
		id, err := taskService.AddTask(ctx, meta, &perforator.TaskSpec{})
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, taskService.taskMap[id].Meta, meta)
		assert.Equal(t, taskService.taskMap[id].Status.State, perforator.TaskState_Created)

		ids = append(ids, id)
	}

	for i, id := range ids {
		task, err := taskService.GetTask(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, task.ID)
		assert.Equal(t, metas[i], task.Meta)
	}

	_, err = taskService.GetTask(ctx, "")
	require.Error(t, err, "there is no task with id:  ")
}

func TestFinishTask(t *testing.T) {
	config := &Config{}
	logger, err := xlog.TryNew(zap.NewDeployLogger(log.DebugLevel))
	require.NoError(t, err)
	taskService, err := NewInMemoryTaskService(config, logger, nil)

	require.NoError(t, err)

	ctx := context.Background()

	meta := &perforator.TaskMeta{}
	spec := &perforator.TaskSpec{}

	id, err := taskService.AddTask(ctx, meta, spec)
	require.NoError(t, err)

	result := &perforator.TaskResult{}

	err = taskService.FinishTask(ctx, id, result)
	require.NoError(t, err)

	task, err := taskService.GetTask(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, perforator.TaskState_Finished, task.Status.State)
	assert.Equal(t, result, task.Result)
}

func TestFailTask(t *testing.T) {
	config := &Config{}
	logger, err := xlog.TryNew(zap.NewDeployLogger(log.DebugLevel))
	require.NoError(t, err)
	taskService, err := NewInMemoryTaskService(config, logger, nil)

	require.NoError(t, err)

	ctx := context.Background()

	meta := &perforator.TaskMeta{}
	spec := &perforator.TaskSpec{}

	id, err := taskService.AddTask(ctx, meta, spec)
	require.NoError(t, err)

	errMsg := "task failed due to error"

	err = taskService.FailTask(ctx, id, errMsg)
	require.NoError(t, err)

	task, err := taskService.GetTask(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, perforator.TaskState_Failed, task.Status.State)
	assert.Equal(t, errMsg, task.Status.Error)
}

func TestListTasks(t *testing.T) {
	logger, err := xlog.TryNew(zap.NewDeployLogger(log.DebugLevel))
	require.NoError(t, err)
	taskService, err := NewInMemoryTaskService(&Config{}, logger, nop.Registry{})
	require.NoError(t, err)

	ctx := context.Background()
	ids := make([]asynctask.TaskID, 0, 3)

	metas := []*perforator.TaskMeta{
		{
			Author:       "Alice",
			CreationTime: 100000},
		{
			Author:       "Alice",
			CreationTime: 203000},
		{
			Author:       "Alice",
			CreationTime: 206000},
		{
			Author:       "Alice",
			CreationTime: 210000},
		{
			Author:       "Alice",
			CreationTime: 215000},
		{
			Author:       "Alice",
			CreationTime: 336000},
		{
			Author:       "Tom",
			CreationTime: 400000},
		{
			Author:       "Bob",
			CreationTime: 207000},
		{
			Author:       "Bob",
			CreationTime: 225000},
		{
			Author:       "Bob",
			CreationTime: 252000},
		{
			Author:       "Tom",
			CreationTime: 202000},
	}

	idsMap := make(map[asynctask.TaskID]*perforator.TaskMeta)

	for _, meta := range metas {
		ts := meta.CreationTime
		id, err := taskService.AddTask(ctx, meta, &perforator.TaskSpec{})
		meta.CreationTime = ts
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Equal(t, taskService.taskMap[id].Meta, meta)
		assert.Equal(t, taskService.taskMap[id].Status.State, perforator.TaskState_Created)
		taskService.taskMap[id].Meta = meta

		idsMap[id] = meta
	}

	for i, id := range ids {
		task, err := taskService.GetTask(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, task.ID)
		assert.Equal(t, metas[i], task.Meta)
	}

	for _, test := range []struct {
		name     string
		filter   *asynctask.TaskFilter
		limit    uint64
		offset   uint64
		expected []asynctask.Task
	}{
		{
			name: "simple",
			filter: &asynctask.TaskFilter{
				Author: "Alice",
				From:   time.UnixMicro(200000),
				To:     time.UnixMicro(220000),
			},
			offset: 0,
			limit:  100,
			expected: []asynctask.Task{
				{
					Meta: &perforator.TaskMeta{
						Author:       "Alice",
						CreationTime: 203000},
				},
				{
					Meta: &perforator.TaskMeta{
						Author:       "Alice",
						CreationTime: 206000},
				},
				{
					Meta: &perforator.TaskMeta{
						Author:       "Alice",
						CreationTime: 210000},
				},
				{
					Meta: &perforator.TaskMeta{
						Author:       "Alice",
						CreationTime: 215000},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res, err := taskService.ListTasks(context.Background(), test.filter, test.limit, test.offset)
			require.NoError(t, err)
			require.True(t, equalTasks(test.expected, res))
		})
	}
}

func equalTasks(t1, t2 []asynctask.Task) bool {
	if len(t1) != len(t2) {
		return false
	}

	for i, task := range t1 {
		if task.Meta.CreationTime != t2[i].Meta.CreationTime && task.Meta.Author != t2[i].Meta.Author {
			return false
		}
	}

	return true
}
