package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gohade/hade/framework/gin"
)

func TestNewRateLimit(t *testing.T) {
	r := NewRateLimit(WithRate(1), WithCap(3), WithWaitMaxDuration(5*time.Second))
	assert.Equal(t, float64(1), r.rate)
	assert.Equal(t, int64(3), r.cap)
	assert.Equal(t, 5*time.Second, r.waitMaxDuration)
}

func TestRateLimit(t *testing.T) {
	r := NewRateLimit(WithRate(1), WithCap(3))
	var ctx *gin.Context
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
		w := httptest.NewRecorder()
		ctx = &gin.Context{
			Request: req,
			Writer:  gin.NewResponseWriter(w),
		}
		r.Func()(ctx)
		if i < 3 {
			assert.Equal(t, http.StatusOK, ctx.Writer.Status())
		} else {
			assert.Equal(t, http.StatusTooManyRequests, ctx.Writer.Status())
		}
	}

	time.Sleep(10 * time.Second)

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	w := httptest.NewRecorder()
	ctx = &gin.Context{
		Request: req,
		Writer:  gin.NewResponseWriter(w),
	}
	r.Func()(ctx)

	assert.Equal(t, http.StatusOK, ctx.Writer.Status())
}

func TestRateLimitWithWaitMaxDuration(t *testing.T) {
	r := NewRateLimit(WithRate(1), WithCap(3), WithWaitMaxDuration(5*time.Second))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
		w := httptest.NewRecorder()
		ctx := &gin.Context{
			Request: req,
			Writer:  gin.NewResponseWriter(w),
		}
		r.Func()(ctx)

		if i < 8 {
			assert.Equal(t, http.StatusOK, ctx.Writer.Status())
		} else {
			assert.Equal(t, http.StatusTooManyRequests, ctx.Writer.Status())
		}

	}
}
