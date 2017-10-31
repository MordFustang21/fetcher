package fetcher

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"time"
)

var _ Fetcher = (*Client)(nil)

// Client implements Fetcher interface and is required to execute a Request
type Client struct {
	client *http.Client

	keepAlive        time.Duration
	handshakeTimeout time.Duration
}

// NewClient returns a new Client with the given options executed
func NewClient(c context.Context, opts ...ClientOption) (*Client, error) {
	cl := &Client{
		keepAlive:        60 * time.Second,
		handshakeTimeout: 10 * time.Second,
	}

	var err error

	// execute all options
	for _, opt := range opts {
		if err = opt(c, cl); err != nil {
			return nil, err
		}
	}

	cl.setClient()

	return cl, nil
}

func (cl *Client) Do(c context.Context, req *Request) (*Response, error) {
	// if the context has been canceled or the deadline exceeded, don't start the request
	if c.Err() != nil {
		return nil, c.Err()
	}

	req.client = cl

	reqc := req.request.WithContext(c)

	if buf, ok := req.payload.(*bytes.Buffer); ok {
		defer putBuffer(buf)
	}

	resp := &Response{}
	var err error
	for i := 0; i < req.maxAttempts; i++ {
		resp.response, err = cl.client.Do(reqc)
		if err != nil {
			return nil, err
		}

		// further attempts will be made only on 500+ status codes
		// NOTE: the error returned from cl.client.Do(reqc) only contains scenarios regarding
		// a bad request given, or a response with Location header missing or bad
		if resp.response.StatusCode < 500 {
			break
		}
	}

	// execute all afterDoFuncs
	for _, afterDo := range req.afterDoFuncs {
		if err = afterDo(req, resp); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// Get is a helper func for Do, setting the Method internally
func (cl *Client) Get(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodGet, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Head is a helper func for Do, setting the Method internally
func (cl *Client) Head(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodHead, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Post is a helper func for Do, setting the Method internally
func (cl *Client) Post(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodPost, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Put is a helper func for Do, setting the Method internally
func (cl *Client) Put(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodPut, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Patch is a helper func for Do, setting the Method internally
func (cl *Client) Patch(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodPatch, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// Delete is a helper func for Do, setting the Method internally
func (cl *Client) Delete(c context.Context, url string, opts ...RequestOption) (*Response, error) {
	req, err := NewRequest(c, http.MethodDelete, url, opts...)
	if err != nil {
		return nil, err
	}
	return cl.Do(c, req)
}

// ClientOption is a func to configure optional Client settings
type ClientOption func(c context.Context, cl *Client) error

// ClientWithKeepAlive is a ClientOption that sets the cl.keepAlive field to the given duration
func ClientWithKeepAlive(dur time.Duration) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.keepAlive = dur
		return nil
	}
}

// ClientWithHandshakeTimeout is a ClientOption that sets the cl.handshakeTimeout field to the given duration
func ClientWithHandshakeTimeout(dur time.Duration) ClientOption {
	return func(c context.Context, cl *Client) error {
		cl.handshakeTimeout = dur
		return nil
	}
}

// setClient creates the standard http.Client using the settings in the given Client
func (cl *Client) setClient() {
	cl.client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				KeepAlive: cl.keepAlive,
			}).Dial,
			TLSHandshakeTimeout: cl.handshakeTimeout,
		},
	}
}