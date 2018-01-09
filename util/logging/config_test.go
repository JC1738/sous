package logging

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func pangramConfig() Config {
	cfg := Config{}
	cfg.Basic.Level = "Critical"
	cfg.Kafka.Enabled = true
	cfg.Kafka.DefaultLevel = ""
	cfg.Kafka.Topic = "logging"
	cfg.Kafka.BrokerList = "127.0.0.1:5000,127.0.0.2:5000"
	cfg.Graphite.Enabled = true
	cfg.Graphite.Server = "graphite.example.com"
	return cfg
}

func TestLoggingConfig(t *testing.T) {
	t.Run("level", func(t *testing.T) {
		cfg := pangramConfig()
		assert.Equal(t, cfg.GetBasicLevel(), CriticalLevel)
		assert.Equal(t, cfg.getLogrusLevel(), logrus.ErrorLevel)
	})

	t.Run("kafka config", func(t *testing.T) {
		cfg := pangramConfig()
		assert.Equal(t, cfg.getBrokers(), []string{"127.0.0.1:5000", "127.0.0.2:5000"})
	})

	t.Run("empty kafka brokers", func(t *testing.T) {
		cfg := pangramConfig()
		cfg.Kafka.Enabled = true
		cfg.Kafka.BrokerList = ""
		assert.Error(t, cfg.validateKafka(), "Error should have occurred")
	})

	t.Run("graphite server", func(t *testing.T) {
		cfg := pangramConfig()
		assert.Equal(t, cfg.getGraphiteServer(), "graphite.example.com:2003")
	})

	t.Run("Equal", func(t *testing.T) {
		cfg := pangramConfig()
		other := Config{}

		assert.False(t, cfg.Equal(other))
		assert.True(t, cfg.Equal(pangramConfig()))
	})
}
