package machine

import (
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/uprobe"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux/kallsyms"
)

type bpfLinks struct {
	l log.Logger

	finishTaskSwitch link.Link
	signalDeliver    link.Link

	uprobes        []uprobe.Uprobe
	uprobeRegistry *uprobe.Registry
}

func (l *bpfLinks) close() error {
	if l == nil {
		return nil
	}

	var errs []error
	if l.finishTaskSwitch != nil {
		errs = append(errs, l.finishTaskSwitch.Close())
	}
	if l.signalDeliver != nil {
		errs = append(errs, l.signalDeliver.Close())
	}

	for _, uprobe := range l.uprobes {
		errs = append(errs, uprobe.Close())
	}
	l.uprobes = l.uprobes[:0]

	return errors.Join(errs...)
}

func (l *bpfLinks) setup(conf *Config, progs *unwinder.Progs) (err error) {
	defer func() {
		if err != nil {
			closeErr := l.close()
			if closeErr != nil {
				l.l.Error("Failed to close links on failed setupLinks", log.Error(closeErr))
			}
		}
	}()

	if enabled := conf.TraceWallTime; enabled != nil && *enabled {
		l.finishTaskSwitch, err = createKprobeBySymbolRegex(`^finish_task_switch(\.isra\.\d+)?$`, progs.PerforatorFinishTaskSwitch)
		if err != nil {
			return fmt.Errorf("failed to setup kprobe finish_task_switch link: %w", err)
		}
	}

	if enabled := conf.TraceSignals; enabled != nil && *enabled {
		l.signalDeliver, err = link.Tracepoint("signal", "signal_deliver", progs.PerforatorSignalDeliver, nil)
		if err != nil {
			return fmt.Errorf("failed to setup tracepoint signal_deliver link: %w", err)
		}
	}

	for _, uprobeConfig := range conf.Uprobes {
		opts := []uprobe.Option{}
		if uprobeConfig.Pid != 0 {
			opts = append(opts, uprobe.WithPID(uint32(uprobeConfig.Pid)))
		}
		uprobe := l.uprobeRegistry.Create(uprobeConfig.Config, opts...)
		err = uprobe.Attach(progs.PerforatorUprobe)
		if err != nil {
			return fmt.Errorf("failed to attach uprobe: %w", err)
		}
		l.uprobes = append(l.uprobes, uprobe)
	}

	return nil
}

func newBPFLinks(l log.Logger) *bpfLinks {
	return &bpfLinks{
		l:              l,
		uprobeRegistry: uprobe.NewRegistry(),
	}
}

// See https://github.com/iovisor/bcc/pull/3315
func findKernelSymbolsByRegex(name string) ([]string, error) {
	resolver, err := kallsyms.DefaultKallsymsResolver()
	if err != nil {
		return nil, err
	}
	symbols, err := resolver.LookupSymbolRegex(name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup kprobe symbol %s: %w", name, err)
	}

	return symbols, nil
}

func createKprobeBySymbolRegex(symbolRegex string, prog *ebpf.Program) (link.Link, error) {
	symbols, err := findKernelSymbolsByRegex(symbolRegex)
	if err != nil {
		return nil, err
	}

	var errs []error
	for _, symbol := range symbols {
		kprobe, err := link.Kprobe(symbol, prog, nil)
		if err == nil {
			return kprobe, nil
		}

		errs = append(errs, err)
	}

	return nil, fmt.Errorf("failed to attach kprobe by regex %s: %w", symbolRegex, errors.Join(errs...))
}
