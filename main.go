package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	// Parse command line arguments
	var (
		listenAddr = flag.String("listen", ":8000", "Address to listen on")
		targetURL  = flag.String("target", "", "Target HTTPS server URL (required)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging of requests and responses")
		insecure   = flag.Bool("insecure", false, "Skip TLS certificate verification")
		timeout    = flag.Int("timeout", 30, "Request timeout in seconds")
	)

	flag.Parse()

	// Validate required arguments
	if *targetURL == "" {
		fmt.Println("Error: target URL is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the target URL
	target, err := url.Parse(*targetURL)
	if err != nil {
		log.Fatalf("Error parsing target URL: %v", err)
	}

	// Check if target is HTTPS
	if target.Scheme != "https" {
		log.Printf("Warning: Target URL scheme is %s, not https", target.Scheme)
	}

	// Configure logging
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Create a custom transport with configurable TLS
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: *insecure,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Create a reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom director to modify the request before sending to the target
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		
		// Explicitly set the Host header to match the target's host
		req.Host = target.Host
		
		// Log request details if verbose
		if *verbose {
			logRequest(logger, req)
		} else {
			logger.Printf("%s %s -> %s (Host: %s)", req.Method, req.URL.Path, target.String()+req.URL.Path, req.Host)
		}
	}

	// Set the custom transport
	proxy.Transport = transport

	// Add error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Printf("Error proxying request: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Error proxying request: %v", err)
	}

	// Add response modification through a custom handler
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For verbose logging, we use a custom response writer that captures the response
		if *verbose {
			// Create a response recorder
			recorder := newResponseRecorder(w)
			
			// Process the request with our recorder
			proxy.ServeHTTP(recorder, r)
			
			// Log the response
			logResponse(logger, recorder)
		} else {
			// Standard handling without verbose logging
			proxy.ServeHTTP(w, r)
		}
	})

	// Start the server
	logger.Printf("Starting proxy server on %s forwarding to %s", *listenAddr, *targetURL)
	if *insecure {
		logger.Printf("WARNING: TLS certificate verification disabled (insecure mode)")
	}

	server := &http.Server{
		Addr:         *listenAddr,
		Handler:      proxyHandler,
		ReadTimeout:  time.Duration(*timeout) * time.Second,
		WriteTimeout: time.Duration(*timeout) * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Error starting proxy server: %v", err)
	}
}

// logRequest logs detailed information about the request
func logRequest(logger *log.Logger, req *http.Request) {
	// Convert request to dump
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		logger.Printf("Error dumping request: %v", err)
		return
	}
	
	// Log with formatting
	logger.Printf("\n%s[REQUEST]%s\n%s\n%s[END REQUEST]%s\n",
		strings.Repeat("=", 30),
		strings.Repeat("=", 30),
		string(dump),
		strings.Repeat("=", 30),
		strings.Repeat("=", 30))
}

// logResponse logs detailed information about the response
func logResponse(logger *log.Logger, recorder *responseRecorder) {
	logger.Printf("\n%s[RESPONSE: %d]%s\n",
		strings.Repeat("=", 30),
		recorder.statusCode,
		strings.Repeat("=", 30))
	
	// Log headers
	logger.Println("Headers:")
	for key, values := range recorder.Header() {
		for _, value := range values {
			logger.Printf("  %s: %s", key, value)
		}
	}
	
	// Log body if exists (truncate if too large)
	if len(recorder.body) > 0 {
		maxBodyLogSize := 1000
		body := recorder.body
		if len(body) > maxBodyLogSize {
			logger.Printf("\nBody (truncated to %d bytes):\n%s... [truncated]", 
				maxBodyLogSize, body[:maxBodyLogSize])
		} else {
			logger.Printf("\nBody:\n%s", body)
		}
	}
	
	logger.Printf("\n%s[END RESPONSE]%s\n",
		strings.Repeat("=", 30),
		strings.Repeat("=", 30))
}

// responseRecorder is a custom ResponseWriter that captures the response
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

// newResponseRecorder creates a new response recorder
func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default status
	}
}

// WriteHeader captures the status code
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the body
func (r *responseRecorder) Write(b []byte) (int, error) {
	// Append to our copy of the body
	r.body = append(r.body, b...)
	// Write to the underlying ResponseWriter
	return r.ResponseWriter.Write(b)
}