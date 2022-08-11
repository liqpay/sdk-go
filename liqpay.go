package liqpay

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
)

const liqpayURL = "https://www.liqpay.ua/api/"

type formData struct {
	Data      string
	Signature string
}

type Client struct {
	httpClient *http.Client
	publicKey  []byte
	privateKey []byte
}

type Request map[string]interface{}

type Response map[string]interface{}

func New(pubKey string, privKey string, client *http.Client) *Client {
	var c *http.Client
	if client == nil {
		c = &http.Client{}
	} else {
		c = client
	}
	return &Client{
		httpClient: c,
		publicKey:  []byte(pubKey),
		privateKey: []byte(privKey),
	}
}

func (c Client) Send(apiUrl string, req Request) (Response, error) {
	req.addMissingPubKey(string(c.publicKey))

	encodedJSON, err := req.Encode()
	if err != nil {
		return nil, err
	}

	signature := c.Sign([]byte(encodedJSON))
	form := url.Values{
		"data":      {encodedJSON},
		"signature": {signature},
	}

	reqBody := bytes.NewBufferString(form.Encode())
	resp, err := http.Post(liqpayURL+apiUrl, "application/x-www-form-urlencoded", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("bad response status code, status code is %d", resp.StatusCode))
	}

	var res Response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if res["status"] == "error" || res["result"] == "error" {
		errMsg, ok := res["err_description"].(string)
		if ok {
			return nil, errors.New(errMsg)
		}
		return nil, errors.New("response body has status error but didn't get error description")
	}

	return res, nil
}

func (c Client) RenderForm(req Request) (string, error) {
	req.addMissingPubKey(string(c.publicKey))

	encodedJSON, err := req.Encode()
	if err != nil {
		return "", err
	}

	signature := c.Sign([]byte(encodedJSON))

	t, err := template.ParseFiles("liqpay_form.html")
	if err != nil {
		return "", err
	}
	buf := bytes.Buffer{}
	if err := t.Execute(&buf, formData{
		Data:      encodedJSON,
		Signature: signature,
	}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (r Request) addMissingPubKey(key string) {
	if r["public_key"] == key {
		return
	}
	r["public_key"] = key
}

func (r Request) Encode() (string, error) {
	obj, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(obj), nil
}

func (c Client) Sign(data []byte) string {
	hasher := sha1.New()
	hasher.Write(c.privateKey)
	hasher.Write(data)
	hasher.Write(c.privateKey)
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
