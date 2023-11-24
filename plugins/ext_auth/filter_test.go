package ext_auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/plugins/tests/pkg/envoy"
)

func response(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       http.NoBody,
		Header:     http.Header{},
	}
}

func TestExtAuth(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		hdr    map[string][]string
		server func(r *http.Request) (*http.Response, error)
		res    api.ResultAction
		upHdr  map[string][]string
	}{
		{
			name: "default",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth"
			}}`,
			hdr: map[string][]string{
				"Authorization": {"Basic amFjazIwMjE6MTIzNDU2"},
				"Other":         {"not passed"},
			},
			server: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "test.local", r.Host)
				assert.Equal(t, "/ext_auth/", r.URL.Path)
				assert.Equal(t, "Basic amFjazIwMjE6MTIzNDU2", r.Header.Get("Authorization"))
				assert.Equal(t, "", r.Header.Get("Other"))
				return response(200), nil
			},
		},
		{
			name: "add headers",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth",
				"authorization_request": {
					"headers_to_add": [
						{"key": "foo", "value": "bar"},
						{"key": "foo", "value": "baz"}
					]
				}
			}}`,
			hdr: map[string][]string{
				"Foo": {"blah"},
			},
			server: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, []string{"baz"}, r.Header.Values("Foo"))
				return response(200), nil
			},
		},
		{
			name: "auth denied",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth"
			}}`,
			server: func(r *http.Request) (*http.Response, error) {
				resp := response(401)
				resp.Header.Set("foo", "bar")
				resp.Header.Set("date", "now")
				return resp, nil
			},
			res: &api.LocalResponse{Code: 401,
				Header: http.Header(map[string][]string{
					"Foo":  {"bar"},
					"Date": {"now"},
				}),
			},
		},
		{
			name: "auth error",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth"
			}}`,
			server: func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("ouch")
			},
			res: &api.LocalResponse{Code: 403},
		},
		{
			name: "auth error, status_on_error configured",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth",
				"status_on_error": 401
			}}`,
			server: func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("ouch")
			},
			res: &api.LocalResponse{Code: 401},
		},
		{
			name: "add matched headers",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth",
				"authorization_response": {
					"allowed_upstream_headers": [
						{"exact": "foo"},
						{"regex": "^ba(r|lh)$"}
					]
				}
			}}`,
			hdr: map[string][]string{
				// header from request will be overridden
				"foo": {"blah"},
			},
			server: func(r *http.Request) (*http.Response, error) {
				resp := response(200)
				resp.Header.Set("foo", "bar")
				resp.Header.Set("bar", "foo")
				resp.Header.Add("balh", "foo")
				resp.Header.Add("balh", "bar")
				resp.Header.Set("blah", "foo")
				return resp, nil
			},
			upHdr: map[string][]string{
				"foo":  {"bar"},
				"bar":  {"foo"},
				"balh": {"bar"},
			},
		},
		{
			name: "auth denied, only matched header to the client",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth",
				"authorization_response": {
					"allowed_client_headers": [
						{"exact": "foo"}
					]
				}
			}}`,
			server: func(r *http.Request) (*http.Response, error) {
				resp := response(401)
				resp.Header.Set("foo", "bar")
				resp.Header.Add("foo", "blah")
				resp.Header.Set("date", "now")
				return resp, nil
			},
			res: &api.LocalResponse{Code: 401,
				Header: http.Header(map[string][]string{
					"Foo": {"bar", "blah"},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			conf := &config{}
			protojson.Unmarshal([]byte(tt.input), conf)
			conf.Init(nil)
			patches := gomonkey.ApplyMethodFunc(conf.client, "Do", tt.server)
			defer patches.Reset()
			f := configFactory(conf)(cb)
			defaultHdr := map[string][]string{
				":authority": {"test.local"},
				":method":    {"DELETE"},
				":path":      {"/"},
			}
			for k, v := range tt.hdr {
				defaultHdr[k] = v
			}
			hdr := envoy.NewRequestHeaderMap(http.Header(defaultHdr))
			res := f.DecodeHeaders(hdr, true)
			if tt.res != nil {
				assert.Equal(t, tt.res, res)
			}

			for k, v := range tt.upHdr {
				assert.Equal(t, v, hdr.Values(k))
			}
		})
	}
}
