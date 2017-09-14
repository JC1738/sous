package messages

import (
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportCHResponseFields(t *testing.T) {
	logger, control := logging.NewLogSinkSpy()
	ReportClientHTTPResponseFields(logger, "GET", "http://example.com", "/api", map[string]string{"a": "a"}, 200, time.Millisecond*30)

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
		"@loglov3-otl": "sous-client-http-response-v1",
		"method":       "GET",
		"server":       "http://example.com",
		"parms":        map[string]string{"a": "a"},
		"path":         "/api",
		"dur":          time.Duration(30000000),
		"status":       200,
	}, actualFields)

}