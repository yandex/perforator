package custom_profiling_operation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/models"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
	cpo_internal "github.com/yandex/perforator/perforator/internal/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/btime"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

var (
	_ models.OperationController = (*operationController)(nil)
)

type operationController struct {
	l        xlog.Logger
	profiler *profiler.Profiler

	uprobes            []profiler.Uprobe
	perfEvents         []*profiler.PerfEvent
	sampleConsumerName string

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
			if _, err := btime.GetBootTime(); err != nil {
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

func (o *operationController) disableEventSources() error {
	errs := []error{}
	for _, uprobe := range o.uprobes {
		err := uprobe.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	for _, bundle := range o.perfEvents {
		err := bundle.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (o *operationController) releaseProfilerResources() error {
	err := o.disableEventSources()
	o.profiler.SampleConsumerRegistry().Unregister(o.sampleConsumerName)
	return err
}

func buildIDString(id models.OperationID) string {
	return fmt.Sprintf("cpo_%s", string(id))
}

func (o *operationController) createUprobesForEvent(ctx context.Context, eventSettings *cpo_proto.EventSettings_Uprobe, target *cpo_proto.Target) error {
	baseUprobeConfig := uprobe.Config{
		OutputProfileName: buildIDString(o.id),
	}

	switch target := target.Target.(type) {
	case *cpo_proto.Target_NodeProcess:
		currentNamespacePID, err := o.convertTargetProcessToCurrentNamespace(ctx, target.NodeProcess)
		if err != nil {
			return err
		}

		baseUprobeConfig.Pid = currentNamespacePID
	}

	switch location := eventSettings.Uprobe.BinaryLocation.Location.(type) {
	case *cpo_proto.BinaryLocation_Path:
		baseUprobeConfig.Path = location.Path
	case *cpo_proto.BinaryLocation_ChrootPath:
		baseUprobeConfig.Path = filepath.Join(fmt.Sprintf("/proc/%d/root", baseUprobeConfig.Pid), location.ChrootPath)
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
			closeErr := uprobe.Close()
			if closeErr != nil {
				o.l.Error(ctx, "Failed to close uprobe", log.Error(closeErr))
			}
			return fmt.Errorf("failed to attach uprobe: %w", err)
		}
		o.l.Info(ctx, "Attached uprobe", log.Error(err))
		o.uprobes = append(o.uprobes, uprobe)
	}

	return nil
}

func (o *operationController) createPerfEvents(ctx context.Context, eventSettings *cpo_proto.EventSettings_PerfEvent, target *cpo_proto.Target) error {
	typ := perfevent.GetTypeByNameOrAlias(eventSettings.PerfEvent.Type)
	if typ == nil {
		return fmt.Errorf("unknown event type: %s", eventSettings.PerfEvent.Type)
	}
	perfEventType, ok := typ.(*perfevent.PerfEventType)
	if !ok {
		return fmt.Errorf("[BUG] event type %s is not a perf event", eventSettings.PerfEvent.Type)
	}

	options := &perfevent.Options{
		Type:   perfEventType,
		Enable: true,
	}

	if eventSettings.PerfEvent.Frequency > 0 {
		options.Frequency = &eventSettings.PerfEvent.Frequency
	} else if eventSettings.PerfEvent.SampleRate > 0 {
		options.SampleRate = &eventSettings.PerfEvent.SampleRate
	} else {
		return fmt.Errorf("neither Frequency nor SampleRate is set for perf event")
	}

	perfTarget := &perfevent.Target{}
	switch t := target.Target.(type) {
	case *cpo_proto.Target_NodeProcess:
		pid, err := o.convertTargetProcessToCurrentNamespace(ctx, t.NodeProcess)
		if err != nil {
			return err
		}
		ipid := int(pid)
		perfTarget.ProcessID = &ipid
	}

	perfEvent, err := o.profiler.PerfEventManager().Open(perfTarget, options)
	if err != nil {
		return fmt.Errorf("failed to create perf event: %w", err)
	}

	err = perfEvent.Attach()
	if err != nil {
		closeErr := perfEvent.Close()
		if closeErr != nil {
			o.l.Error(ctx, "Failed to close perf event", log.Error(closeErr))
		}
		return fmt.Errorf("failed to attach perf event: %w", err)
	}

	o.perfEvents = append(o.perfEvents, perfEvent)
	return nil
}

func (o *operationController) convertTargetProcessToCurrentNamespace(ctx context.Context, nodeProcessTarget *cpo_proto.NodeProcessTarget) (linux.CurrentNamespacePID, error) {
	if nodeProcessTarget.PidNamespaceInode == 0 {
		return linux.CurrentNamespacePID(nodeProcessTarget.ProcessID), nil
	}

	resolvedPID := o.profiler.PidNamespaceIndex().ResolveCurrentNamespacePID(
		linux.NamespacedPID(nodeProcessTarget.ProcessID),
		linux.PIDNamespaceInode(nodeProcessTarget.PidNamespaceInode),
	)
	if resolvedPID == nil || *resolvedPID == 0 {
		o.l.Warn(
			ctx,
			"Failed to resolve namespaced pid into current namespace pid",
			log.Int("namespaced_pid", int(nodeProcessTarget.ProcessID)),
			log.Int("pid_namespace_inode", int(nodeProcessTarget.PidNamespaceInode)),
		)
		return linux.CurrentNamespacePID(0), errors.New("failed to resolve namespaced pid into current namespace pid")
	}

	o.l.Info(
		ctx,
		"Resolved namespaced pid into current namespace pid",
		log.Int("resolved_pid", int(*resolvedPID)),
		log.Int("namespaced_pid", int(nodeProcessTarget.ProcessID)),
		log.Int("pid_namespace_inode", int(nodeProcessTarget.PidNamespaceInode)),
	)

	return *resolvedPID, nil
}

func (o *operationController) setupSampleConsumer(ctx context.Context) (err error) {
	profileLabels := map[string]string{}
	if o.spec.ProfileLabels != nil {
		maps.Copy(profileLabels, o.spec.ProfileLabels)
	}
	profileLabels[profilequerylang.CPOIDLabel] = string(o.id)

	sampleConsumerFeatures := profiler.DefaultSampleConsumerFeatures()
	for _, feature := range o.spec.Features {
		switch feature.Feature.(type) {
		case *cpo_proto.Feature_CollectStackAbsoluteTimestampsFeature:
			sampleConsumerFeatures.EnableSampleTimeCollection = true
		case *cpo_proto.Feature_CollectInnermostPidnsFeature:
			sampleConsumerFeatures.EnableInnermostPidnsCollection = true
		}
	}

	var pid linux.CurrentNamespacePID
	switch target := o.spec.Target.Target.(type) {
	case *cpo_proto.Target_NodeProcess:
		pid, err = o.convertTargetProcessToCurrentNamespace(ctx, target.NodeProcess)
		if err != nil {
			return fmt.Errorf("failed to convert target process to current namespace: %w", err)
		}
	}

	allowedUprobes := make(map[uprobe.BinaryInfo]struct{})
	for _, uprobe := range o.uprobes {
		allowedUprobes[uprobe.Info().BinaryInfo] = struct{}{}
	}

	sampleConsumerName := buildIDString(o.id)
	eventSampleFilters := []profiler.SampleFilterFunc{
		profiler.NewUprobeSampleFilter(o.profiler, allowedUprobes),
	}
	if len(o.perfEvents) > 0 {
		eventSampleFilters = append(eventSampleFilters, profiler.NewPerfEventIDSampleFilter(o.perfEvents...))
	}

	for _, feature := range o.spec.Features {
		switch feature.Feature.(type) {
		case *cpo_proto.Feature_ExperimentalCollectSystemWidePerfEventSamplesFeature:
			eventSampleFilters = append(eventSampleFilters, profiler.NewPerfEventSampleFilter())
		}
	}
	var filters []profiler.SampleFilterFunc
	filters = append(filters, profiler.NewORSampleFilter(eventSampleFilters...))
	if pid != 0 {
		filters = append(filters, profiler.NewPIDOrTIDSampleFilter(pid))
	}

	// Avoid resolving same uprobes from other concurrent CPOs in sample consumer
	localResolver := uprobe.NewResolver()
	for _, u := range o.uprobes {
		localResolver.Add(u.Info())
	}

	err = o.profiler.SampleConsumerRegistry().Register(
		sampleConsumerName,
		profiler.NewFilterSampleConsumerAdapter(
			profiler.NewSimpleSampleConsumer(o.profiler, sampleConsumerFeatures, profileLabels, localResolver),
			profiler.NewANDSampleFilter(filters...),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to register sample consumer: %w", err)
	}
	o.sampleConsumerName = sampleConsumerName

	return nil
}

func (o *operationController) Start(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			releaseErr := o.releaseProfilerResources()
			if releaseErr != nil {
				o.l.Error(ctx, "Failed to release profiler resources on CPO start failure", log.Error(releaseErr))
			}
		}
	}()

	for _, event := range o.spec.Events {
		switch eventSettings := event.Settings.Settings.(type) {
		case *cpo_proto.EventSettings_Uprobe:
			err := o.createUprobesForEvent(ctx, eventSettings, o.spec.Target)
			if err != nil {
				return fmt.Errorf("failed to create uprobes: %w", err)
			}
		case *cpo_proto.EventSettings_PerfEvent:
			err := o.createPerfEvents(ctx, eventSettings, o.spec.Target)
			if err != nil {
				return fmt.Errorf("failed to create perf events: %w", err)
			}
		}
	}

	err = o.setupSampleConsumer(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup sample consumer: %w", err)
	}

	return nil
}

func (o *operationController) Stop(ctx context.Context) error {
	errs := []error{}

	// 1. Disable event sources so BPF stops generating new samples for this operation.
	if err := o.disableEventSources(); err != nil {
		o.l.Error(ctx, "Failed to disable event sources", log.Error(err))
		errs = append(errs, err)
	}

	// 2. Wait until the sample reader processes all samples that were already in the perfbuf.
	if err := o.profiler.WaitForSampleProcessing(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			o.l.Warn(ctx, "Timed out waiting for sample processing", log.Error(err))
		} else {
			o.l.Error(ctx, "Failed to wait for sample processing", log.Error(err))
		}
		errs = append(errs, err)
	}

	// 3. Flush the sample consumer to serialize accumulated samples into a profile.
	sampleConsumer := o.profiler.SampleConsumerRegistry().Get(o.sampleConsumerName)
	if sampleConsumer != nil {
		if err := sampleConsumer.Flush(ctx); err != nil {
			o.l.Error(ctx, "Failed to flush CPO sample consumer", log.Error(err))
			errs = append(errs, err)
		} else {
			o.l.Info(ctx, "Successfully flushed CPO sample consumer")
		}
	}

	// 4. Unregister the sample consumer.
	o.profiler.SampleConsumerRegistry().Unregister(o.sampleConsumerName)

	return errors.Join(errs...)
}
