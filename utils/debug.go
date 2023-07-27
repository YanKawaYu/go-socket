package utils

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
)

/**
启动Go routine、内存、CPU监控
Go routine：http://localhost:port/go_routine
CPU：go tool pprof http://localhost:port/debug/pprof/profile?seconds=30
	 web
内存：go tool pprof http://localhost:port/debug/pprof/heap
	 web
*/
func GoPpf() {
	GoPpfByPort(os.Getpid())
}

func GoPpfByPort(port int) {
	http.HandleFunc("/go_routine", handler)
	addr := fmt.Sprintf(":%d", port)
	http.ListenAndServe(addr, nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	p := pprof.Lookup("goroutine")
	p.WriteTo(w, 1)
}
