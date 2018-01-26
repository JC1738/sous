package server

import (
	"net/http/httptest"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
)

func TestHandlesGDMGet(t *testing.T) {
	assert := assert.New(t)

	w := httptest.NewRecorder()
	//etag := "swordfish"

	th := &GETGDMHandler{
		RzWriter: &restful.ResponseWriter{w},
		LogSink:  logging.Log,
		GDM:      sous.NewState(),
	}

	data, status := th.Exchange()
	//assert.Equal(w.Header().Get("Etag"), etag)
	assert.Equal(status, 200)
	assert.Len(data.(GDMWrapper).Deployments, 0)
}

func TestReturnFlawMsg_nil_flaws(t *testing.T) {
	assert := assert.New(t)

	hmsg := handleGDMMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		msg:        "test",
		flaws:      nil,
		err:        nil,
	}

	//   a.NotPanics(func(){
	//     RemainCalm()
	//   },

	assert.NotPanics(func() {
		hmsg.returnFlawMsg()
	}, "Calling returnFlawMsg should not panic with flaws is nil")

}

func TestReturnFlawMsg(t *testing.T) {

	empty := make(sous.Resources)

	flaws := empty.Validate()
	assert.Len(t, flaws, 3)

	hmsg := handleGDMMessage{
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		msg:        "test",
		flaws:      flaws,
		err:        nil,
	}

	flawsMsg := hmsg.returnFlawMsg()

	assert.Contains(t, flawsMsg, "Missing resource")

}
