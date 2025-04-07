package handler

import (
	"context"
	"datadog-proxy/config"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Config config.Config
	Logger *slog.Logger
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) proxyHandler(c *gin.Context) {
	h.Logger.DebugContext(c, "proxying request")
	ddforward := c.Query("ddforward")
	if ddforward == "" {
		err := "missing ddforward param"
		h.Logger.ErrorContext(c, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	h.Logger.DebugContext(c, "decoding params", "ddforward", ddforward)
	ddforwardDecoded, err := url.QueryUnescape(ddforward)
	if err != nil {
		message := "invalid ddforward encoding"
		h.Logger.ErrorContext(c, message, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	h.Logger.DebugContext(c, "ddforward decoded", "ddforwardDecoded", ddforwardDecoded)
	if !(strings.HasPrefix(ddforwardDecoded, "/api/v2/rum") || strings.HasPrefix(ddforwardDecoded, "/api/v2/replay") || strings.HasPrefix(ddforwardDecoded, "/api/v2/logs")) {
		err := "invalid target path"
		h.Logger.ErrorContext(c, err, "ddforwardDecoded", ddforwardDecoded)
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	targetURL := h.Config.DatadogBaseURL + ddforwardDecoded
	h.Logger.DebugContext(c, "creating request", "targetURL", targetURL)
	req, err := http.NewRequest("POST", targetURL, c.Request.Body)
	if err != nil {
		message := "failed to create request"
		h.Logger.ErrorContext(c, message, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}
	req.Header = c.Request.Header.Clone()

	h.Logger.DebugContext(c, "sending request", "targetURL", targetURL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.Logger.ErrorContext(c, "error sending request", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach Datadog", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	h.Logger.DebugContext(c, "received response, copying headers", "status", resp.Status)
	for k, v := range resp.Header {
		for _, val := range v {
			c.Writer.Header().Add(k, val)
		}
	}
	h.Logger.DebugContext(c, "returning status code and copying response body")
	c.Status(resp.StatusCode)
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		message := "failed to copy response body"
		h.Logger.ErrorContext(c, message, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": message, "details": err.Error()})
		return
	}
}

func (h *Handler) Run(ctx context.Context) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/healthz", healthHandler)
	router.POST("/dd", h.proxyHandler)

	h.Logger.InfoContext(ctx, "starting server", "config", h.Config)
	if err := router.Run(":8080"); err != nil {
		h.Logger.ErrorContext(ctx, "failed to start server", "error", err)
		os.Exit(1)
	}
	return nil
}
