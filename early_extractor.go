package requestbody

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/pucora/lura/v2/config"
)

const EarlyExtractorNamespace = "modifier/request-body-extractor/early"

// HandlerFactory creates an early router middleware to extract body fields to headers/query.
// It parses the Go-gin Context before auth middlewares run.
func HandlerFactory(cfg *config.EndpointConfig) gin.HandlerFunc {
	v, ok := cfg.ExtraConfig[EarlyExtractorNamespace]
	if !ok {
		return nil
	}

	cfgM, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}

	headersCfg, headersOk := cfgM["headers"].(map[string]interface{})
	querystringCfg, querystringOk := cfgM["querystring"].(map[string]interface{})

	if !headersOk && !querystringOk {
		return nil
	}

	return func(c *gin.Context) {
		if c.Request.Body == nil {
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			var parsed map[string]interface{}
			if json.Unmarshal(bodyBytes, &parsed) == nil {
				if headersOk {
					for bodyField, headerName := range headersCfg {
						if v, ok := parsed[bodyField]; ok {
							if strV, ok := v.(string); ok {
								c.Request.Header.Add(headerName.(string), strV)
							}
						}
					}
				}

				if querystringOk {
					q := c.Request.URL.Query()
					modified := false
					for bodyField, paramName := range querystringCfg {
						if v, ok := parsed[bodyField]; ok {
							if strV, ok := v.(string); ok {
								q.Add(paramName.(string), strV)
								modified = true
							}
						}
					}
					if modified {
						c.Request.URL.RawQuery = q.Encode()
					}
				}
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		c.Next()
	}
}
