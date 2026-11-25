package podtracker

import (
	"context"
	"regexp"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/deploysystem"
)

type podFilterRule struct {
	pattern *regexp.Regexp
	skip    bool
	reason  string
}

// filteringSource matches pod names against deny rules first, then allow rules.
// The first match wins. Unmatched pods follow the configured default.
type filteringSource struct {
	source         deploysystem.Source
	log            log.Logger
	rules          []podFilterRule
	defaultEnabled bool
}

func newFilteringSource(source deploysystem.Source, l log.Logger, options config.PodProfileOptions) *filteringSource {
	filter := &filteringSource{source: source, log: l, defaultEnabled: options.Default}
	addRule := func(pattern string, skip bool, reason string) {
		re, err := regexp.Compile(pattern)
		if err != nil {
			l.Warn("Ignoring invalid pod filter", log.String("pattern", pattern), log.Error(err))
			return
		}
		filter.rules = append(filter.rules, podFilterRule{pattern: re, skip: skip, reason: reason})
	}

	for _, entry := range options.DenyList {
		addRule(entry.Pattern, true, entry.Reason)
	}
	for _, entry := range options.AllowList {
		addRule(entry.Pattern, false, "")
	}
	return filter
}

func (f *filteringSource) skip(podName string) (bool, string) {
	for _, rule := range f.rules {
		if rule.pattern.MatchString(podName) {
			return rule.skip, rule.reason
		}
	}
	if f.defaultEnabled {
		return false, ""
	}
	return true, "perforator is disabled by default"
}

func (f *filteringSource) Snapshot(ctx context.Context) (deploysystem.Snapshot, error) {
	snapshot, err := f.source.Snapshot(ctx)
	if err != nil {
		return deploysystem.Snapshot{}, err
	}
	selected := make([]deploysystem.Pod, 0, len(snapshot.Pods))
	for _, pod := range snapshot.Pods {
		if skip, reason := f.skip(pod.Name); skip {
			f.log.Info("Filtered pod", log.String("pod_name", pod.Name), log.String("reason", reason))
			continue
		}
		selected = append(selected, pod)
	}
	snapshot.Pods = selected
	return snapshot, nil
}
