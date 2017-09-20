package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSilentMessageError(t *testing.T) {
	log, ctrl := NewLogSinkBundle()
	reportSilentMessage(log, struct{}{})

	if assert.Len(t, ctrl.CallsTo("Console"), 1) {
		assert.Len(t, ctrl.Console.CallsTo("Write"), 1)
	}

	assert.Len(t, ctrl.CallsTo("LogMessage"), 1)
}
