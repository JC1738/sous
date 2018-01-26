package restful

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

type (
	// The MetaHandler collects common behavior for route handlers.
	MetaHandler struct {
		router        *httprouter.Router
		statusHandler *StatusMiddleware
		logging.LogSink
	}

	// ResponseWriter wraps the the http.ResponseWriter interface.
	// XXX This is a workaround for Psyringe
	ResponseWriter struct {
		http.ResponseWriter
	}

	// ExchangeLogger wraps and logs the exchange.
	ExchangeLogger struct {
		Exchanger Exchanger
		logging.LogSink
		*http.Request
		httprouter.Params
	}

	// Injector is an interface for DI systems.
	Injector interface {
		Inject(...interface{}) error
		MustInject(...interface{})
		Add(...interface{})
	}
)

// Exchange implements Exchanger on ExchangeLogger.
func (xlog *ExchangeLogger) Exchange() (data interface{}, status int) {
	logging.ReportMsg(logging.Log, logging.DebugLevel, fmt.Sprintf("Server: <- %s %s params: %v", xlog.Method, xlog.URL.String(), xlog.Params))
	data, status = xlog.Exchanger.Exchange()
	logging.ReportMsg(logging.Log, logging.DebugLevel, fmt.Sprintf("Server: -> %d: %#v", status, data))
	return
}

type loggingResponseWriter struct {
	http.ResponseWriter
	req          *http.Request
	log          logging.LogSink
	resourceName string
	start        time.Time
	statusCode   int
}

// WriteHeader implements and overrides ResponseWriter on loggingResponseWriter -
// it records the status as reported for logging.
func (lrw *loggingResponseWriter) WriteHeader(status int) {
	lrw.statusCode = status
	lrw.ResponseWriter.WriteHeader(status)
}

// Write logs the response. If ContentLength is set, we use that, otherwise we report a 0 length.
// Unfortunately, ResponseWriter's contract makes it impossible to get the true content length.
func (lrw loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)

	// ParseInt returns 0 and an syntax error if the provided string doesn't parse well.
	contentLength, _ := strconv.ParseInt(lrw.ResponseWriter.Header().Get("Content-Length"), 10, 64)
	messages.ReportServerHTTPResponse(lrw.log, lrw.req, lrw.statusCode, contentLength, lrw.resourceName, time.Now().Sub(lrw.start))

	return n, err
}

func (mh *MetaHandler) wrapResponseWriter(resName string, rq *http.Request, rw http.ResponseWriter) http.ResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: rw,
		req:            rq,
		log:            mh.LogSink,
		start:          time.Now(),
		statusCode:     http.StatusOK,
		resourceName:   resName,
	}
}

// GetHandling handles Get requests.
func (mh *MetaHandler) GetHandling(resName string, factory ExchangeFactory) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w := mh.wrapResponseWriter(resName, r, rw)
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()
		w.Header().Add("Access-Control-Allow-Origin", "*") //XXX configurable by app
		mh.renderData(status, w, r, data)
	}
}

// DeleteHandling handles Delete requests.
func (mh *MetaHandler) DeleteHandling(resName string, factory ExchangeFactory) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w := mh.wrapResponseWriter(resName, r, rw)
		h := mh.injectedHandler(factory, w, r, p)
		_, status := h.Exchange()
		mh.renderData(status, w, r, nil)
	}
}

// HeadHandling handles Head requests.
func (mh *MetaHandler) HeadHandling(resName string, factory ExchangeFactory) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w := mh.wrapResponseWriter(resName, r, rw)
		h := mh.injectedHandler(factory, w, r, p)
		_, status := h.Exchange()
		mh.writeHeaders(status, w, r, nil)
	}
}

// OptionsHandling handles Options requests.
func (mh *MetaHandler) OptionsHandling(resName string, factory ExchangeFactory) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w := mh.wrapResponseWriter(resName, r, rw)
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()

		w.Header().Add("Access-Control-Allow-Origin", r.Header.Get("Origin")) //XXX Yup: whoever was asking
		w.Header().Add("Access-Control-Max-Age", "86400")
		if methods, ok := data.([]string); ok {
			w.Header().Add("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		}
		mh.writeHeaders(status, w, r, nil)
	}
}

// PutHandling handles PUT requests.
func (mh *MetaHandler) PutHandling(resName string, factory ExchangeFactory) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w := mh.wrapResponseWriter(resName, r, rw)
		if r.Header.Get("If-Match") == "" && r.Header.Get("If-None-Match") == "" {
			mh.writeHeaders(http.StatusPreconditionRequired, w, r, "PUT requires If-Match or If-None-Match")
			return
		}

		gr := copyRequest(r)
		gr.Method = "GET"
		grez := mh.synthResponse(gr)

		if r.Header.Get("If-None-Match") == "*" && grez.StatusCode != 404 {
			mh.writeHeaders(http.StatusPreconditionFailed, w, r, "resource present for If-None-Match=*!")
			return
		}
		if etag := r.Header.Get("If-Match"); etag != "" {
			grezEtag := grez.Header.Get("Etag")
			if grezEtag != etag {
				rezBody, _ := ioutil.ReadAll(grez.Body)
				rezStr := string(rezBody)
				mh.writeHeaders(http.StatusPreconditionFailed, w, r,
					fmt.Sprintf("Etag mismatch: provided %q != existing %q\nExisting resource:\n%s",
						etag, grezEtag, rezStr))
				return
			}

			if !mh.validCanaryAttr(w, r, etag) {
				return
			}
		}
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()
		mh.renderData(status, w, r, data)
	}
}

// InstallPanicHandler installs an panic handler into the router.
func (mh *MetaHandler) InstallPanicHandler() {
	mh.router.PanicHandler = func(w http.ResponseWriter, r *http.Request, recovered interface{}) {
		//log.Print(recovered)
		mh.statusHandler.HandlePanic(w, r, recovered)
	}
}

func (mh *MetaHandler) buildLogger(h Exchanger, r *http.Request, p httprouter.Params) *ExchangeLogger {
	return &ExchangeLogger{
		LogSink:   mh.LogSink,
		Exchanger: h,
		Request:   r,
		Params:    p,
	}
}

func (mh *MetaHandler) injectedHandler(factory ExchangeFactory, w http.ResponseWriter, r *http.Request, p httprouter.Params) Exchanger {
	h := factory(w, r, p)

	return mh.buildLogger(h, r, p)
}

func (mh *MetaHandler) writeHeaders(status int, w http.ResponseWriter, r *http.Request, data interface{}) {
	mh.statusHandler.HandleResponse(status, r, w, data)
}

var etagHeader = http.CanonicalHeaderKey("Etag")
var contentTypeHeader = http.CanonicalHeaderKey("Content-Type")
var contentLengthHeader = http.CanonicalHeaderKey("Content-Length")

func (mh *MetaHandler) renderData(status int, w http.ResponseWriter, r *http.Request, data interface{}) {
	if data == nil || status >= 300 {
		mh.writeHeaders(status, w, r, data)
		return
	}

	buf := &bytes.Buffer{}
	// xxx conneg

	var etag string
	if _, got := w.Header()[etagHeader]; !got {
		digest := md5.New()
		e := json.NewEncoder(io.MultiWriter(buf, digest))
		e.Encode(data)
		etag = base64.URLEncoding.EncodeToString(digest.Sum(nil))
		w.Header().Add(etagHeader, etag)
	} else {
		e := json.NewEncoder(buf)
		e.Encode(data)
		etag = w.Header().Get(etagHeader)
	}

	if _, got := w.Header()[contentTypeHeader]; !got {
		w.Header().Add(contentTypeHeader, "application/json")
	}

	if _, got := w.Header()[contentLengthHeader]; !got {
		w.Header().Add(contentLengthHeader, fmt.Sprintf("%d", calcContentLength(buf, etag)))
	}

	mh.writeHeaders(status, w, r, data)
	if buf.Len() > 0 {
		io.Copy(w, InjectCanaryAttr(buf, etag))
	} else {
		io.Copy(w, buf)
	}
}

func emptyBody() io.ReadCloser {
	return ioutil.NopCloser(&bytes.Buffer{})
}
func copyRequest(req *http.Request) *http.Request {
	nr := &http.Request{}
	*nr = *req
	if req.URL != nil {
		nr.URL = &url.URL{}
		*nr.URL = *req.URL
	}
	nr.Body = emptyBody() //users must copy body themselves
	return nr
}

func (mh *MetaHandler) synthResponse(req *http.Request) *http.Response {
	rw := httptest.NewRecorder()
	mh.router.ServeHTTP(rw, req)
	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		StatusCode: rw.Code,
		Header:     rw.HeaderMap,
	}
	if res.StatusCode == 0 {
		res.StatusCode = 200
	}
	res.Status = http.StatusText(res.StatusCode)
	if rw.Body != nil {
		res.Body = ioutil.NopCloser(bytes.NewReader(rw.Body.Bytes()))
	}

	if trailers, ok := res.Header["Trailer"]; ok {
		res.Trailer = make(http.Header, len(trailers))
		for _, k := range trailers {
			// TODO: use http2.ValidTrailerHeader, but we can't
			// get at it easily because it's bundled into net/http
			// unexported. This is good enough for now:
			switch k {
			case "Transfer-Encoding", "Content-Length", "Trailer":
				// Ignore since forbidden by RFC 2616 14.40.
				continue
			}
			k = http.CanonicalHeaderKey(k)
			vv, ok := rw.HeaderMap[k]
			if !ok {
				continue
			}
			vv2 := make([]string, len(vv))
			copy(vv2, vv)
			res.Trailer[k] = vv2
		}
	}
	return res
}
