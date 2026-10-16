// gopher-powers：大国崛起（Go 重写版）服务入口。
// 单二进制：本地进程承载模拟核心（单机打 AI）与联机对战（房间/快照），
// 前端静态资源 M3 起经 go:embed 内嵌；当前 -root 指向前端构建产物目录。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cnyup/gopher-powers/internal/server"
)

var version = "0.2.0-dev"

func main() {
	host := flag.String("host", "127.0.0.1", "监听地址")
	port := flag.Int("port", 4180, "监听端口")
	root := flag.String("root", "dist", "前端静态资源目录（旧 JS 前端或未来 Vue 构建）")
	dir := flag.String("data", "server-data", "账户数据目录")
	dev := flag.Bool("dev", true, "开发模式（放行 localhost:4173 预览源）")
	flag.Parse()

	core, err := server.New(server.Config{Directory: *dir, MaxPlayers: 2})
	if err != nil {
		log.Fatal(err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go core.StartTicker(stop)

	wsLayer := server.NewWSServer(core, "", *dev)
	rootAbs, _ := filepath.Abs(*root)

	mux := http.NewServeMux()
	mux.Handle("/api/", core)
	mux.HandleFunc("/battle-socket", wsLayer.ServeWS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveStatic(w, r, rootAbs)
	})

	addr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("gopher-powers %s 启动于 http://%s（root=%s）", version, addr, rootAbs)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// serveStatic 对齐 static-files.mjs 核心：穿越防护 + 目录 → index.html。
func serveStatic(w http.ResponseWriter, r *http.Request, root string) {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	full := filepath.Join(root, filepath.FromSlash(name))
	absRoot, _ := filepath.Abs(root)
	abs, err := filepath.Abs(full)
	if err != nil || (abs != absRoot && !strings.HasPrefix(abs, absRoot+string(filepath.Separator))) {
		http.Error(w, `{"error":"禁止访问"}`, http.StatusForbidden)
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		http.Error(w, `{"error":"文件不存在"}`, http.StatusNotFound)
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		http.Error(w, `{"error":"文件不存在"}`, http.StatusNotFound)
		return
	}
	defer f.Close()
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
