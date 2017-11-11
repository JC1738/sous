package logging

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/samsalisbury/semv"
	"github.com/sirupsen/logrus"
)

type (
	// LogSet is the stopgap for a decent injectable logger
	LogSet struct {
		// xxx remove these as phase 1 of completing transition
		Debug  logwrapper
		Info   logwrapper
		Warn   logwrapper
		Notice logwrapper
		Vomit  logwrapper

		level   Level
		name    string
		appRole string

		metrics metrics.Registry
		*dumpBundle
	}

	// ugh - I don't know what else to call this though
	dumpBundle struct {
		appIdent        *applicationID
		context         context.Context
		err, defaultErr io.Writer
		logrus          *logrus.Logger
		liveConfig      *Config
		kafkaSink       *kafkaSink
		graphiteCancel  func()
	}

	// A temporary type until we can stop using the LogSet loggers directly
	// XXX remove and fix accesses to Debug, Info, etc. to be Debugf etc
	logwrapper func(string, ...interface{})
)

var (
	// Log collects various loggers to use for different levels of logging
	// XXX A goal should be to remove this global, and instead inject logging where we need it.
	//
	// Notice that the global LotSet doesn't have metrics available - when you
	// want metrics in a component, you need to add an injected LogSet. c.f.
	// ext/docker/image_mapping.go
	Log = func() LogSet {
		//return *(NewLogSet(semv.MustParse("0.0.0"), "sous.global", "", os.Stderr))
		return *(SilentLogSet()) // we'll configure it later
	}()
)

func (w logwrapper) Printf(f string, vs ...interface{}) {
	w(f, vs...)
}

func (w logwrapper) Print(vs ...interface{}) {
	w(fmt.Sprint(vs...))
}

func (w logwrapper) Println(vs ...interface{}) {
	w(fmt.Sprint(vs...))
}

// SilentLogSet returns a logset that discards everything by default
func SilentLogSet() *LogSet {
	ls := NewLogSet(semv.MustParse("0.0.0"), "", "", os.Stderr)
	ls.BeQuiet()
	return ls
}

// NewLogSet builds a new Logset that feeds to the listed writers
func NewLogSet(version semv.Version, name string, role string, err io.Writer) *LogSet {
	// logrus uses a pool for entries, which means we probably really should only have one.
	// this means that output configuration and level limiting is global to the logset and
	// its children.
	lgrs := logrus.New()
	lgrs.Out = err

	bundle := newdb(version, err, lgrs)

	ls := newls(name, role, CriticalLevel, bundle) //level constrains Kafka output
	ls.imposeLevel()

	// use sous.<env>.<region>.*, he said
	// sous. comes from the GraphiteConfig "Prefix" field.
	// <env>.<region> from metricsScope()
	ls.metrics = metrics.NewPrefixedRegistry(bundle.appIdent.metricsScope() + ".")
	return ls
}

// Child produces a child logset, namespaced under "name".
func (ls LogSet) Child(name string) LogSink {
	child := newls(ls.name+"."+name, ls.appRole, ls.level, ls.dumpBundle)
	child.metrics = metrics.NewPrefixedChildRegistry(ls.metrics, name+".")
	return child
}

func getEnvHash() map[string]string {
	h := map[string]string{}
	get := func(n string) {
		if v, has := os.LookupEnv(n); has {
			h[n] = v
		}
	}
	get("OT_ENV")
	get("OT_ENV_TYPE")
	get("OT_ENV_LOCATION")
	get("TASK_ID")
	get("INSTANCE_NO")
	return h
}

func newdb(vrsn semv.Version, err io.Writer, lgrs *logrus.Logger) *dumpBundle {
	env := getEnvHash()

	return &dumpBundle{
		appIdent:   collectAppID(vrsn, env),
		context:    context.Background(),
		err:        err,
		defaultErr: err,
		logrus:     lgrs,
	}
}

func (db *dumpBundle) replaceKafka(sink *kafkaSink) {
	var old *kafkaSink
	old, db.kafkaSink = db.kafkaSink, sink
	if old != nil {
		old.closedown()
	}
}

func (db *dumpBundle) sendToKafka(lvl Level, entry *logrus.Entry) error {
	if db.kafkaSink == nil {
		return nil
	}
	return db.kafkaSink.send(lvl, entry)
}

func newls(name string, role string, level Level, bundle *dumpBundle) *LogSet {
	ls := &LogSet{
		name:       name,
		appRole:    role,
		level:      level,
		dumpBundle: bundle,
	}

	ls.Warn = logwrapper(func(f string, as ...interface{}) { ls.Warnf(f, as) })
	ls.Notice = ls.Warn
	ls.Info = ls.Warn
	ls.Debug = logwrapper(func(f string, as ...interface{}) { ls.Debugf(f, as) })
	ls.Vomit = logwrapper(func(f string, as ...interface{}) { ls.Vomitf(f, as) })

	return ls
}

// Configure allows an existing LogSet to change its settings.
func (ls *LogSet) Configure(cfg Config) error {
	err := ls.configureKafka(cfg)
	if err != nil {
		return err
	}

	err = ls.configureGraphite(cfg)
	if err != nil {
		return err
	}

	ls.logrus.SetLevel(cfg.getLogrusLevel())

	if cfg.Basic.DisableConsole {
		ls.dumpBundle.err = ioutil.Discard
	} else {
		ls.dumpBundle.err = ls.dumpBundle.defaultErr
	}

	ls.liveConfig = &cfg
	return nil
}

// AtExit implements part of LogSink on LogSet
func (ls LogSet) AtExit() {
	if ls.dumpBundle.kafkaSink != nil {
		ls.dumpBundle.kafkaSink.closedown()
	}
}

func logrusFormatter() logrus.Formatter {
	return &logrus.JSONFormatter{
		DisableTimestamp: true, //we capture the timestamp when message created

		//our names for these fields
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyMsg:   "call-stack-message",
			logrus.FieldKeyLevel: "severity",
		},
	}
}

func (ls LogSet) configureKafka(cfg Config) error {
	if !cfg.useKafka() {
		reportKafkaConfig(nil, cfg, ls)
		return nil
	}

	sink, err := newKafkaSink("kafkahook",
		cfg.getKafkaLevel(),
		logrusFormatter(),
		cfg.getBrokers(),
		cfg.Kafka.Topic,
		false)

	// One cause of errors: can't reach any brokers
	// c.f. https://github.com/Shopify/sarama/blob/master/client.go#L114
	if err != nil {
		return err
	}
	reportKafkaConfig(sink, cfg, ls)

	ls.dumpBundle.replaceKafka(sink)

	return nil
}

func (ls LogSet) configureGraphite(cfg Config) error {
	var gCfg *graphite.Config

	if cfg.useGraphite() {
		addr, err := net.ResolveTCPAddr("tcp", cfg.getGraphiteServer())
		if err != nil {
			return err
		}

		gCfg = &graphite.Config{
			Addr:          addr,
			Registry:      ls.metrics,
			FlushInterval: 30 * time.Second,
			DurationUnit:  time.Nanosecond,
			Prefix:        "sous",
			Percentiles:   []float64{0.5, 0.75, 0.95, 0.99, 0.999},
		}

	}
	reportGraphiteConfig(gCfg, ls)

	gCtx, cancel := context.WithCancel(ls.context)

	if ls.graphiteCancel != nil {
		ls.graphiteCancel()
	}

	ls.graphiteCancel = cancel
	go metricsLoop(gCtx, ls, gCfg)

	return nil
}

func metricsLoop(ctx context.Context, ls LogSet, cfg *graphite.Config) {
	interval := time.Second * 30
	if cfg != nil {
		interval = cfg.FlushInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// TODO: metrics observation goes here
			if cfg != nil {
				if err := graphite.Once(*cfg); err != nil {
					reportGraphiteError(ls, err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// Metrics returns a MetricsSink, which can receive various metrics related method calls. (c.f)
// LogSet.Metrics returns itself -
// xxx quickie for providing metricssink
func (ls LogSet) Metrics() MetricsSink {
	return ls
}

// Done signals that the LogSet (as a MetricsSink) is done being used -
// LogSet's current implementation treats this as a no-op but c.f. MetricsSink.
// xxx noop until extracted a metrics sink
func (ls LogSet) Done() {
}

// Console implements LogSink on LogSet
func (ls LogSet) Console() WriteDoner {
	return nopDoner(ls.err)
}

// xxx phase 2 of complete transition: remove these methods in favor of specific messages

// Vomitf logs a message at ExtraDebug1Level.
func (ls LogSet) Vomitf(f string, as ...interface{}) {
	m := NewGenericMsg(ExtraDebug1Level, fmt.Sprintf(f, as...), nil)
	m.ExcludeMe()
	Deliver(m, ls)
}

// Debugf logs a message a DebugLevel.
func (ls LogSet) Debugf(f string, as ...interface{}) {
	m := NewGenericMsg(DebugLevel, fmt.Sprintf(f, as...), nil)
	m.ExcludeMe()
	Deliver(m, ls)
}

// Warnf logs a message at WarningLevel.
func (ls LogSet) Warnf(f string, as ...interface{}) {
	m := NewGenericMsg(WarningLevel, fmt.Sprintf(f, as...), nil)
	m.ExcludeMe()
	Deliver(m, ls)
}

func (ls LogSet) imposeLevel() {
	ls.logrus.SetLevel(logrus.ErrorLevel)

	if ls.level >= 1 {
		ls.logrus.SetLevel(logrus.WarnLevel)
	}

	if ls.level >= 2 {
		ls.logrus.SetLevel(logrus.DebugLevel)
	}

	if ls.level >= 3 {
		ls.logrus.SetLevel(logrus.DebugLevel)
	}
}

// BeQuiet gets the LogSet to discard all its output
func (ls LogSet) BeQuiet() {
	ls.level = 0
	ls.imposeLevel()
}

// BeTerse gets the LogSet to print debugging output
func (ls LogSet) BeTerse() {
	ls.level = 1
	ls.imposeLevel()
}

// BeHelpful gets the LogSet to print debugging output
func (ls LogSet) BeHelpful() {
	ls.level = 2
	ls.imposeLevel()
}

// BeChatty gets the LogSet to print all its output - useful for temporary debugging
func (ls LogSet) BeChatty() {
	ls.level = 3
	ls.imposeLevel()
}
