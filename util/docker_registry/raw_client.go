package docker_registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/context"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

/*

	Nota bene: this started as a direct copy registry/client/registry.go from
	docker/distribution That code didn't return enough information about the HTTP
	response to make use of etags or other caching.  Code from this file is being
	absorbed into sous/util/docker_registry/client.go - pieces remain here mostly
	for easy referece and as a source for continuing absorbtion

*/

type httpClient struct {
	http *http.Client
	log  logging.LogSink
}

// Do wraps http.Client.Do, and logs the response
func (c *httpClient) Do(resourceName string, req *http.Request) (*http.Response, error) {
	start := time.Now()
	res, err := c.http.Do(req)
	messages.ReportClientHTTPResponse(c.log, res, resourceName, time.Now().Sub(start))
	return res, err
}

// Get wraps http.Client.Get, and logs the response
func (c *httpClient) Get(resourceName string, url string) (resp *http.Response, err error) {
	start := time.Now()
	res, err := c.http.Get(url)
	messages.ReportClientHTTPResponse(c.log, res, resourceName, time.Now().Sub(start))
	return res, err
}

// Head wraps http.Client.Head, and logs the response
func (c *httpClient) Head(resourceName string, url string) (resp *http.Response, err error) {
	start := time.Now()
	res, err := c.http.Head(url)
	messages.ReportClientHTTPResponse(c.log, res, resourceName, time.Now().Sub(start))
	return res, err
}

// checkHTTPRedirect is a callback that can manipulate redirected HTTP
// requests. It is used to preserve Accept and Range headers.
func checkHTTPRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}

	if len(via) > 0 {
		for headerName, headerVals := range via[0].Header {
			if headerName != "Accept" && headerName != "Range" {
				continue
			}
			for _, val := range headerVals {
				// Don't add to redirected request if redirected
				// request already has a header with the same
				// name and value.
				hasValue := false
				for _, existingVal := range req.Header[headerName] {
					if existingVal == val {
						hasValue = true
						break
					}
				}
				if !hasValue {
					req.Header.Add(headerName, val)
				}
			}
		}
	}

	return nil
}

// NewRegistry creates a registry namespace which can be used to get a listing of repositories
func NewRegistryOld(ctx context.Context, baseURL string, transport http.RoundTripper) (*oldRegistry, error) {
	ub, err := v2.NewURLBuilderFromString(baseURL, false)
	if err != nil {
		return nil, err
	}

	client := httpClient{
		http: &http.Client{
			Transport:     transport,
			Timeout:       1 * time.Minute,
			CheckRedirect: checkHTTPRedirect,
		},
	}

	return &oldRegistry{
		client:  client,
		ub:      ub,
		context: ctx,
	}, nil
}

// Registry provides an interface for calling Repositories, which returns a catalog of repositories.
//
// This is (for the moment) a copy of the client from docker/distribution
// We need the actual HTTP responses in order to usefully get Etags. There are
// parts of the code that depend on private implementations in the docker code,
// but we don't need to features. At some point this likely becomes our own
// private Docker client, so don't worry about keeping parity with the original
type oldRegistry struct {
	client  httpClient
	ub      *v2.URLBuilder
	context context.Context
}

// Repositories returns a lexigraphically sorted catalog given a base URL.  The 'entries' slice will be filled up to the size
// of the slice, starting at the value provided in 'last'.  The number of entries will be returned along with io.EOF if there
// are no more entries
func (r *oldRegistry) Repositories(ctx context.Context, entries []string, last string) (int, error) {
	var numFilled int
	var returnErr error

	values := buildCatalogValues(len(entries), last)
	u, err := r.ub.BuildCatalogURL(values)
	if err != nil {
		return 0, err
	}

	resp, err := r.client.Get("docker.repositories", u)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if client.SuccessStatus(resp.StatusCode) {
		var ctlg struct {
			Repositories []string `json:"repositories"`
		}
		decoder := json.NewDecoder(resp.Body)

		if err := decoder.Decode(&ctlg); err != nil {
			return 0, err
		}

		for cnt := range ctlg.Repositories {
			entries[cnt] = ctlg.Repositories[cnt]
		}
		numFilled = len(ctlg.Repositories)

		link := resp.Header.Get("Link")
		if link == "" {
			returnErr = io.EOF
		}
	} else {
		return 0, client.HandleErrorResponse(resp)
	}

	return numFilled, returnErr
}

func (r *oldRegistry) Blobs(ctx context.Context) distribution.BlobStore {
	return nil
}

/*
func (r *repository) Blobs(ctx context.Context) distribution.BlobStore {
	statter := &blobStatter{
		name:   r.name,
		ub:     r.ub,
		client: r.client,
	}
	return &blobs{
		name:    r.name,
		ub:      r.ub,
		client:  r.client,
		statter: cache.NewCachedBlobStatter(memory.NewInMemoryBlobDescriptorCacheProvider(), statter),
	}
}
*/

func descriptorFromResponse(response *http.Response) (distribution.Descriptor, error) {
	desc := distribution.Descriptor{}
	headers := response.Header

	ctHeader := headers.Get("Content-Type")
	if ctHeader == "" {
		return distribution.Descriptor{}, errors.New("missing or empty Content-Type header")
	}
	desc.MediaType = ctHeader

	digestHeader := headers.Get("Docker-Content-Digest")
	if digestHeader == "" {
		bytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return distribution.Descriptor{}, err
		}
		_, desc, err := distribution.UnmarshalManifest(ctHeader, bytes)
		if err != nil {
			return distribution.Descriptor{}, err
		}
		return desc, nil
	}

	dgst, err := digest.ParseDigest(digestHeader)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	desc.Digest = dgst

	lengthHeader := headers.Get("Content-Length")
	if lengthHeader == "" {
		return distribution.Descriptor{}, errors.New("missing or empty Content-Length header")
	}
	length, err := strconv.ParseInt(lengthHeader, 10, 64)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	desc.Size = length

	return desc, nil

}

func (r *oldRegistry) Tags(ctx context.Context) *tags {
	return &tags{
		client: r.client,
		ub:     r.ub,
	}
}

// tags implements remote tagging operations.
type tags struct {
	client  httpClient
	ub      *v2.URLBuilder
	context context.Context
	name    reference.Named
}

// Get issues a HEAD request for a Manifest against its named endpoint in order
// to construct a descriptor for the tag.  If the registry doesn't support HEADing
// a manifest, fallback to GET.
func (t *tags) Get(ctx context.Context, tag string) (distribution.Descriptor, error) {
	ref, err := reference.WithTag(t.name, tag)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	u, err := t.ub.BuildManifestURL(ref)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	req, err := http.NewRequest("HEAD", u, nil)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	for _, t := range distribution.ManifestMediaTypes() {
		req.Header.Add("Accept", t)
	}

	var attempts int
	resp, err := t.client.Do("docker.tag", req)
check:
	if err != nil {
		return distribution.Descriptor{}, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 400:
		return descriptorFromResponse(resp)
	case resp.StatusCode == http.StatusMethodNotAllowed:
		req, err = http.NewRequest("GET", u, nil)
		if err != nil {
			return distribution.Descriptor{}, err
		}

		for _, t := range distribution.ManifestMediaTypes() {
			req.Header.Add("Accept", t)
		}

		resp, err = t.client.Do("docker.tag", req)
		attempts++
		if attempts > 1 {
			return distribution.Descriptor{}, err
		}
		goto check
	default:
		return distribution.Descriptor{}, client.HandleErrorResponse(resp)
	}
}

func (t *tags) Lookup(ctx context.Context, digest distribution.Descriptor) ([]string, error) {
	panic("not implemented")
}

func (t *tags) Tag(ctx context.Context, tag string, desc distribution.Descriptor) error {
	panic("not implemented")
}

func (t *tags) Untag(ctx context.Context, tag string) error {
	panic("not implemented")
}

/*
func (ms *manifests) Exists(ctx context.Context, dgst digest.Digest) (bool, error) {
	ref, err := reference.WithDigest(ms.name, dgst)
	if err != nil {
		return false, err
	}
	u, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return false, err
	}

	resp, err := ms.client.Head(u)
	if err != nil {
		return false, err
	}

	if client.SuccessStatus(resp.StatusCode) {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, client.HandleErrorResponse(resp)
}

// Put puts a manifest.  A tag can be specified using an options parameter which uses some shared state to hold the
// tag name in order to build the correct upload URL.
func (ms *manifests) Put(ctx context.Context, m distribution.Manifest, options ...ManifestServiceOption) (digest.Digest, error) {
	ref := ms.name
	var tagged bool

	for _, option := range options {
		if opt, ok := option.(WithTagOption); ok {
			var err error
			ref, err = reference.WithTag(ref, opt.Tag)
			if err != nil {
				return "", err
			}
			tagged = true
		} else {
			err := option.Apply(ms)
			if err != nil {
				return "", err
			}
		}
	}
	mediaType, p, err := m.Payload()
	if err != nil {
		return "", err
	}

	if !tagged {
		// generate a canonical digest and Put by digest
		_, d, err := distribution.UnmarshalManifest(mediaType, p)
		if err != nil {
			return "", err
		}
		ref, err = reference.WithDigest(ref, d.Digest)
		if err != nil {
			return "", err
		}
	}

	manifestURL, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return "", err
	}

	putRequest, err := http.NewRequest("PUT", manifestURL, bytes.NewReader(p))
	if err != nil {
		return "", err
	}

	putRequest.Header.Set("Content-Type", mediaType)

	resp, err := ms.client.Do(putRequest)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if client.SuccessStatus(resp.StatusCode) {
		dgstHeader := resp.Header.Get("Docker-Content-Digest")
		dgst, err := digest.ParseDigest(dgstHeader)
		if err != nil {
			return "", err
		}

		return dgst, nil
	}

	return "", client.HandleErrorResponse(resp)
}

func (ms *manifests) Delete(ctx context.Context, dgst digest.Digest) error {
	ref, err := reference.WithDigest(ms.name, dgst)
	if err != nil {
		return err
	}
	u, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	resp, err := ms.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if client.SuccessStatus(resp.StatusCode) {
		return nil
	}
	return client.HandleErrorResponse(resp)
}

// todo(richardscothern): Restore interface and implementation with merge of #1050
/*func (ms *manifests) Enumerate(ctx context.Context, manifests []distribution.Manifest, last distribution.Manifest) (n int, err error) {
	panic("not supported")
}*/

type blobs struct {
	name   reference.Named
	ub     *v2.URLBuilder
	client httpClient

	statter distribution.BlobDescriptorService
	distribution.BlobDeleter
}

func sanitizeLocation(location, base string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	locationURL, err := url.Parse(location)
	if err != nil {
		return "", err
	}

	return baseURL.ResolveReference(locationURL).String(), nil
}

func (bs *blobs) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	return bs.statter.Stat(ctx, dgst)

}

func (bs *blobs) Get(ctx context.Context, dgst digest.Digest) ([]byte, error) {
	reader, err := bs.Open(ctx, dgst)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (bs *blobs) Open(ctx context.Context, dgst digest.Digest) (distribution.ReadSeekCloser, error) {
	ref, err := reference.WithDigest(bs.name, dgst)
	if err != nil {
		return nil, err
	}
	blobURL, err := bs.ub.BuildBlobURL(ref)
	if err != nil {
		return nil, err
	}

	return transport.NewHTTPReadSeeker(bs.client.http, blobURL,
		func(resp *http.Response) error {
			if resp.StatusCode == http.StatusNotFound {
				return distribution.ErrBlobUnknown
			}
			return client.HandleErrorResponse(resp)
		}), nil
}

func (bs *blobs) ServeBlob(ctx context.Context, w http.ResponseWriter, r *http.Request, dgst digest.Digest) error {
	panic("not implemented")
}

/*
func (bs *blobs) Put(ctx context.Context, mediaType string, p []byte) (distribution.Descriptor, error) {
	writer, err := bs.Create(ctx)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	dgstr := digest.Canonical.New()
	n, err := io.Copy(writer, io.TeeReader(bytes.NewReader(p), dgstr.Hash()))
	if err != nil {
		return distribution.Descriptor{}, err
	}
	if n < int64(len(p)) {
		return distribution.Descriptor{}, fmt.Errorf("short copy: wrote %d of %d", n, len(p))
	}

	desc := distribution.Descriptor{
		MediaType: mediaType,
		Size:      int64(len(p)),
		Digest:    dgstr.Digest(),
	}

	return writer.Commit(ctx, desc)
}
*/

// createOptions is a collection of blob creation modifiers relevant to general
// blob storage intended to be configured by the BlobCreateOption.Apply method.
type createOptions struct {
	Mount struct {
		ShouldMount bool
		From        reference.Canonical
	}
}

type optionFunc func(interface{}) error

func (f optionFunc) Apply(v interface{}) error {
	return f(v)
}

// WithMountFrom returns a BlobCreateOption which designates that the blob should be
// mounted from the given canonical reference.
func WithMountFrom(ref reference.Canonical) distribution.BlobCreateOption {
	return optionFunc(func(v interface{}) error {
		opts, ok := v.(*createOptions)
		if !ok {
			return fmt.Errorf("unexpected options type: %T", v)
		}

		opts.Mount.ShouldMount = true
		opts.Mount.From = ref

		return nil
	})
}

/* For the moment, we don't need this method and we don't have an impl of BlobWriter
func (bs *blobs) Create(ctx context.Context, options ...distribution.BlobCreateOption) (distribution.BlobWriter, error) {
	var opts createOptions

	for _, option := range options {
		err := option.Apply(&opts)
		if err != nil {
			return nil, err
		}
	}

	var values []url.Values

	if opts.Mount.ShouldMount {
		values = append(values, url.Values{"from": {opts.Mount.From.Name()}, "mount": {opts.Mount.From.Digest().String()}})
	}

	u, err := bs.ub.BuildBlobUploadURL(bs.name, values...)
	if err != nil {
		return nil, err
	}

	resp, err := bs.client.Post(u, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		desc, err := bs.statter.Stat(ctx, opts.Mount.From.Digest())
		if err != nil {
			return nil, err
		}
		return nil, distribution.ErrBlobMounted{From: opts.Mount.From, Descriptor: desc}
	case http.StatusAccepted:
		// TODO(dmcgowan): Check for invalid UUID
		uuid := resp.Header.Get("Docker-Upload-UUID")
		location, err := sanitizeLocation(resp.Header.Get("Location"), u)
		if err != nil {
			return nil, err
		}

		return &httpBlobUpload{
			statter:   bs.statter,
			client:    bs.client,
			uuid:      uuid,
			startedAt: time.Now(),
			location:  location,
		}, nil
	default:
		return nil, client.HandleErrorResponse(resp)
	}
}
*/

func (bs *blobs) Resume(ctx context.Context, id string) (distribution.BlobWriter, error) {
	panic("not implemented")
}

func (bs *blobs) Delete(ctx context.Context, dgst digest.Digest) error {
	return bs.statter.Clear(ctx, dgst)
}

type blobStatter struct {
	name   reference.Named
	ub     *v2.URLBuilder
	client httpClient
}

func (bs *blobStatter) Stat(ctx context.Context, dgst digest.Digest) (distribution.Descriptor, error) {
	ref, err := reference.WithDigest(bs.name, dgst)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	u, err := bs.ub.BuildBlobURL(ref)
	if err != nil {
		return distribution.Descriptor{}, err
	}

	resp, err := bs.client.Head("docker.blob", u)
	if err != nil {
		return distribution.Descriptor{}, err
	}
	defer resp.Body.Close()

	if client.SuccessStatus(resp.StatusCode) {
		lengthHeader := resp.Header.Get("Content-Length")
		if lengthHeader == "" {
			return distribution.Descriptor{}, fmt.Errorf("missing content-length header for request: %s", u)
		}

		length, err := strconv.ParseInt(lengthHeader, 10, 64)
		if err != nil {
			return distribution.Descriptor{}, fmt.Errorf("error parsing content-length: %v", err)
		}

		return distribution.Descriptor{
			MediaType: resp.Header.Get("Content-Type"),
			Size:      length,
			Digest:    dgst,
		}, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return distribution.Descriptor{}, distribution.ErrBlobUnknown
	}
	return distribution.Descriptor{}, client.HandleErrorResponse(resp)
}

func buildCatalogValues(maxEntries int, last string) url.Values {
	values := url.Values{}

	if maxEntries > 0 {
		values.Add("n", strconv.Itoa(maxEntries))
	}

	if last != "" {
		values.Add("last", last)
	}

	return values
}

func (bs *blobStatter) Clear(ctx context.Context, dgst digest.Digest) error {
	ref, err := reference.WithDigest(bs.name, dgst)
	if err != nil {
		return err
	}
	blobURL, err := bs.ub.BuildBlobURL(ref)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", blobURL, nil)
	if err != nil {
		return err
	}

	resp, err := bs.client.Do("docker.blob", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if client.SuccessStatus(resp.StatusCode) {
		return nil
	}
	return client.HandleErrorResponse(resp)
}

func (bs *blobStatter) SetDescriptor(ctx context.Context, dgst digest.Digest, desc distribution.Descriptor) error {
	return nil
}
