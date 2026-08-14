package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"rkw_hatcher/internal/config"
	"rkw_hatcher/internal/db"
	"rkw_hatcher/internal/handlers"
	"rkw_hatcher/internal/singleinstance"
	"rkw_hatcher/web"
)

func main() {
	cfg := config.Load()

	lockDir := filepath.Dir(cfg.DBPath)
	releaseLock, err := singleinstance.Acquire(lockDir)
	if err != nil {
		fatalf("单实例锁失败: %v", err)
	}
	defer releaseLock()

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		fatalf("打开数据库失败: %v\n\n数据库文件: %s", err, cfg.DBPath)
	}
	defer conn.Close()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	api := &handlers.API{DB: conn}
	api.Register(r)
	web.Register(r)

	addr := cfg.HTTPAddr
	url := openURL(addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fatalf("端口监听失败 (%s): %v\n\n请确认端口未被占用。", addr, err)
	}

	log.Printf("db: %s", cfg.DBPath)
	log.Printf("listening on %s  →  %s", addr, url)
	log.Printf("关闭本窗口即可退出")
	if os.Getenv("RKW_NO_BROWSER") == "" {
		go func() {
			time.Sleep(300 * time.Millisecond)
			_ = openBrowser(url)
		}()
	}

	srv := &http.Server{Handler: r}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fatalf("server failed: %v", err)
	}
}

func fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Println(msg)
	fmt.Fprintln(os.Stderr, msg)
	if stdinIsTTY() {
		fmt.Println()
		fmt.Println("按回车键退出…")
		_, _ = fmt.Scanln()
	}
	os.Exit(1)
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func openURL(addr string) string {
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	return "http://" + host
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
