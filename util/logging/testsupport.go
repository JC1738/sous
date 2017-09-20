package logging

import (
	"fmt"
	"time"

	"github.com/nyarly/spies"
)

type (
	metricsSinkSpy struct {
		spy *spies.Spy
	}

	metricsSinkController struct {
		*spies.Spy
	}

	writeDonerSpy struct {
		spy *spies.Spy
	}

	writeDonerController struct {
		*spies.Spy
	}

	logSinkSpy struct {
		spy *spies.Spy
	}

	logSinkController struct {
		*spies.Spy
		Metrics metricsSinkController
		Console writeDonerController
	}
)

// NewLogSinkSpy returns a spy/controller pair
func NewLogSinkSpy() (LogSink, logSinkController) {
	spy := spies.NewSpy()
	return logSinkSpy{spy: spy}, logSinkController{Spy: spy}
}

// NewLogSinkBundle returns a spy/controller for a LogSink with the Console and
// Metrics calls prepared to return MetricsSink and WriteDoner spies.
func NewLogSinkBundle() (LogSink, logSinkController) {
	spy, ctrl := NewLogSinkSpy()

	console, cc := NewWriteDonerSpy()
	ctrl.Console = cc
	ctrl.MatchMethod("Console", spies.AnyArgs, console)

	metrics, mc := NewMetricsSpy()
	ctrl.Metrics = mc
	ctrl.MatchMethod("Metrics", spies.AnyArgs, metrics)

	return spy, ctrl
}

func (lss logSinkSpy) LogMessage(lvl Level, msg LogMessage) {
	lss.spy.Called(lvl, msg)
}

// These do what LogSet does so that it'll be easier to replace the interface

// Vomitf is part of the interface for LogSink
func (lss logSinkSpy) Vomitf(f string, as ...interface{}) {
	m := NewGenericMsg(ExtraDebug1Level, fmt.Sprintf(f, as...), nil)
	Deliver(m, lss)
}

// Debugf is part of the interface for LogSink.
func (lss logSinkSpy) Debugf(f string, as ...interface{}) {
	m := NewGenericMsg(DebugLevel, fmt.Sprintf(f, as...), nil)
	Deliver(m, lss)
}

// Warnf is part of the interface for LogSink.
func (lss logSinkSpy) Warnf(f string, as ...interface{}) {
	m := NewGenericMsg(WarningLevel, fmt.Sprintf(f, as...), nil)
	Deliver(m, lss)
}

// Child is part of the interface for LogSink.
func (lss logSinkSpy) Child(name string) LogSink {
	lss.spy.Called(name)
	return lss //easier than managing a whole new lss
}

// Console is part of the interface for LogSink
func (lss logSinkSpy) Console() WriteDoner {
	res := lss.spy.Called()
	return res.Get(0).(WriteDoner)
}

// Metrics is part of the interface for LogSink
func (lss logSinkSpy) Metrics() MetricsSink {
	res := lss.spy.Called()
	return res.Get(0).(MetricsSink)
}

// NewMetricsSpy returns a spy/controller pair
func NewMetricsSpy() (MetricsSink, metricsSinkController) {
	spy := spies.NewSpy()
	return metricsSinkSpy{spy}, metricsSinkController{spy}
}

// ClearCounter is part of the MetricsSink interface.
func (mss metricsSinkSpy) ClearCounter(name string) {
	mss.spy.Called(name)
}

// IncCounter is part of the MetricsSink interface.
func (mss metricsSinkSpy) IncCounter(name string, amount int64) {
	mss.spy.Called(name, amount)
}

// DecCounter is part of the MetricsSink interface.
func (mss metricsSinkSpy) DecCounter(name string, amount int64) {
	mss.spy.Called(name, amount)
}

// UpdateTimer is part of the MetricsSink interface.
func (mss metricsSinkSpy) UpdateTimer(name string, dur time.Duration) {
	mss.spy.Called(name, dur)
}

// UpdateTimerSince is part of the MetricsSink interface.
func (mss metricsSinkSpy) UpdateTimerSince(name string, time time.Time) {
	mss.spy.Called(name, time)
}

// UpdateSample is part of the MetricsSink interface.
func (mss metricsSinkSpy) UpdateSample(name string, value int64) {
	mss.spy.Called(name, value)
}

// Done is part of the MetricsSink interface.
func (mss metricsSinkSpy) Done() {
	mss.spy.Called()
}

// NewWriteDonerSpy returns a spy/controller pair for WriteDoner
func NewWriteDonerSpy() (WriteDoner, writeDonerController) {
	spy := spies.NewSpy()
	return writeDonerSpy{spy: spy}, writeDonerController{Spy: spy}
}

// Write is part of the WriteDoner interface.
func (wds writeDonerSpy) Write(p []byte) (n int, err error) {
	res := wds.spy.Called()
	return res.Int(0), res.Error(1)
}

// Done is part of the WriteDoner interface.
func (wds writeDonerSpy) Done() {
	wds.spy.Called()
}
