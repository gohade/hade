package middleware

import (
	"net/http"
	"strings"

	"github.com/gohade/hade/framework/gin"
)

const (
	allowOrigin      = "Access-Control-Allow-Origin"
	allowMethods     = "Access-Control-Allow-Methods"
	allowHeaders     = "Access-Control-Allow-Headers"
	allowCredentials = "Access-Control-Allow-Credentials"
	exposeHeaders    = "Access-Control-Expose-Headers"
	requestMethod    = "Access-Control-Request-Method"
	requestHeaders   = "Access-Control-Request-Headers"
	maxAgeHeader     = "Access-Control-Max-Age"
	varyHeader       = "Vary"
	originHeader     = "Origin"
)

// defaultCORS default cors
var defaultCORS = CORS{
	allowCredentials: false,
	allowHeaders:     []string{"Content-Type", "Origin", "X-CSRF-Token", "Authorization", "AccessToken", "Token", "Range"},
	maxAge:           "86400",
	allowOrigins:     []string{"*"},
	exposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers"},
	allowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete},
}

// CORS cors
type CORS struct {
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	allowCredentials bool
	maxAge           string
}

// CORSOption ...
type CORSOption func(c *CORS)

// WithAllowOrigins set AllowOrigins
func WithAllowOrigins(allowOrigins []string) CORSOption {
	return func(c *CORS) {
		c.allowOrigins = allowOrigins
	}
}

// WithAllowHeaders set AllowHeaders
func WithAllowHeaders(allowHeaders []string) CORSOption {
	return func(c *CORS) {
		c.allowHeaders = allowHeaders
	}
}

// WithAllowMethods set AllowMethods
func WithAllowMethods(allowMethods []string) CORSOption {
	return func(c *CORS) {
		c.allowMethods = allowMethods
	}
}

// WithExposeHeaders set ExposeHeaders
func WithExposeHeaders(exposeHeaders []string) CORSOption {
	return func(c *CORS) {
		c.exposeHeaders = exposeHeaders
	}
}

// WithAllowCredentials set AllowCredentials
func WithAllowCredentials(allowCredentials bool) CORSOption {
	return func(c *CORS) {
		c.allowCredentials = allowCredentials
	}
}

// WithMaxAge set MaxAge
func WithMaxAge(maxAge string) CORSOption {
	return func(c *CORS) {
		c.maxAge = maxAge
	}
}

// NewCORS ...
func NewCORS(opts ...CORSOption) *CORS {
	cors := defaultCORS
	for _, opt := range opts {
		opt(&cors)
	}

	return &cors
}

// Func ...
func (m *CORS) Func() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Add(varyHeader, originHeader)
		if c.Request.Method == http.MethodOptions {
			c.Writer.Header().Add(varyHeader, requestMethod)
			c.Writer.Header().Add(varyHeader, requestHeaders)
		}

		origin := c.Request.Header.Get(originHeader)
		if b, allowOrigins := isOriginAllowed(m.allowOrigins, origin); b {
			m.setCORSHeader(c, allowOrigins)
		}

		if c.Request.Method == http.MethodOptions {
			c.JSON(http.StatusNoContent, nil)

			return
		}

		c.Next()
	}
}

// setCORSHeader set cors header
func (m *CORS) setCORSHeader(c *gin.Context, origins []string) {
	c.Writer.Header().Set(allowOrigin, strings.Join(origins, ","))
	c.Writer.Header().Set(allowMethods, strings.Join(m.allowMethods, ","))
	c.Writer.Header().Set(allowHeaders, strings.Join(m.allowHeaders, ","))
	c.Writer.Header().Set(exposeHeaders, strings.Join(m.exposeHeaders, ","))
	if m.allowCredentials {
		c.Writer.Header().Set(allowCredentials, "true")
	}

	if m.maxAge != "0" {
		c.Writer.Header().Set(maxAgeHeader, m.maxAge)
	}
}

// isOriginAllowed ...
func isOriginAllowed(allows []string, origin string) (bool, []string) {
	for _, o := range allows {
		if o == "*" {
			return true, []string{"*"}
		}

		if o == origin {
			return true, []string{origin}
		}
	}

	return false, []string{}
}
