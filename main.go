package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	// Parse command line arguments
	targetServer := flag.String("target", "http://localhost:8080", "Target server URL")
	listenAddr := flag.String("listen", ":8000", "Address to listen on")
	flag.Parse()

	// Parse the target URL
	target, err := url.Parse(*targetServer)
	if err != nil {
		log.Fatalf("Error parsing target URL: %v", err)
	}

	// Create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Log requests
	handler := logHandler(proxy)

	// Start proxy server
	fmt.Printf("Starting proxy server on %s forwarding to %s\n", *listenAddr, *targetServer)
	if err := http.ListenAndServe(*listenAddr, handler); err != nil {
		log.Fatalf("Error starting proxy server: %v", err)
	}
}

// logHandler wraps the proxy handler with logging
func logHandler(proxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Proxying request: %s %s\n", r.Method, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})
}