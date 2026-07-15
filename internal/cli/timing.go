package cli

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type ruleTiming struct {
	duration time.Duration
	calls    int
}

type runTimings struct {
	mu          sync.Mutex
	started     time.Time
	parse       time.Duration
	semantic    time.Duration
	controlFlow time.Duration
	project     time.Duration
	output      time.Duration
	rules       map[string]ruleTiming
}

func newRunTimings() *runTimings {
	return &runTimings{started: time.Now(), rules: make(map[string]ruleTiming)}
}

func (t *runTimings) observeLint(event lint.TimingEvent) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	switch event.Stage {
	case lint.TimingParse:
		t.parse += event.Duration
	case lint.TimingSemantic:
		t.semantic += event.Duration
	case lint.TimingControlFlow:
		t.controlFlow += event.Duration
	case lint.TimingRule:
		entry := t.rules[event.RuleID]
		entry.duration += event.Duration
		entry.calls++
		t.rules[event.RuleID] = entry
	}
}

func (t *runTimings) observeProject(event project.TimingEvent) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	switch event.Stage {
	case project.TimingParse:
		t.parse += event.Duration
	case project.TimingSemantic:
		t.semantic += event.Duration
	}
}

func (t *runTimings) addProject(duration time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.project += duration
	t.mu.Unlock()
}

func (t *runTimings) addOutput(duration time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.output += duration
	t.mu.Unlock()
}

func (t *runTimings) write(w io.Writer) {
	if t == nil {
		return
	}
	t.mu.Lock()
	parse := t.parse
	semantic := t.semantic
	controlFlow := t.controlFlow
	projectDuration := t.project
	outputDuration := t.output
	rules := make(map[string]ruleTiming, len(t.rules))
	var ruleDuration time.Duration
	for id, entry := range t.rules {
		rules[id] = entry
		ruleDuration += entry.duration
	}
	total := time.Since(t.started)
	t.mu.Unlock()
	_, _ = fmt.Fprintln(w, "pawnlint timings:")
	_, _ = fmt.Fprintf(w, "  %-16s %s\n", "parse", timingDuration(parse))
	_, _ = fmt.Fprintf(w, "  %-16s %s\n", "semantic", timingDuration(semantic))
	_, _ = fmt.Fprintf(w, "  %-16s %s\n", "control-flow", timingDuration(controlFlow))
	_, _ = fmt.Fprintf(w, "  %-16s %s\n", "project", timingDuration(projectDuration))
	_, _ = fmt.Fprintf(w, "  %-16s %s\n", "rules", timingDuration(ruleDuration))
	_, _ = fmt.Fprintf(w, "  %-16s %s\n", "output", timingDuration(outputDuration))
	_, _ = fmt.Fprintf(w, "  %-16s %s\n", "total", timingDuration(total))
	if len(rules) == 0 {
		return
	}
	type row struct {
		id string
		ruleTiming
	}
	rows := make([]row, 0, len(rules))
	for id, entry := range rules {
		rows = append(rows, row{id: id, ruleTiming: entry})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].duration != rows[j].duration {
			return rows[i].duration > rows[j].duration
		}
		return rows[i].id < rows[j].id
	})
	_, _ = fmt.Fprintln(w, "pawnlint rule timings:")
	for _, entry := range rows {
		_, _ = fmt.Fprintf(w, "  %-32s %6d  %s\n", entry.id, entry.calls, timingDuration(entry.duration))
	}
}

func timingDuration(duration time.Duration) string {
	if duration < time.Microsecond {
		return duration.Round(time.Nanosecond).String()
	}
	return duration.Round(time.Microsecond).String()
}
