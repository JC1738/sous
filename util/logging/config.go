package logging

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Config captures outside configuration for a root LogSet
type Config struct {
	Basic struct {
		Level          string `env:"SOUS_LOGGING_LEVEL"`
		DisableConsole bool
	}
	Kafka struct {
		Enabled      bool
		DefaultLevel string `env:"SOUS_KAFKA_LOG_LEVEL"`
		Topic        string `env:"SOUS_KAFKA_TOPIC"`
		Brokers      []string
		BrokerList   string `env:"SOUS_KAFKA_BROKERS"`
	}
	Graphite struct {
		Enabled bool
		Server  string `env:"SOUS_GRAPHITE_SERVER"`
	}
}

// Equal compares values of Config.
func (cfg Config) Equal(other Config) bool {
	if cfg.Kafka.Enabled != other.Kafka.Enabled {
		return false
	}
	if cfg.Kafka.Enabled {
		if cfg.Kafka.DefaultLevel != other.Kafka.DefaultLevel ||
			cfg.Kafka.Topic != other.Kafka.Topic {
			return false
		}
		lbrokers := cfg.getBrokers()
		rbrokers := cfg.getBrokers()
		if len(lbrokers) != len(rbrokers) {
			return false
		}

		// order?
		for i := len(lbrokers) - 1; i >= 0; i-- {
			if lbrokers[i] != rbrokers[i] {
				return false
			}
		}
	}
	if cfg.Graphite.Enabled != other.Graphite.Enabled {
		return false
	}

	if cfg.Graphite.Enabled && (cfg.Graphite.Server != other.Graphite.Server) {
		return false
	}

	return true
}

// GetBasicLevel gets the basic level of logging for this configuration.
// that is, the level of messages that would be logged that should be emitted to the console
// this is separate from messages that are designed for human consumption
func (cfg Config) GetBasicLevel() Level {
	if cfg.Basic.Level == "" {
		return CriticalLevel // console output should be via ConsoleMessages, not logging.
	}
	return levelFromString(cfg.Basic.Level)
}

func (cfg Config) getLogrusLevel() logrus.Level {
	switch cfg.GetBasicLevel() {
	default:
		return logrus.WarnLevel
	case DebugLevel, ExtraDebug1Level, ExtremeLevel:
		return logrus.DebugLevel
	case InformationLevel:
		return logrus.InfoLevel
	case CriticalLevel:
		return logrus.ErrorLevel
	}
}

func (cfg Config) getBrokers() []string {
	if len(cfg.Kafka.Brokers) != 0 {
		return cfg.Kafka.Brokers
	}
	return strings.Split(cfg.Kafka.BrokerList, ",")
}

func (cfg Config) getKafkaLevels() []logrus.Level {
	level := levelFromString(cfg.Kafka.DefaultLevel)
	kafkaLevels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
	}

	if level >= WarningLevel {
		kafkaLevels = append(kafkaLevels, logrus.WarnLevel)
	}

	if level >= InformationLevel {
		kafkaLevels = append(kafkaLevels, logrus.InfoLevel)
	}

	if level >= DebugLevel {
		kafkaLevels = append(kafkaLevels, logrus.InfoLevel)
	}

	return kafkaLevels
}

func (cfg Config) useKafka() bool {
	return cfg.Kafka.Enabled
}

// Validate asserts the validity of the logging configuration
func (cfg Config) Validate() error {
	if err := cfg.validateKafka(); cfg.useKafka() && err != nil {
		return err
	}
	return nil
}

func (cfg Config) validateKafka() error {
	switch cfg.Kafka.DefaultLevel {
	default:
		return errors.Errorf("default Kafka log level unrecognized: configured as %q", cfg.Kafka.DefaultLevel)
	case "":
		return errors.Errorf("default Kafka log level empty")
	case "Critical", "Warning", "Information", "Debug", "ExtraDebug1":
	}

	switch {
	default:
		return nil
	case len(cfg.getBrokers()) == 0:
		return errors.Errorf("no brokers specified for kafka")
	case cfg.Kafka.Topic == "":
		return errors.Errorf("no Kafka topic configured")
	}
}
