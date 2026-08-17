package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Regression for #445: fetchAppUpdate must always close the response body
// (including on non-200 and read-error paths) and the caller must be able to
// enforce a timeout via the client.

func TestFetchAppUpdateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"update":{"release_version":"v1.2.3"}}`))
	}))
	defer srv.Close()

	out, err := fetchAppUpdate(srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Update.ReleaseVersion != "v1.2.3" {
		t.Fatalf("got release version %q, want v1.2.3", out.Update.ReleaseVersion)
	}
}

func TestFetchAppUpdateNon200ClosesBody(t *testing.T) {
	var closed bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// Wrap the transport so we can observe that the body is drained/closed.
	client := srv.Client()
	client.Transport = &closeTrackingTransport{
		base:    client.Transport,
		onClose: func() { closed = true },
	}

	out, err := fetchAppUpdate(client, srv.URL)
	if err == nil {
		t.Fatal("expected an error for a non-200 status")
	}
	if out != nil {
		t.Fatalf("expected nil result on error, got %+v", out)
	}
	if !closed {
		t.Fatal("response body was not closed on the non-200 path (leak)")
	}
}

func TestFetchAppUpdateTimesOut(t *testing.T) {
	// A server that accepts the connection but never responds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	client := srv.Client()
	client.Timeout = 100 * time.Millisecond

	start := time.Now()
	_, err := fetchAppUpdate(client, srv.URL)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("call did not respect the client timeout: took %s", elapsed)
	}
}

// closeTrackingTransport wraps a RoundTripper and swaps the response body for
// one that reports when it is closed.
type closeTrackingTransport struct {
	base    http.RoundTripper
	onClose func()
}

func (t *closeTrackingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	resp.Body = &closeNotifyReadCloser{ReadCloser: resp.Body, onClose: t.onClose}
	return resp, nil
}

type closeNotifyReadCloser struct {
	io.ReadCloser
	onClose func()
}

func (c *closeNotifyReadCloser) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return c.ReadCloser.Close()
}
