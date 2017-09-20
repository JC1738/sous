package messages

import (
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportCHResponseFields(t *testing.T) {
	logger, control := logging.NewLogSinkBundle()
	msg := newClientHTTPResponse("GET", "http://example.com/api?a=a", 200, 0, 123, time.Millisecond*30)
	logging.Deliver(msg, logger)

	assert.Len(t, control.Metrics.CallsTo("UpdateTimer"), 1)
	logCalls := control.CallsTo("LogMessage")
	require.Len(t, logCalls, 1)
	assert.Equal(t, logCalls[0].PassedArgs().Get(0), logging.InformationLevel)
	message := logCalls[0].PassedArgs().Get(1).(logging.LogMessage)
	actualFields := map[string]interface{}{}
	message.EachField(func(name string, value interface{}) {
		assert.NotContains(t, actualFields, name) //don't clobber a field
		actualFields[name] = value
	})

	/*
		"line":610,
		"function":"testing.tRunner",
		"file":"/nix/store/br0ngwcjyffc7d060spw44wah1hdnlwn-go-1.7.4/share/go/src/testing/testing.go",
		"time":logging.callTime{sec:63639633602, nsec:854240181, loc:(*time.Location)(0x8f3780)},
	*/

	variableFields := []string{"line", "function", "file", "@timestamp", "thread-name"}
	for _, f := range variableFields {
		assert.Contains(t, actualFields, f)
		delete(actualFields, f)
	}

	assert.Equal(t, map[string]interface{}{
		"@loglov3-otl":  "http-v1",
		"incoming":      false,
		"method":        "GET",
		"url":           "http://example.com/api?a=a",
		"server":        "example.com",
		"path":          "/api",
		"querystring":   "a=a",
		"duration":      time.Duration(30000000),
		"body-size":     int64(0),
		"response-size": int64(123),
		"status":        200,
	}, actualFields)

}
