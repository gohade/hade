package kernel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gohade/hade/framework/gin"
	"google.golang.org/grpc"
)

func TestHadeKernelService_AgentEngine(t *testing.T) {
	e := gin.New()
	e.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	s, err := NewHadeKernelService(e, grpc.NewServer(), e)
	if err != nil {
		t.Fatal(err)
	}
	ks := s.(*HadeKernelService)
	w := httptest.NewRecorder()
	ks.AgentEngine().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Body.String() != "pong" {
		t.Fatalf("got %q", w.Body.String())
	}
}
