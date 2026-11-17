package api_client

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func droppingServer(t *testing.T, drop func(*net.TCPConn)) (*httptest.Server, *atomic.Int32) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		drop(conn.(*net.TCPConn))
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

func TestPostCreate_ConnectionDroppedAfterSendIsAmbiguous(t *testing.T) {
	stubRetrySleep(t)
	for name, drop := range map[string]func(*net.TCPConn){
		"closed": func(c *net.TCPConn) { _ = c.Close() },
		"reset": func(c *net.TCPConn) {
			_ = c.SetLinger(0)
			_ = c.Close()
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv, n := droppingServer(t, drop)
			c := &APIClient{APIEndpoint: srv.URL, APIToken: "secret", HTTPClient: srv.Client()}
			_, err := c.PostCreate("/api/x", map[string]string{"a": "b"})
			if !errors.Is(err, ErrAmbiguousWrite) {
				t.Fatalf("err = %v, want ErrAmbiguousWrite", err)
			}
			if got := n.Load(); got != 1 {
				t.Fatalf("server saw %d creates, want 1", got)
			}
		})
	}
}

func TestPostCreate_ConnectionRefusedIsNotAmbiguous(t *testing.T) {
	stubRetrySleep(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	c := &APIClient{APIEndpoint: url, APIToken: "secret", HTTPClient: &http.Client{}}
	_, err := c.PostCreate("/api/x", map[string]string{"a": "b"})
	if err == nil || errors.Is(err, ErrAmbiguousWrite) {
		t.Fatalf("err = %v, want a plain error: nothing was sent", err)
	}
}

func TestPost_ConnectionDroppedIsNotAmbiguous(t *testing.T) {
	stubRetrySleep(t)
	srv, _ := droppingServer(t, func(c *net.TCPConn) { _ = c.Close() })
	c := &APIClient{APIEndpoint: srv.URL, APIToken: "secret", HTTPClient: srv.Client()}
	if _, err := c.Post("/api/x", map[string]string{"a": "b"}); errors.Is(err, ErrAmbiguousWrite) {
		t.Fatalf("err = %v: only PostCreate marks ambiguous writes", err)
	}
}
