package liqpay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// newTestClient creates a Client pointed at the given test server using the
// public WithBaseURL option, so no transport-level hacks are required.
func newTestClient(srv *httptest.Server) *Client {
	return New("test_pub_key", "test_priv_key", nil, WithBaseURL(srv.URL))
}

func TestClient_Send(t *testing.T) {
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		req      Request
		want     Response
		wantErr  bool
		errMatch string
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"success","result":"ok"}`))
			},
			req:     Request{"str": "value", "num": 124.0},
			want:    Response{"status": "success", "result": "ok"},
			wantErr: false,
		},
		{
			name: "liqpay_error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"error","err_description":"invalid public_key"}`))
			},
			req:      Request{"str": "value"},
			wantErr:  true,
			errMatch: "invalid public_key",
		},
		{
			name: "liqpay_failure",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"failure","err_code":"err_auth","err_description":"auth failed"}`))
			},
			req:      Request{"str": "value"},
			wantErr:  true,
			errMatch: "auth failed",
		},
		{
			name: "bad_status_code",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal boom"))
			},
			req:      Request{"str": "value"},
			wantErr:  true,
			errMatch: "bad response status code 500: internal boom",
		},
		{
			name: "invalid_json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`<html>not json</html>`))
			},
			req:      Request{"str": "value"},
			wantErr:  true,
			errMatch: "decode response",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			c := newTestClient(srv)
			got, err := c.Send("request", tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMatch != "" && (err == nil || !strings.Contains(err.Error(), tt.errMatch)) {
					t.Errorf("Client.Send() error = %v, want contains %q", err, tt.errMatch)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.Send() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestClient_Send_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"failure","err_code":"err_auth","err_description":"auth failed"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Send("request", Request{"str": "value"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != "err_auth" {
		t.Errorf("APIError.Code = %q, want %q", apiErr.Code, "err_auth")
	}
	if apiErr.Description != "auth failed" {
		t.Errorf("APIError.Description = %q, want %q", apiErr.Description, "auth failed")
	}
	if apiErr.Status != "failure" {
		t.Errorf("APIError.Status = %q, want %q", apiErr.Status, "failure")
	}
	if apiErr.Response["err_code"] != "err_auth" {
		t.Errorf("APIError.Response not populated: %#v", apiErr.Response)
	}
}

func TestClient_SendContext_Canceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before sending

	c := newTestClient(srv)
	_, err := c.SendContext(ctx, Request{"str": "value"})
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWithBaseURL_Normalization(t *testing.T) {
	// Both with and without a trailing slash must resolve to the same URL.
	c1 := New("p", "s", nil, WithBaseURL("https://example.test/api"))
	c2 := New("p", "s", nil, WithBaseURL("https://example.test/api/"))
	if c1.baseURL != c2.baseURL {
		t.Errorf("baseURL not normalized: %q vs %q", c1.baseURL, c2.baseURL)
	}
	if c1.baseURL != "https://example.test/api/" {
		t.Errorf("baseURL = %q, want %q", c1.baseURL, "https://example.test/api/")
	}
}

func TestNew_DefaultBaseURL(t *testing.T) {
	c := New("p", "s", nil)
	if c.baseURL != DefaultBaseURL {
		t.Errorf("default baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
	if c.httpClient == nil {
		t.Error("httpClient should default to non-nil")
	}
}

func TestClient_RenderForm(t *testing.T) {
	c := New("test_pub_key", "test_priv_key", nil)
	got, err := c.RenderForm(Request{"str": "value", "num": 124.0})
	if err != nil {
		t.Fatalf("Client.RenderForm() error = %v", err)
	}

	const wantData = "eyJudW0iOjEyNCwicHVibGljX2tleSI6InRlc3RfcHViX2tleSIsInN0ciI6InZhbHVlIn0="
	const wantSig = "9Altos5Ex1c7MX5uoKSrKXFK3mQ="

	if !strings.Contains(got, `name="data" value="`+wantData+`"`) {
		t.Errorf("RenderForm() missing expected data value, got:\n%s", got)
	}
	if !strings.Contains(got, `name="signature" value="`+wantSig+`"`) {
		t.Errorf("RenderForm() missing expected signature value, got:\n%s", got)
	}
	if !strings.Contains(got, `action="https://www.liqpay.ua/api/3/checkout"`) {
		t.Errorf("RenderForm() missing expected action, got:\n%s", got)
	}
}

func TestRequest_addMissingPubKey(t *testing.T) {
	tests := []struct {
		name string
		r    Request
		key  string
	}{
		{
			name: "empty_public_key",
			r:    Request{},
			key:  "public_key",
		},
		{
			name: "another_public_key",
			r:    Request{"public_key": "another_key"},
			key:  "public_key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.addMissingPubKey(tt.key)
			if tt.r["public_key"] != tt.key {
				t.Fail()
			}
		})
	}
}

func TestRequest_Encode(t *testing.T) {
	tests := []struct {
		name    string
		r       Request
		wantErr bool
	}{
		{
			name:    "basic",
			r:       Request{"str": "value", "num": 124.0},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Request.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			req, err := decode(got)
			if err != nil {
				t.Errorf("decode() error = %v", err)
				return
			}
			if !reflect.DeepEqual(req, tt.r) {
				t.Errorf("Request = %#v, want %#v", req, tt.r)
			}
		})
	}
}

func decode(encoded string) (Request, error) {
	var req Request
	buf, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(buf, &req); err != nil {
		return nil, err
	}
	return req, nil
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "code_and_description",
			err:  &APIError{Code: "err_auth", Description: "auth failed"},
			want: "liqpay api error (code err_auth): auth failed",
		},
		{
			name: "description_only",
			err:  &APIError{Description: "auth failed"},
			want: "liqpay api error: auth failed",
		},
		{
			name: "code_only",
			err:  &APIError{Code: "err_auth"},
			want: "liqpay api error: code err_auth",
		},
		{
			name: "status_only",
			err:  &APIError{Status: "failure"},
			want: `liqpay api error: status "failure"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("APIError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequest_Encode_MarshalError(t *testing.T) {
	// A function value cannot be marshaled to JSON, forcing the error branch.
	r := Request{"bad": func() {}}
	_, err := r.Encode()
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
	if !strings.Contains(err.Error(), "marshal request") {
		t.Errorf("Encode() error = %v, want contains %q", err, "marshal request")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{name: "shorter_than_limit", s: "abc", n: 10, want: "abc"},
		{name: "equal_to_limit", s: "abcde", n: 5, want: "abcde"},
		{name: "ascii_truncated", s: "abcdef", n: 3, want: "abc..."},
		{
			// "é" is 2 bytes (0xC3 0xA9). Cutting at n=1 lands inside the rune;
			// truncate must back off to byte 0 so the rune is not split.
			name: "utf8_boundary_backoff",
			s:    "é" + "xyz",
			n:    1,
			want: "...",
		},
		{
			// Cutting right after a full multi-byte rune keeps it intact.
			name: "utf8_clean_boundary",
			s:    "é" + "xyz",
			n:    2,
			want: "é...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncate(tt.s, tt.n); got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func TestRequest_addMissingPubKey_AlreadySet(t *testing.T) {
	// When the key already matches, the map must be left untouched (early return).
	r := Request{"public_key": "same_key", "other": "value"}
	r.addMissingPubKey("same_key")
	if r["public_key"] != "same_key" {
		t.Errorf("public_key = %v, want %q", r["public_key"], "same_key")
	}
	if r["other"] != "value" {
		t.Errorf("other field mutated: %v", r["other"])
	}
}

func TestClient_RenderForm_CustomBaseURL(t *testing.T) {
	c := New("test_pub_key", "test_priv_key", nil, WithBaseURL("https://sandbox.example/api"))
	got, err := c.RenderForm(Request{"str": "value"})
	if err != nil {
		t.Fatalf("RenderForm() error = %v", err)
	}
	const wantAction = `action="https://sandbox.example/api/3/checkout"`
	if !strings.Contains(got, wantAction) {
		t.Errorf("RenderForm() missing custom action %q, got:\n%s", wantAction, got)
	}
}
