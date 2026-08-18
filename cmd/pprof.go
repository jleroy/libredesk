package main

import (
	"net/http"
	"net/http/pprof"

	"github.com/abhinavxd/libredesk/internal/colorlog"
)

// startPprof starts the net/http/pprof server on its own address if enabled in config.
func startPprof() {
	if !ko.Bool("app.pprof.enabled") {
		return
	}

	addr := ko.String("app.pprof.address")
	if addr == "" {
		addr = "127.0.0.1:6060"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		colorlog.Green("pprof server started at %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			colorlog.Red("error starting pprof server: %v", err)
		}
	}()
}
