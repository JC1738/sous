package restful

import (
	"fmt"

	"github.com/opentable/sous/util/logging"
)

type (
	logSet interface {
		Vomitf(format string, a ...interface{})
		Debugf(format string, a ...interface{})
		Warnf(format string, a ...interface{})
	}

	silentLogSet   struct{}
	fallbackLogger struct{}
)

// PlaceholderLogger returns a log set that fulfills the restful logging
// interface - a convenience for when you don't want or need to wrap a logger
// in appropriate interface fulfillment.
func PlaceholderLogger() logSet {
	return &silentLogSet{}
}

func (sl *silentLogSet) Warnf(string, ...interface{})  {}
func (sl *silentLogSet) Debugf(string, ...interface{}) {}
func (sl *silentLogSet) Vomitf(string, ...interface{}) {}

func (l *fallbackLogger) Warnf(f string, as ...interface{})  { fmt.Printf(f+"\n", as...) }
func (l *fallbackLogger) Debugf(f string, as ...interface{}) { fmt.Printf(f+"\n", as...) }
func (l *fallbackLogger) Vomitf(f string, as ...interface{}) { fmt.Printf(f+"\n", as...) }

func (l *fallbackLogger) Child(name string) logging.LogSink {
	return l
}

func (l *fallbackLogger) LogMessage(lvl logging.Level, msg logging.LogMessage) {
	fmt.Printf("%s %#v\n", lvl, msg)
}

func (l *fallbackLogger) Metrics() logging.MetricsSink {
	panic("not implemented")
}

func (l *fallbackLogger) Console() logging.WriteDoner {
	panic("not implemented")
}

func (l *fallbackLogger) AtExit() {}
