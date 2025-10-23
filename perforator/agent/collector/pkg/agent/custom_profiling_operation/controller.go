package custom_profiling_operation

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/models"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
	cpo_internal "github.com/yandex/perforator/perforator/internal/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

var (
	_ models.OperationController = (*operationController)(nil)
)

type operationController struct {
	l                        xlog.Logger
	profiler                 *profiler.Profiler
	profilerResourcesClosers []profiler.Closer

	id   models.OperationID
	spec *models.OperationSpec
}

func newOperationController(l xlog.Logger, profiler *profiler.Profiler, id models.OperationID, spec *models.OperationSpec) (*operationController, error) {
	err := cpo_internal.ValidateOperationSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to validate spec: %w", err)
	}

	for _, feature := range spec.Features {
		switch feature.Feature.(type) {
		case *cpo_proto.Feature_CollectStackAbsoluteTimestampsFeature:
			if _, err := procfs.GetBootTime(); err != nil {
				return nil, fmt.Errorf("failed to get host boot time: %w", err)
			}
		}
	}

	c := &operationController{
		l:        l.With(log.String("operation_id", string(id))),
		profiler: profiler,
		id:       id,
		spec:     spec,
	}

	return c, nil
}

func (o *operationController) releaseProfilerResources() error {
	errs := []error{}
	for _, closer := range o.profilerResourcesClosers {
		err := closer.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func outputProfileName(id models.OperationID) string {
	return fmt.Sprintf("cpo_%s", string(id))
}

func (o *operationController) createUprobes(ctx context.Context, eventSettings *cpo_proto.EventSettings_Uprobe, target *cpo_proto.Target) error {
	baseUprobeConfig := uprobe.Config{
		Path:              eventSettings.Uprobe.BinaryLocation.GetPath(),
		OutputProfileName: outputProfileName(o.id),
	}
	switch target.Target.(type) {
	case *cpo_proto.Target_NodeProcess:
		baseUprobeConfig.Pid = int(target.GetNodeProcess().ProcessID)
	}

	uprobeConfigs := []uprobe.Config{}
	for _, elfTarget := range eventSettings.Uprobe.ELFTarget {
		uprobeConfig := baseUprobeConfig
		switch loc := elfTarget.ELFFileLocation.Location.(type) {
		case *cpo_proto.ELFFileLocation_Symbol:
			uprobeConfig.Symbol = loc.Symbol
		}
		uprobeConfigs = append(uprobeConfigs, uprobeConfig)
	}

	for _, uprobeConfig := range uprobeConfigs {
		uprobe := o.profiler.UprobeManager().Create(uprobeConfig)
		err := uprobe.Attach()
		if err != nil {
			return fmt.Errorf("failed to attach uprobe: %w", err)
		}
		o.l.Info(ctx, "Attached uprobe", log.Error(err))
		o.profilerResourcesClosers = append(o.profilerResourcesClosers, uprobe)
	}

	return nil
}

func (o *operationController) Start(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err := o.releaseProfilerResources()
			if err != nil {
				o.l.Logger().Error("Failed to release profiler resources on CPO start failure", log.Error(err))
			}
		}
	}()

	switch eventSettings := o.spec.Event.Settings.Settings.(type) {
	case *cpo_proto.EventSettings_Uprobe:
		err := o.createUprobes(ctx, eventSettings, o.spec.Target)
		if err != nil {
			return fmt.Errorf("failed to create uprobes: %w", err)
		}
	}

	profileLabels := map[string]string{}
	if o.spec.ProfileLabels != nil {
		maps.Copy(profileLabels, o.spec.ProfileLabels)
	}
	profileLabels[profilequerylang.CPOIDLabel] = string(o.id)

	traceOpts := []profiler.TraceOption{
		profiler.WithProfileLabels(profileLabels),
	}
	for _, feature := range o.spec.Features {
		switch feature.Feature.(type) {
		case *cpo_proto.Feature_CollectStackAbsoluteTimestampsFeature:
			traceOpts = append(traceOpts, profiler.WithAbsoluteSampleTimeCollection())
		}
	}

	switch target := o.spec.Target.Target.(type) {
	case *cpo_proto.Target_NodeProcess:
		closer, err := o.profiler.TracePid(linux.ProcessID(target.NodeProcess.ProcessID), traceOpts...)
		if err != nil {
			return fmt.Errorf("failed to trace pid %d: %w", target.NodeProcess.ProcessID, err)
		}
		o.profilerResourcesClosers = append(o.profilerResourcesClosers, closer)
	}

	return nil
}

func (o *operationController) Stop() error {
	return o.releaseProfilerResources()
}
