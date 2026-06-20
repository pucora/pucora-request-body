package requestbody_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/pucora/lura/v2/config"
	"github.com/pucora/lura/v2/proxy"
	requestbody "github.com/pucora/pucora-request-body"
)

func TestProxyFactory(t *testing.T) {
	next := proxy.FactoryFunc(func(_ *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, req *proxy.Request) (*proxy.Response, error) {
			return &proxy.Response{
				Data: map[string]interface{}{
					"extracted_header": req.Headers["X-Custom-Header"][0],
				},
				IsComplete: true,
			}, nil
		}, nil
	})

	cfg := &config.EndpointConfig{
		ExtraConfig: map[string]interface{}{
			requestbody.ExtractorNamespace: map[string]interface{}{
				"headers": map[string]interface{}{
					"fieldToExtract": "X-Custom-Header",
				},
			},
		},
	}

	pf := requestbody.ProxyFactory(next)
	p, err := pf.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"fieldToExtract": "extracted-value"}`)
	req := &proxy.Request{
		Body:    io.NopCloser(bytes.NewBuffer(body)),
		Headers: make(map[string][]string),
	}

	resp, err := p(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Data["extracted_header"] != "extracted-value" {
		t.Errorf("failed to extract header from body, got: %v", resp.Data)
	}
}
