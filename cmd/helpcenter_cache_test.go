package main

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	goredis "github.com/zerodha/fastcache/stores/goredis/v9"
	"github.com/zerodha/fastcache/v4"
	"github.com/zerodha/fastglue"
)

func TestHelpCenterPageCacheServesAndInvalidates(t *testing.T) {
	fc := newTestFastCache(t)

	renders := 0
	body := "<h1>original</h1>"
	handler := func(r *fastglue.Request) error {
		renders++
		r.RequestCtx.SetContentType("text/html; charset=utf-8")
		r.RequestCtx.Response.Header.Set("Cache-Control", helpCenterCacheControl)
		r.RequestCtx.Response.SetBodyString(body)
		return nil
	}
	cached := fc.Cached(handler, helpCenterCacheOpts, helpCenterCacheGroup)

	const uri = "/hc/support/en/articles/refunds"

	first := request(uri, "support", "")
	if err := cached(first); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if renders != 1 {
		t.Fatalf("first request should render, renders = %d", renders)
	}
	etag := string(first.RequestCtx.Response.Header.Peek("ETag"))
	if etag == "" {
		t.Fatal("expected an ETag on the rendered page")
	}

	second := request(uri, "support", "")
	if err := cached(second); err != nil {
		t.Fatalf("second request: %v", err)
	}
	if renders != 1 {
		t.Errorf("second request re-rendered instead of using the cache, renders = %d", renders)
	}
	if got := string(second.RequestCtx.Response.Body()); got != body {
		t.Errorf("cached body = %q, want %q", got, body)
	}

	revalidate := request(uri, "support", etag)
	if err := cached(revalidate); err != nil {
		t.Fatalf("revalidation: %v", err)
	}
	if got := revalidate.RequestCtx.Response.StatusCode(); got != fasthttp.StatusNotModified {
		t.Errorf("matching ETag returned %d, want 304", got)
	}
	if got := revalidate.RequestCtx.Response.Body(); len(got) != 0 {
		t.Errorf("304 carried a body of %d bytes", len(got))
	}

	// An admin write clears the group, so the next read must re-render the edit.
	if err := fc.DelGroup("support", helpCenterCacheGroup); err != nil {
		t.Fatalf("clearing the group: %v", err)
	}
	body = "<h1>edited</h1>"
	afterEdit := request(uri, "support", "")
	if err := cached(afterEdit); err != nil {
		t.Fatalf("request after edit: %v", err)
	}
	if renders != 2 {
		t.Errorf("cleared cache did not re-render, renders = %d", renders)
	}
	if got := string(afterEdit.RequestCtx.Response.Body()); got != body {
		t.Errorf("stale body served after an edit: got %q, want %q", got, body)
	}
	if newETag := string(afterEdit.RequestCtx.Response.Header.Peek("ETag")); newETag == etag {
		t.Error("ETag survived an edit, readers holding the old copy would never refetch")
	}
}

func TestHelpCenterPageCacheIsolatesHelpCentersAndQueries(t *testing.T) {
	fc := newTestFastCache(t)

	served := map[string]int{}
	handler := func(r *fastglue.Request) error {
		uri := string(r.RequestCtx.URI().RequestURI())
		served[uri]++
		r.RequestCtx.Response.SetBodyString(uri)
		return nil
	}
	cached := fc.Cached(handler, helpCenterCacheOpts, helpCenterCacheGroup)

	for _, req := range []*fastglue.Request{
		request("/hc/support/en/search?q=refund", "support", ""),
		request("/hc/support/en/search?q=billing", "support", ""),
		request("/hc/docs/en/search?q=refund", "docs", ""),
	} {
		if err := cached(req); err != nil {
			t.Fatalf("request: %v", err)
		}
	}
	if len(served) != 3 {
		t.Fatalf("expected 3 distinct cache entries, got %d: %v", len(served), served)
	}

	// Clearing one help center must not touch another's pages.
	if err := fc.DelGroup("support", helpCenterCacheGroup); err != nil {
		t.Fatalf("clearing the group: %v", err)
	}
	if err := cached(request("/hc/docs/en/search?q=refund", "docs", "")); err != nil {
		t.Fatalf("request: %v", err)
	}
	if got := served["/hc/docs/en/search?q=refund"]; got != 1 {
		t.Errorf("clearing 'support' evicted 'docs', renders = %d", got)
	}
}

// newTestFastCache returns a cache backed by an in-memory redis.
func newTestFastCache(t *testing.T) *fastcache.FastCache {
	t.Helper()
	mr := miniredis.RunT(t)
	return fastcache.New(goredis.New(goredis.Config{Prefix: fastCachePrefix}, redis.NewClient(&redis.Options{Addr: mr.Addr()})))
}

// request builds a GET for the given URI with the slug namespace the options read.
func request(uri, slug, ifNoneMatch string) *fastglue.Request {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI(uri)
	ctx.SetUserValue("slug", slug)
	if ifNoneMatch != "" {
		ctx.Request.Header.Set("If-None-Match", ifNoneMatch)
	}
	return &fastglue.Request{RequestCtx: ctx}
}
