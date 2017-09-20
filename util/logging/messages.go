// The goal of this package is to integrate structured logging an metrics
// reporting with error handling in an interface as close as possible to the
// fluency of fmt.Errorf(...)

// or of errors.Wrapf(err, "fmt", args...)

// Concerns:
//   a. structured logging using a defined scheme
//   b. build-time checking of errors
//   c. 3 purposes, which each message type can make use of 1-3 of:
//     logging to ELK,
//     metrics collection
//     error reporting
//   d. Contextualization - i.e. pull message fields from a context.Context
//        or from a logging context likewise contextualized.
//   e. ELK specific fields (i.e. "this is schema xyz")

// Nice to have:
//   z. Output filtering disjoint from creation (i.e. *not* log.debug but rather debug stuff from the singularity API)
//   y. Runtime output filtering, via e.g. HTTP requests.
//   x. A live ringbuffer of all messages

// b & d are in tension.
// also, a with OTLs, because optional fields

package logging

import (
	"bytes"
	"io"
	"time"
)

type (
	messageSink interface {
	}

	/*
		Counter interface {
			Clear()
			Inc(int64)
			Dec(int64)
		}

		// Timer is a write-only interface over a timer.
		Timer interface {
			Time(func())
			Update(time.Duration)
			UpdateSince(time.Time)
		}

		// Updater is a generalization of write-only metrics - integers that can be set.
		// e.g. simple gauges or analyzed samples etc.
		Updater interface {
			Update(int64)
		}

	*/

	temporaryOldStyleLogging interface {
		Debugf(f string, as ...interface{})
		Vomitf(f string, as ...interface{})
		Warnf(f string, as ...interface{})
	}

	// A LogSink can be used in Deliver to send messages for logging.
	LogSink interface {
		temporaryOldStyleLogging

		// Child returns a namespaced child
		Child(name string) LogSink

		// LogMessage is used to record structured LogMessages
		LogMessage(Level, LogMessage)

		// Metrics returns a MetricsSink, which will be used to record MetricsMessages.
		Metrics() MetricsSink

		// Console returns a WriteDoner, which will be used to record ConsoleMessages.
		Console() WriteDoner
	}

	// A MetricsSink is passed into a MetricsMessage's MetricsTo(), so that the
	// it can record its metrics. Once done, the Done method is called - if the
	// metrics are incomplete or insistent, the MetricsSink can then report
	// errors.
	// xxx this facility is preliminary, and Sous doesn't yet record these errors.
	MetricsSink interface {
		ClearCounter(name string)
		IncCounter(name string, amount int64)
		DecCounter(name string, amount int64)

		UpdateTimer(name string, dur time.Duration)
		UpdateTimerSince(name string, time time.Time)

		UpdateSample(name string, value int64)

		Done()
	}

	// A WriteDoner is Something like a WriteCloser, but the Done message also
	// asserts that something useful was written After a console message has been
	// written, the Done method is called, so that the WriteDoner can report
	// about badly formed or missing console messages.
	// xxx this facility is preliminary, and Sous doesn't yet record these errors.
	WriteDoner interface {
		io.Writer
		Done()
	}

	writeDoner struct {
		io.Writer
	}

	// A LogMessage has structured data to report to a log (c.f. Deliver)
	LogMessage interface {
		DefaultLevel() Level
		Message() string
		EachField(FieldReportFn)
	}

	// A MetricsMessage has metrics data to record (c.f. Deliver)
	MetricsMessage interface {
		MetricsTo(MetricsSink)
	}

	// A ConsoleMessage has messages to report to a local human operator (c.f. Deliver)
	ConsoleMessage interface {
		WriteToConsole(console io.Writer)
	}

	// FieldReportFn is used by LogMessages to report their fields.
	FieldReportFn func(string, interface{})

	// error interface{}
)

/*
  A static analysis approach here would:

	Check that the JSON tags on structs matched the schemas they claim.
	Check that schema-required fields tie with params to the contructor.
	Maybe check that contexted messages were always receiving contexts with the right WithValues

	A code generation approach would:

	Take the schemas and produce structs with JSON tags
	Produce constructors for the structs with the required fields.
	Produce LogXXX methods and functions around those constructors.

	We can live without those, probably, if we build the interfaces *as if*...

*/

func nopDoner(w io.Writer) WriteDoner {
	return &writeDoner{w}
}

func (writeDoner) Done() {}

// Deliver is the core of the logging messages design. Messages may implement
// any of LogMessage, MetricsMessage or ConsoleMessage, and appropriate action
// will be taken. The upshot is that messages can be Delivered on the spot and
// later determine what facilities are appropriate.
func Deliver(message interface{}, logger LogSink) {
	silent := true

	if lm, is := message.(LogMessage); is {
		silent = false
		Level := getLevel(lm)
		logger.LogMessage(Level, lm)
	}

	if mm, is := message.(MetricsMessage); is {
		silent = false
		metrics := logger.Metrics()
		mm.MetricsTo(metrics)
		metrics.Done()
	}

	if cm, is := message.(ConsoleMessage); is {
		silent = false
		cm.WriteToConsole(logger.Console())
	}

	if _, dont := message.(*silentMessageError); silent && !dont {
		reportSilentMessage(logger, message)
	}
}

// ClearCounter implements part of LogSink on LogSet
func (ls LogSet) ClearCounter(name string) {
	ls.GetCounter(name).Clear()
}

// IncCounter implements part of LogSink on LogSet
func (ls LogSet) IncCounter(name string, amount int64) {
	ls.GetCounter(name).Inc(amount)
}

// DecCounter implements part of LogSink on LogSet
func (ls LogSet) DecCounter(name string, amount int64) {
	ls.GetCounter(name).Dec(amount)
}

// UpdateTimer implements part of LogSink on LogSet
func (ls LogSet) UpdateTimer(name string, dur time.Duration) {
	ls.GetTimer(name).Update(dur)
}

// UpdateTimerSince implements part of LogSink on LogSet
func (ls LogSet) UpdateTimerSince(name string, time time.Time) {
	ls.GetTimer(name).UpdateSince(time)
}

// UpdateSample implements part of LogSink on LogSet
func (ls LogSet) UpdateSample(name string, value int64) {
	ls.GetUpdater(name).Update(value)
}

// The plan here is to be able to extend this behavior such that e.g. the rules
// for levels of messages can be configured or updated at runtime.
func getLevel(lm LogMessage) Level {
	return lm.DefaultLevel()
}

// ConsoleError receives a ConsoleMessage and returns the string as it would be printed to the console.
// This can be used to implement the error interface on ConsoleMessages
func ConsoleError(msg ConsoleMessage) string {
	buf := &bytes.Buffer{}
	msg.WriteToConsole(buf)
	return buf.String()
}

// DefaultLevel is a convenience - by embedding a Level, a message can partially implement LogMessage
func (lvl Level) DefaultLevel() Level {
	return lvl
}
