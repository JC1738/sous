package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// GDMResource is the resource for the GDM
	GDMResource struct {
		userExtractor
		context ComponentLocator
	}

	// GETGDMHandler is an injectable request handler
	GETGDMHandler struct {
		logging.LogSink
		GDM      *sous.State
		RzWriter http.ResponseWriter
	}

	// PUTGDMHandler is an injectable request handler
	PUTGDMHandler struct {
		*http.Request
		logging.LogSink
		GDM          *sous.State
		StateManager sous.StateManager
		User         ClientUser
	}
)

func newGDMResource(ctx ComponentLocator) *GDMResource {
	return &GDMResource{context: ctx}
}

// Get implements Getable on GDMResource
func (gr *GDMResource) Get(writer http.ResponseWriter, _ *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETGDMHandler{
		LogSink:  gr.context.LogSink,
		GDM:      gr.context.liveState(),
		RzWriter: writer,
	}
}

// Exchange implements the Handler interface
func (h *GETGDMHandler) Exchange() (interface{}, int) {
	reportDebugHandleGDMMessage(fmt.Sprintf("Get GDM Handler Exchange with GDM: %v", h.GDM), nil, nil, h.LogSink)

	data := GDMWrapper{Deployments: make([]*sous.Deployment, 0)}
	deps, err := h.GDM.Deployments()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	keys := sous.DeploymentIDSlice(deps.Keys())
	sort.Sort(keys)

	for _, k := range keys {
		d, has := deps.Get(k)
		if !has {
			return "Error serializing GDM", http.StatusInternalServerError
		}
		data.Deployments = append(data.Deployments, d)
	}
	etag, _ := h.GDM.GetEtag()
	h.RzWriter.Header().Set("Etag", etag)

	return data, http.StatusOK
}

// Put implements Putable on GDMResource
func (gr *GDMResource) Put(_ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &PUTGDMHandler{
		Request:      req,
		LogSink:      gr.context.LogSink,
		GDM:          gr.context.liveState(),
		StateManager: gr.context.StateManager,
		User:         gr.GetUser(req),
	}
}

// Exchange implements the Handler interface
func (h *PUTGDMHandler) Exchange() (interface{}, int) {
	reportDebugHandleGDMMessage(fmt.Sprintf("Put GDM Handler Exchange with GDM: %v", h.GDM), nil, nil, h.LogSink)

	data := GDMWrapper{}
	dec := json.NewDecoder(h.Request.Body)
	dec.Decode(&data)
	deps := sous.NewDeployments(data.Deployments...)

	state, err := h.StateManager.ReadState()
	if err != nil {
		msg := "Error loading state from storage"
		reportHandleGDMMessage(msg, nil, err, h.LogSink)
		return msg, http.StatusInternalServerError
	}

	state.Manifests, err = deps.PutbackManifests(state.Defs, state.Manifests)
	if err != nil {
		msg := "Error getting state"
		reportHandleGDMMessage(msg, nil, err, h.LogSink)
		return msg, http.StatusConflict
	}

	flaws := state.Validate()
	if len(flaws) > 0 {
		msg := "Invalid GDM"
		reportHandleGDMMessage(msg, flaws, nil, h.LogSink)
		return msg, http.StatusBadRequest
	}

	if _, got := h.Header["Etag"]; got {
		state.SetEtag(h.Header.Get("Etag"))
	}

	if err := h.StateManager.WriteState(state, sous.User(h.User)); err != nil {
		msg := "Error committing state"
		reportHandleGDMMessage(msg, flaws, err, h.LogSink)
		return msg, http.StatusInternalServerError
	}

	return "", http.StatusNoContent
}

type handleGDMMessage struct {
	logging.CallerInfo
	msg   string
	flaws []sous.Flaw
	err   error
	debug bool
}

func reportDebugHandleGDMMessage(msg string, flaws []sous.Flaw, err error, log logging.LogSink) {
	reportHandleGDMMessage(msg, flaws, err, log, true)
}

func reportHandleGDMMessage(msg string, flaws []sous.Flaw, err error, log logging.LogSink, debug ...bool) {

	isDebug := false
	if len(debug) > 0 {
		isDebug = debug[0]
	}

	msgLog := handleGDMMessage{
		msg:        msg,
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		err:        err,
		flaws:      flaws,
		debug:      isDebug,
	}
	logging.Deliver(msgLog, log)
}

func (msg handleGDMMessage) WriteToConsole(console io.Writer) {
	fmt.Fprintf(console, "%s\n", msg.composeMsg())
}

func (msg handleGDMMessage) DefaultLevel() logging.Level {
	level := logging.WarningLevel

	if msg.debug == true {
		level = logging.DebugLevel
	}

	return level
}

func (msg handleGDMMessage) Message() string {
	return msg.composeMsg()
}

func (msg handleGDMMessage) returnFlawMsg() string {
	var msgFlaws string
	for _, flaw := range msg.flaws {
		joined := []string{msgFlaws, fmt.Sprintf("%v", flaw)}
		msgFlaws = strings.Join(joined, " ")
	}
	return msgFlaws
}

func (msg handleGDMMessage) composeMsg() string {
	errMsg := "nil"
	if msg.err != nil {
		errMsg = msg.err.Error()
	}
	flaws := msg.returnFlawMsg()
	if flaws == "" {
		flaws = "nil"
	}
	return fmt.Sprintf("Handle GDM Message %s: flaws {%s}, error {%s}", msg.msg, msg.returnFlawMsg(), errMsg)
}

func (msg handleGDMMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-generic-v1")

	flaws := msg.returnFlawMsg()

	if flaws != "" {
		f("flaws", flaws)
	}

	if msg.err != nil {
		f("error", msg.err.Error())
	}
	msg.CallerInfo.EachField(f)
}
