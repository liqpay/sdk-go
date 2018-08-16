package liqpay

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func TestClient_Send(t *testing.T) {
	type fields struct {
		httpClient *http.Client
		publicKey  []byte
		privateKey []byte
	}
	type args struct {
		apiUrl string
		req    Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Response
		wantErr bool
	}{
		{
			name: `error_resp`,
			fields: fields{
				httpClient: nil,
				publicKey:  []byte("test_pub_key"),
				privateKey: []byte("test_priv_key"),
			},
			args: args{
				apiUrl: "request",
				req: Request{
					"str": "value",
					"num": 124.0,
				},
			},
			want: Response{
				"result":          "error",
				"status":          "error",
				"code":            "public_key_not_found",
				"err_code":        "public_key_not_found",
				"err_description": "Не найден public_key",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Client{
				httpClient: tt.fields.httpClient,
				publicKey:  tt.fields.publicKey,
				privateKey: tt.fields.privateKey,
			}
			got, err := c.Send(tt.args.apiUrl, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Send() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.Send() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_RenderForm(t *testing.T) {
	type fields struct {
		httpClient *http.Client
		publicKey  []byte
		privateKey []byte
	}
	type args struct {
		req Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				httpClient: nil,
				publicKey:  []byte("test_pub_key"),
				privateKey: []byte("test_priv_key"),
			},
			args: args{
				req: Request{
					"str": "value",
					"num": 124.0,
				},
			},
			want: `<form method="POST" action="https://www.liqpay.ua/api/3/checkout" accept-charset="utf-8">
    					<input type="hidden" name="data" value="eyJudW0iOjEyNCwicHVibGljX2tleSI6InRlc3RfcHViX2tleSIsInN0ciI6InZhbHVlIn0="/>
    					<input type="hidden" name="signature" value="9Altos5Ex1c7MX5uoKSrKXFK3mQ="/>
    					<input type="image" src="https://static.liqpay.ua/buttons/p1ru.radius.png"/>
					</form>`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Client{
				httpClient: tt.fields.httpClient,
				publicKey:  tt.fields.publicKey,
				privateKey: tt.fields.privateKey,
			}
			got, err := c.RenderForm(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.RenderForm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Client.RenderForm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequest_addMissingPubKey(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		r    Request
		args args
	}{
		{
			name: `empty_public_key`,
			r:    Request{},
			args: args{
				key: "public_key",
			},
		},
		{
			name: `another_public_key`,
			r: Request{
				"public_key": "another_key",
			},
			args: args{
				key: "public_key",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.addMissingPubKey(tt.args.key)
			if tt.r["public_key"] != tt.args.key {
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
			name: `empty_public_key`,
			r: Request{
				"str": "value",
				"num": 124.0,
			},
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
			if (err != nil) != tt.wantErr {
				t.Errorf("Request.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(req, tt.r) {
				t.Errorf("Request = %#v, want %#v", tt.r, req)
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
