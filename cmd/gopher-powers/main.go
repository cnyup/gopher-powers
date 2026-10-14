// gopher-powers：大国崛起（Go 重写版）服务入口。
// 单二进制：本地进程承载模拟核心（单机打 AI）与联机对战（房间/快照），
// 前端静态资源经 go:embed 内嵌。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var version = "0.1.0-dev"

func main() {
	host := flag.String("host", "127.0.0.1", "监听地址")
	port := flag.Int("port", 4180, "监听端口")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, `{"ok":true,"version":%q,"protocol":2,"maxPlayers":2,"rooms":0,"battles":0}`, version)
	})

	addr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("gopher-powers %s 启动于 http://%s", version, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
	_ = os.Stdout
}
