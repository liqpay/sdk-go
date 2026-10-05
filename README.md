# sdk-go

[![Go Reference](https://pkg.go.dev/badge/github.com/liqpay/sdk-go.svg)](https://pkg.go.dev/github.com/liqpay/sdk-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/liqpay/sdk-go)](go.mod)

Liqpay SDK for Go — a thin, idiomatic client for the [Liqpay](https://www.liqpay.ua) payment API.

## Features

- Simple request/response API over the Liqpay HTTP endpoint
- Automatic request signing (`base64(sha1(privateKey + data + privateKey))`)
- Typed API errors via `*APIError` (works with `errors.As`)
- `context.Context` support for cancellation and deadlines
- Configurable base URL for sandbox and testing environments
- Checkout form rendering

## Requirements

- Go 1.18 or newer

## Installation

```bash
go get github.com/liqpay/sdk-go
```

## Usage

### Sending a request

Example: retrieve received payments.

```go
package main

import (
	"context"
	"fmt"

	liqpay "github.com/liqpay/sdk-go"
)

func main() {
	c := liqpay.New("your_pub_key", "your_priv_key", nil)

	r := liqpay.Request{
		"action":    "reports",
		"version":   3,
		"date_from": 1443161386000,
		"date_to":   1443164386000,
	}

	resp, err := c.SendContext(context.Background(), r)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("response: %#v\n", resp)
}
```

> `SendContext` targets the standard `request` endpoint. The older
> `Send(apiURL, req)` method is **deprecated** and kept only for backward
> compatibility.

### Handling API errors

When the Liqpay API reports a failed operation (`status`/`result` of `error`
or `failure`), `SendContext` returns an `*APIError`. Use `errors.As` to
inspect it:

```go
resp, err := c.SendContext(context.Background(), r)
if err != nil {
	var apiErr *liqpay.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("liqpay rejected the request: code=%s description=%s\n",
			apiErr.Code, apiErr.Description)
		return
	}
	// transport/encoding/decoding error
	fmt.Printf("error: %v\n", err)
	return
}
```

### Using a context

`SendContext` lets you pass a `context.Context` for cancellation and deadlines:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := c.SendContext(ctx, r)
```

### Options

`New` accepts optional configuration via functional options:

```go
// Point the client at a different base URL (e.g. for testing or sandboxes).
c := liqpay.New("pub", "priv", nil, liqpay.WithBaseURL("https://sandbox.example/api"))

// Provide a custom *http.Client via the third argument of New.
c = liqpay.New("pub", "priv", &http.Client{Timeout: 10 * time.Second})
```

The base URL defaults to `liqpay.DefaultBaseURL`.

### Rendering a checkout form

```go
html, err := c.RenderForm(liqpay.Request{
	"action":   "pay",
	"version":  3,
	"amount":   1,
	"currency": "UAH",
})
```

## Documentation

- Package reference: [pkg.go.dev/github.com/liqpay/sdk-go](https://pkg.go.dev/github.com/liqpay/sdk-go)
- Liqpay API documentation: [www.liqpay.ua/uk/doc](https://www.liqpay.ua/uk/doc)
- Liqpay website: [www.liqpay.ua](https://www.liqpay.ua)
