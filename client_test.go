package unifi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestClientBadContentType(t *testing.T) {
	c, done := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`foo`))
	})
	defer done()

	req, err := c.newRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Not the best possible check but verifies that the content type is incorrect
	_, err = c.do(req, nil)
	if want, got := `received "text/plain; charset=utf-8"`, err.Error(); !strings.Contains(got, want) {
		t.Fatalf("unexpected error message: %v", got)
	}
}

func TestClientBadHTTPStatusCode(t *testing.T) {
	c, done := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", jsonContentType)
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer done()

	req, err := c.newRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Not the best possible check but verifies that the content type is incorrect
	_, err = c.do(req, nil)
	if want, got := `unexpected HTTP status code: 500`, err.Error(); !strings.Contains(got, want) {
		t.Fatalf("unexpected error message: %v", got)
	}
}

func TestClientBadJSON(t *testing.T) {
	c, done := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", jsonContentType)
		_, _ = w.Write([]byte(`foo`))
	})
	defer done()

	req, err := c.newRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pass empty struct to trigger JSON unmarshaling path
	var v struct{}

	_, err = c.do(req, &v)
	if _, ok := err.(*json.SyntaxError); !ok {
		t.Fatalf("unexpected error type: %T", err)
	}
}

func TestClientRetainsCookies(t *testing.T) {
	const cookieName = "foo"
	wantCookie := &http.Cookie{
		Name:  cookieName,
		Value: "bar",
	}

	var i int
	c, done := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		defer func() { i++ }()

		w.Header().Set("Content-Type", jsonContentType)

		switch i {
		case 0:
			http.SetCookie(w, wantCookie)
		case 1:
			c, err := r.Cookie(cookieName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := wantCookie, c; !reflect.DeepEqual(want, got) {
				t.Fatalf("unexpected cookie:\n- want: %v\n-  got: %v",
					want, got)
			}
		}

		_, _ = w.Write([]byte(`{}`))
	})
	defer done()

	for i := 0; i < 2; i++ {
		req, err := c.newRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = c.do(req, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

func TestClientLogin(t *testing.T) {
	const (
		wantUsername = "test"
		wantPassword = "test"
	)

	wantBody := &login{
		Username: wantUsername,
		Password: wantPassword,
	}

	c, done := testClient(t, testHandler(t, http.MethodPost, "/api/login", wantBody, nil))
	defer done()

	if err := c.Login(wantUsername, wantPassword); err != nil {
		t.Fatalf("unexpected error from Client.Login: %v", err)
	}
}

func TestInsecureHTTPClient(t *testing.T) {
	timeout := 5 * time.Second
	c := InsecureHTTPClient(timeout)

	if want, got := c.Timeout, timeout; want != got {
		t.Fatalf("unexpected client timeout:\n- want: %v\n-  got: %v",
			want, got)
	}

	got := c.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify
	if want := true; want != got {
		t.Fatalf("unexpected client insecure skip verify value:\n- want: %v\n-  got: %v",
			want, got)
	}
}

func testClient(t *testing.T, fn func(w http.ResponseWriter, r *http.Request)) (*Client, func()) {
	s := httptest.NewServer(http.HandlerFunc(fn))

	c, err := NewClient(s.URL, nil)
	if err != nil {
		t.Fatalf("error creating Client: %v", err)
	}

	return c, func() { s.Close() }
}

func testHandler(t *testing.T, method string, path string, body interface{}, out interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if want, got := method, r.Method; want != got {
			t.Fatalf("unexpected HTTP method:\n- want: %v\n-  got: %v", want, got)
		}

		if want, got := path, r.URL.Path; want != got {
			t.Fatalf("unexpected URL path:\n- want: %v\n-  got: %v", want, got)
		}

		if r.Method != http.MethodPost && r.Method != http.MethodPut {
			w.Header().Set("Content-Type", jsonContentType)
			if err := json.NewEncoder(w).Encode(out); err != nil {
				t.Fatalf("error marshaling JSON response body: %v", err)
			}
			return
		}

		// Content-Length must always be set for POST/PUT
		cls := r.Header.Get("Content-Length")
		if cls == "" {
			t.Fatal("Content-Length header is not present or empty")
		}
		cl, err := strconv.Atoi(cls)
		if err != nil {
			t.Fatalf("unexpected error parsing Content-Length: %v", err)
		}

		// Content-Length must match body length
		b := make([]byte, cl)
		if n, err := io.ReadFull(r.Body, b); err != nil {
			t.Fatalf("failed to read entire JSON body: read %d bytes, err: %v", n, err)
		}

		// Body must be valid JSON
		var v struct{}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("error unmarshaling JSON body: %v", err)
		}

		// Request body must match input JSON
		wantJSON, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("error marshaling input body to JSON: %v", err)
		}

		if want, got := string(wantJSON), strings.TrimSpace(string(b)); want != got {
			t.Fatalf("unexpected JSON request body:\n- want: %v\n-  got: %v",
				want, got)
		}

		// Write output JSON if set
		w.Header().Set("Content-Type", jsonContentType)
		if err := json.NewEncoder(w).Encode(out); err != nil {
			t.Fatalf("error marshaling JSON response body: %v", err)
		}
	}
}
