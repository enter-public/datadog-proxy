package handler

import (
	"datadog-proxy/config"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(h Handler) *gin.Engine {
	r := gin.Default()
	r.GET("/health", healthHandler)
	r.POST("/proxy", h.proxyHandler)
	return r
}

func TestHealthHandler(t *testing.T) {
	r := gin.Default()
	r.GET("/health", healthHandler)

	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "ok") {
		t.Errorf("Unexpected body: %s", resp.Body.String())
	}
}

func TestProxyHandler_MissingParam(t *testing.T) {
	h := Handler{
		Config: config.Config{},
		Logger: slog.Default(),
	}
	r := setupRouter(h)

	req, _ := http.NewRequest("POST", "/proxy", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing ddforward, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "missing ddforward") {
		t.Errorf("Unexpected body: %s", resp.Body.String())
	}
}

func TestProxyHandler_InvalidPath(t *testing.T) {
	encoded := url.QueryEscape("/invalid/path")
	h := Handler{
		Config: config.Config{},
		Logger: slog.Default(),
	}
	r := setupRouter(h)

	req, _ := http.NewRequest("POST", "/proxy?ddforward="+encoded, nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid path, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "invalid target path") {
		t.Errorf("Unexpected body: %s", resp.Body.String())
	}
}

func TestProxyHandler_ValidRUMRequest(t *testing.T) {
	// Create a fake upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/rum" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	// Extract base URL and rewrite the config to point to the fake server
	u, _ := url.Parse(upstream.URL)
	h := Handler{
		Config: config.Config{
			DatadogBaseURL: u.Scheme + "://" + u.Host,
		},
		Logger: slog.Default(),
	}
	r := setupRouter(h)

	encoded := url.QueryEscape("/api/v2/rum")
	req, _ := http.NewRequest("POST", "/proxy?ddforward="+encoded, strings.NewReader(`{"test": "data"}`))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusAccepted {
		t.Errorf("Expected 202 from upstream, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "ok") {
		t.Errorf("Unexpected body: %s", resp.Body.String())
	}
}
