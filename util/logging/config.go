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

// Equal tests the equality of two configs.
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
// that is, the level of messages that would be logged that should be emitted to
// the console this is separate from messages that are designed for human
// consumption
func (cfg Config) GetBasicLevel() Level {
	if cfg.Basic.Level == "" {
		// Console output should be via ConsoleMessages, not logging.
		return CriticalLevel
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

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func (cfg Config) getBrokers() []string {
	if len(cfg.Kafka.Brokers) != 0 {
		return deleteEmpty(cfg.Kafka.Brokers)
	}

	return deleteEmpty(strings.Split(cfg.Kafka.BrokerList, ","))
}

func (cfg Config) getKafkaLevel() Level {
	return levelFromString(cfg.Kafka.DefaultLevel)
}

func (cfg Config) getGraphiteServer() string {
	if strings.Index(cfg.Graphite.Server, ":") != -1 {
		return cfg.Graphite.Server
	}
	return strings.Join([]string{cfg.Graphite.Server, "2003"}, ":")
}

func (cfg Config) useKafka() bool {
	return cfg.Kafka.Enabled
}

func (cfg Config) useGraphite() bool {
	return cfg.Graphite.Enabled
}

// Validate asserts the validity of the logging configuration
func (cfg Config) Validate() error {
	if err := cfg.validateGraphite(); cfg.useGraphite() && err != nil {
		return err
	}
	if err := cfg.validateKafka(); cfg.useKafka() && err != nil {
		return err
	}
	return nil
}

func (cfg Config) validateGraphite() error {
	if cfg.Graphite.Server == "" {
		return errors.New("no graphite server address provided")
	}
	return nil
}

func (cfg Config) validateKafka() error {

	switch {
	default:
		return nil
	case len(cfg.getBrokers()) == 0:
		return errors.Errorf("no brokers specified for kafka")
	case cfg.Kafka.Topic == "":
		return errors.Errorf("no Kafka topic configured")
	}
}
