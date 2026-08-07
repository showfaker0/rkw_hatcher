package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distEmbed embed.FS

// Register mounts SPA static files (call after /api routes).
func Register(r *gin.Engine) {
	sub, err := fs.Sub(distEmbed, "dist")
	if err != nil {
		return
	}
	staticFS := http.FS(sub)

	// 打包版：后端即本进程，启停接口占位，避免前端报错
	r.Any("/__boot_backend", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "already": true, "packaged": true})
	})
	r.Any("/__stop_backend", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":     false,
			"detail": "打包版请直接关闭程序窗口退出",
			"packaged": true,
		})
	})

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		name := strings.TrimPrefix(path, "/")
		if name != "" {
			if f, err := sub.Open(name); err == nil {
				_ = f.Close()
				http.FileServer(staticFS).ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// Vue history 路由回退
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "前端资源缺失，请重新执行 build-exe.ps1")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
