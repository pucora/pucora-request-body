package requestbody

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/url"

	"github.com/pucora/lura/v2/config"
	"github.com/pucora/lura/v2/proxy"
)

const ExtractorNamespace = "modifier/request-body-extractor"
const GeneratorNamespace = "modifier/request-body-generator"

// ProxyFactory returns a factory that modifies request body based on configuration
func ProxyFactory(next proxy.Factory) proxy.Factory {
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		nextProxy, err := next.New(cfg)
		if err != nil {
			return proxy.NoopProxy, err
		}

		hasExtractor := hasNamespace(cfg.ExtraConfig, ExtractorNamespace)
		hasGenerator := hasNamespace(cfg.ExtraConfig, GeneratorNamespace)

		if !hasExtractor && !hasGenerator {
			return nextProxy, nil
		}

		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			if req.Body == nil {
				return nextProxy(ctx, req)
			}

			if hasExtractor {
				bodyBytes, err := io.ReadAll(req.Body)
				if err == nil && len(bodyBytes) > 0 {
					var parsed map[string]interface{}
					if json.Unmarshal(bodyBytes, &parsed) == nil {
						cfgM, _ := cfg.ExtraConfig[ExtractorNamespace].(map[string]interface{})
						// naive extractor: copy field -> header or param
						if headers, ok := cfgM["headers"].(map[string]interface{}); ok {
							for bodyField, headerName := range headers {
								if v, ok := parsed[bodyField]; ok {
									if strV, ok := v.(string); ok {
										if req.Headers == nil {
											req.Headers = make(map[string][]string)
										}
										req.Headers[headerName.(string)] = []string{strV}
									}
								}
							}
						}
						
						if params, ok := cfgM["querystring"].(map[string]interface{}); ok {
							if req.Query == nil {
								req.Query = url.Values{}
							}
							for bodyField, paramName := range params {
								if v, ok := parsed[bodyField]; ok {
									if strV, ok := v.(string); ok {
										req.Query.Add(paramName.(string), strV)
									}
								}
							}
						}
					}
					req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			if hasGenerator {
				// example of generator: inject static payload or overwrite
				if gCfg, ok := cfg.ExtraConfig[GeneratorNamespace].(map[string]interface{}); ok {
					if staticBody, ok := gCfg["static"].(map[string]interface{}); ok {
						newBody, _ := json.Marshal(staticBody)
						req.Body = io.NopCloser(bytes.NewBuffer(newBody))
					}
				}
			}

			return nextProxy(ctx, req)
		}, nil
	})
}

// BackendFactory provides similar logic at the backend level
func BackendFactory(next proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		nextProxy := next(cfg)

		hasExtractor := hasNamespace(cfg.ExtraConfig, ExtractorNamespace)
		hasGenerator := hasNamespace(cfg.ExtraConfig, GeneratorNamespace)

		if !hasExtractor && !hasGenerator {
			return nextProxy
		}

		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			if req.Body == nil && !hasGenerator {
				return nextProxy(ctx, req)
			}

			if hasExtractor && req.Body != nil {
				bodyBytes, err := io.ReadAll(req.Body)
				if err == nil && len(bodyBytes) > 0 {
					var parsed map[string]interface{}
					if json.Unmarshal(bodyBytes, &parsed) == nil {
						cfgM, _ := cfg.ExtraConfig[ExtractorNamespace].(map[string]interface{})
						if headers, ok := cfgM["headers"].(map[string]interface{}); ok {
							for bodyField, headerName := range headers {
								if v, ok := parsed[bodyField]; ok {
									if strV, ok := v.(string); ok {
										if req.Headers == nil {
											req.Headers = make(map[string][]string)
										}
										req.Headers[headerName.(string)] = []string{strV}
									}
								}
							}
						}
					}
					req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			if hasGenerator {
				if gCfg, ok := cfg.ExtraConfig[GeneratorNamespace].(map[string]interface{}); ok {
					if staticBody, ok := gCfg["static"].(map[string]interface{}); ok {
						newBody, _ := json.Marshal(staticBody)
						req.Body = io.NopCloser(bytes.NewBuffer(newBody))
					}
				}
			}

			return nextProxy(ctx, req)
		}
	}
}

func hasNamespace(cfg config.ExtraConfig, namespace string) bool {
	if cfg == nil {
		return false
	}
	_, ok := cfg[namespace]
	return ok
}
