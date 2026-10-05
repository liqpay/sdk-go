// Package liqpay is a client SDK for the Liqpay payment API.
//
// See https://www.liqpay.ua for more information about the service.
package liqpay

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // SHA-1 is mandated by the Liqpay signature protocol.
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"
)

// DefaultBaseURL is the default base URL of the Liqpay API. It can be
// overridden per client with WithBaseURL.
const DefaultBaseURL = "https://www.liqpay.ua/api/"

// defaultEndpoint is the Liqpay API endpoint used for data requests. It is
// the only endpoint reachable through Send/SendContext; the checkout form is
// handled separately by RenderForm.
const defaultEndpoint = "request"

// checkoutEndpoint is the Liqpay endpoint the checkout form posts to,
// relative to the client's base URL.
const checkoutEndpoint = "3/checkout"

// maxErrorBodyBytes limits how much of a response body is read into an error
// message, to avoid unbounded memory use on unexpected responses.
const maxErrorBodyBytes = 4 << 10 // 4 KiB

// formData holds the values injected into the checkout form template.
type formData struct {
	// CheckoutURL is the absolute URL the form posts to. It is derived from
	// the client's base URL so that sandbox/custom base URLs stay consistent.
	CheckoutURL string
	Data        string
	Signature   string
}

//go:embed liqpay_form.html
var formTemplateSource string

// formTemplate is parsed once at package initialization so that RenderForm
// does not depend on the current working directory at runtime.
var formTemplate = template.Must(template.New("liqpay_form").Parse(formTemplateSource))

// Client is a Liqpay API client. Create one with New.
type Client struct {
	httpClient *http.Client
	publicKey  string
	privateKey string
	baseURL    string
}

// Request is a set of parameters sent to the Liqpay API.
type Request map[string]any

// Response is a decoded Liqpay API response.
type Response map[string]any

// Option configures a Client. Pass options to New.
type Option func(*Client)

// WithBaseURL overrides the base URL used for API requests. The trailing
// slash is normalized, so both "https://host/api" and "https://host/api/"
// behave identically. Useful for sandbox environments and testing.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/") + "/"
	}
}

// APIError is returned when the Liqpay API reports a failed operation
// (result/status of "error" or "failure"). Use errors.As to inspect it.
type APIError struct {
	// Code is the machine-readable error code (err_code), if present.
	Code string
	// Description is the human-readable error description (err_description).
	Description string
	// Status is the status/result value reported by the API.
	Status string
	// Response is the full decoded response for further inspection.
	//
	// It may contain sensitive fields echoed back by the API (for example
	// transaction or payment details). Avoid logging it verbatim; prefer
	// logging specific fields such as Code and Description.
	Response Response
}

// Error implements the error interface.
func (e *APIError) Error() string {
	switch {
	case e.Description != "" && e.Code != "":
		return fmt.Sprintf("liqpay api error (code %s): %s", e.Code, e.Description)
	case e.Description != "":
		return fmt.Sprintf("liqpay api error: %s", e.Description)
	case e.Code != "":
		return fmt.Sprintf("liqpay api error: code %s", e.Code)
	default:
		return fmt.Sprintf("liqpay api error: status %q", e.Status)
	}
}

// New creates a Client with the given public and private keys.
//
// If client is nil, a default *http.Client is used. Additional behavior can be
// configured with Option values such as WithBaseURL.
func New(pubKey string, privKey string, client *http.Client, opts ...Option) *Client {
	c := &Client{
		httpClient: client,
		publicKey:  pubKey,
		privateKey: privKey,
		baseURL:    DefaultBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}
	return c
}

// Send performs a request to the given Liqpay API endpoint (for example
// "request") and returns the decoded response.
//
// It uses context.Background.
//
// Deprecated: use SendContext, which targets the standard "request" endpoint
// and accepts a context.Context for cancellation and deadlines.
func (c *Client) Send(apiURL string, req Request) (Response, error) {
	return c.sendContext(context.Background(), apiURL, req)
}

// SendContext performs a request to the Liqpay API using the provided context
// and returns the decoded response.
//
// If the API reports a failed operation, the returned error is an *APIError.
func (c *Client) SendContext(ctx context.Context, req Request) (Response, error) {
	return c.sendContext(ctx, defaultEndpoint, req)
}

// sendContext holds the shared request logic for Send and SendContext.
func (c *Client) sendContext(ctx context.Context, apiURL string, req Request) (Response, error) {
	req.addMissingPubKey(c.publicKey)

	encodedJSON, err := req.Encode()
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	signature := c.Sign([]byte(encodedJSON))
	form := url.Values{
		"data":      {encodedJSON},
		"signature": {signature},
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+apiURL,
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body := readLimited(resp.Body)
		if body != "" {
			return nil, fmt.Errorf("bad response status code %d: %s", resp.StatusCode, body)
		}
		return nil, fmt.Errorf("bad response status code %d", resp.StatusCode)
	}

	// Read the body so it can be included in a decode error for easier debugging.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var res Response
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("decode response: %w: %s", err, truncate(string(raw), maxErrorBodyBytes))
	}

	if apiErr := res.asAPIError(); apiErr != nil {
		return nil, apiErr
	}

	return res, nil
}

// RenderForm renders an HTML checkout form for the given request.
func (c *Client) RenderForm(req Request) (string, error) {
	req.addMissingPubKey(c.publicKey)

	encodedJSON, err := req.Encode()
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	signature := c.Sign([]byte(encodedJSON))

	var buf bytes.Buffer
	if err := formTemplate.Execute(&buf, formData{
		CheckoutURL: c.baseURL + checkoutEndpoint,
		Data:        encodedJSON,
		Signature:   signature,
	}); err != nil {
		return "", fmt.Errorf("render form: %w", err)
	}
	return buf.String(), nil
}

// asAPIError returns an *APIError if the response reports a failed operation,
// or nil otherwise.
func (r Response) asAPIError() *APIError {
	status, _ := r["status"].(string)
	result, _ := r["result"].(string)

	if !isFailureStatus(status) && !isFailureStatus(result) {
		return nil
	}

	reported := status
	if reported == "" {
		reported = result
	}

	code, _ := r["err_code"].(string)
	desc, _ := r["err_description"].(string)
	return &APIError{
		Code:        code,
		Description: desc,
		Status:      reported,
		Response:    r,
	}
}

// isFailureStatus reports whether a status/result value indicates a failure.
func isFailureStatus(s string) bool {
	return s == "error" || s == "failure"
}

// addMissingPubKey sets the public_key field if it is not already set to key.
//
// Note: this mutates the provided Request in place.
func (r Request) addMissingPubKey(key string) {
	if r["public_key"] == key {
		return
	}
	r["public_key"] = key
}

// Encode marshals the request to JSON and returns it base64-encoded.
func (r Request) Encode() (string, error) {
	obj, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(obj), nil
}

// Sign computes the Liqpay signature for data using the client's private key.
//
// Liqpay requires the signature to be base64(sha1(privateKey + data + privateKey)).
func (c *Client) Sign(data []byte) string {
	hasher := sha1.New() //nolint:gosec // SHA-1 is mandated by the Liqpay signature protocol.
	hasher.Write([]byte(c.privateKey))
	hasher.Write(data)
	hasher.Write([]byte(c.privateKey))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

// readLimited reads up to maxErrorBodyBytes from r and returns the trimmed
// result as a string. Errors are ignored, since this is best-effort context
// for an error message.
func readLimited(r io.Reader) string {
	buf, _ := io.ReadAll(io.LimitReader(r, maxErrorBodyBytes))
	return strings.TrimSpace(string(buf))
}

// truncate shortens s to at most n bytes, appending an ellipsis if truncated.
// The cut is made on a UTF-8 rune boundary so a multi-byte rune is never split
// in the middle.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Back off to the start of the rune that straddles the n-byte boundary.
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "..."
}

// Ensure APIError satisfies the error interface at compile time.
var _ error = (*APIError)(nil)
