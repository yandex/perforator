package custom_profiling_operation

import (
	"errors"

	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/proto/lib/time_interval"
)

func validateUprobeSettings(uprobeSettings *cpo_proto.UprobeSettings) error {
	if uprobeSettings.BinaryLocation == nil {
		return errors.New("binary location is not set")
	}

	if len(uprobeSettings.ELFTarget) == 0 {
		return errors.New("elf target is not set")
	}

	switch uprobeSettings.BinaryLocation.Location.(type) {
	case *cpo_proto.BinaryLocation_Path:
		if uprobeSettings.BinaryLocation.GetPath() == "" {
			return errors.New("binary location path is not set")
		}
	case *cpo_proto.BinaryLocation_Detector:
		// this requires scanning some binaries to find the necessary one
		return errors.New("binary detector is not supported yet")
	}

	for _, elfTarget := range uprobeSettings.ELFTarget {
		if elfTarget.ELFFileLocation == nil || elfTarget.ELFFileLocation.Location == nil {
			return errors.New("elf target location is not set")
		}

		switch loc := elfTarget.ELFFileLocation.Location.(type) {
		case *cpo_proto.ELFFileLocation_Symbol:
			if loc.Symbol == "" {
				return errors.New("symbol is not set")
			}
		case *cpo_proto.ELFFileLocation_VirtualAddress:
			return errors.New("elf uprobe target: virtual address is not supported yet")
		case *cpo_proto.ELFFileLocation_FileOffset:
			return errors.New("elf uprobe target: file offset is not supported yet")
		}
	}

	return nil
}

func validateTimeInterval(timeInterval *time_interval.TimeInterval) error {
	if timeInterval == nil || timeInterval.From == nil || timeInterval.To == nil {
		return errors.New("time interval is not set")
	}

	if timeInterval.From.AsTime().IsZero() {
		return errors.New("time interval is invalid: start is zero")
	}

	if timeInterval.To.AsTime().IsZero() {
		return errors.New("time interval is invalid: end is zero")
	}

	if timeInterval.From.AsTime().After(timeInterval.To.AsTime()) {
		return errors.New("time interval is invalid: start is after end")
	}

	return nil
}

func ValidateOperationSpec(spec *cpo_proto.OperationSpec) error {
	if spec == nil {
		return errors.New("spec is not set")
	}

	if spec.Target == nil || spec.Target.Target == nil {
		return errors.New("target is not set")
	}

	switch target := spec.Target.Target.(type) {
	case *cpo_proto.Target_Pod: // requires moving deploy system to PerforatorAgent
	case *cpo_proto.Target_NodeCgroup: // not implemented yet
		return errors.New("target type is not supported")
	case *cpo_proto.Target_NodeProcess:
		if target.NodeProcess == nil {
			return errors.New("node process is not set")
		}
		if target.NodeProcess.ProcessID == 0 {
			return errors.New("process id for node process target is not set")
		}
	}

	if spec.Event == nil || spec.Event.Settings == nil {
		return errors.New("event is not set")
	}

	switch eventSettings := spec.Event.Settings.Settings.(type) {
	case *cpo_proto.EventSettings_Uprobe:
		err := validateUprobeSettings(eventSettings.Uprobe)
		if err != nil {
			return err
		}
	case *cpo_proto.EventSettings_PerfEvent:
		// Not implemented yet.
		// This requires mathcing samples from BPF with events by perf_event_id
		// and routing each event's sample to the corresponding profile.
		return errors.New("perf event is not supported yet")
	}

	err := validateTimeInterval(spec.TimeInterval)
	if err != nil {
		return err
	}

	return nil
}
