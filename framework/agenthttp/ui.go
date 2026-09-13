package agenthttp

import (
	"embed"
	"net/http"
	"path/filepath"

	"github.com/gohade/hade/framework/gin"
)

//go:embed ui/*
var uiFiles embed.FS

func serveUIFile(c *gin.Context, name, contentType string) {
	data, err := uiFiles.ReadFile("ui/" + name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	if name == "index.html" {
		c.Header("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'")
	}
	c.Data(http.StatusOK, contentType, data)
}

// ServeUI 返回 Agent 调试台入口。
func ServeUI(c *gin.Context) {
	serveUIFile(c, "index.html", "text/html; charset=utf-8")
}

// ServeUIAsset 返回编译进二进制的样式和脚本。
func ServeUIAsset(c *gin.Context) {
	name := filepath.Base(c.Request.URL.Path)
	contentType := "text/plain; charset=utf-8"
	if filepath.Ext(name) == ".css" {
		contentType = "text/css; charset=utf-8"
	} else if filepath.Ext(name) == ".js" {
		contentType = "text/javascript; charset=utf-8"
	}
	serveUIFile(c, name, contentType)
}
