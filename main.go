package main

import (
	"crypto/tls"
	"encoding/json"
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

// ProxyConfig holds configuration for a single proxy target
type ProxyConfig struct {
	Path        string `json:"path"`        // URL path to match
	TargetURL   string `json:"targetUrl"`   // Target HTTPS server URL
	Insecure    bool   `json:"insecure"`    // Skip TLS certificate verification
	StripPrefix bool   `json:"stripPrefix"` // Strip the path prefix before forwarding
}

// ProxyConfigs holds multiple proxy configurations
type ProxyConfigs struct {
	Listen     string        `json:"listen"`     // Address to listen on
	Timeout    int           `json:"timeout"`    // Request timeout in seconds
	Verbose    bool          `json:"verbose"`    // Enable verbose logging
	Targets    []ProxyConfig `json:"targets"`    // Target configurations
	ConfigFile string        `json:"-"`          // Not serialized, just for CLI
}

// ProxyTarget holds a target configuration and its handler
type ProxyTarget struct {
	Config  ProxyConfig
	Handler http.Handler
	Index   int
}

// CustomRouter is a router that can handle path prefixes properly
type CustomRouter struct {
	Targets  []ProxyTarget
	Fallback http.Handler
	Logger   *log.Logger
}

// ServeHTTP implements the http.Handler interface for our custom router
func (r *CustomRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Find the most specific matching target
	var matchedTarget *ProxyTarget
	matchedPathLen := 0

	for i := range r.Targets {
		target := &r.Targets[i]
		
		// Check if the request path starts with this target's path
		if strings.HasPrefix(req.URL.Path, target.Config.Path) {
			// Check if this is a more specific match than what we've found so far
			pathLen := len(target.Config.Path)
			if pathLen > matchedPathLen {
				matchedTarget = target
				matchedPathLen = pathLen
			}
		}
	}

	if matchedTarget != nil {
		matchedTarget.Handler.ServeHTTP(w, req)
	} else if r.Fallback != nil {
		r.Fallback.ServeHTTP(w, req)
	} else {
		// No match found
		r.Logger.Printf("No target configured for path: %s", req.URL.Path)
		http.NotFound(w, req)
	}
}

func main() {
	// Parse command line arguments
	configs := ProxyConfigs{
		Listen:  ":8000",
		Timeout: 30,
		Verbose: false,
	}

	// Support both legacy single-target mode and new multi-target mode
	var (
		listenAddr = flag.String("listen", ":8000", "Address to listen on")
		targetURL  = flag.String("target", "", "Target HTTPS server URL (legacy mode)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging of requests and responses")
		insecure   = flag.Bool("insecure", false, "Skip TLS certificate verification (legacy mode)")
		timeout    = flag.Int("timeout", 30, "Request timeout in seconds")
		configFile = flag.String("config", "", "Path to JSON config file for multi-target mode")
	)

	flag.Parse()

	// Configure logging
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Determine if we're using multi-target or legacy mode
	if *configFile != "" {
		// Multi-target mode with config file
		configs.ConfigFile = *configFile
		configs.Listen = *listenAddr
		configs.Timeout = *timeout
		configs.Verbose = *verbose

		// Read config from file
		configData, err := os.ReadFile(*configFile)
		if err != nil {
			logger.Fatalf("Error reading config file: %v", err)
		}

		if err := json.Unmarshal(configData, &configs); err != nil {
			logger.Fatalf("Error parsing config file: %v", err)
		}

		// Validate configuration
		if len(configs.Targets) == 0 {
			logger.Fatalf("No targets defined in config file")
		}
	} else if *targetURL != "" {
		// Legacy single-target mode
		configs.Targets = []ProxyConfig{
			{
				Path:        "/",
				TargetURL:   *targetURL,
				Insecure:    *insecure,
				StripPrefix: false,
			},
		}
	} else {
		// No config provided
		fmt.Println("Error: either target URL or config file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Create our custom router
	router := &CustomRouter{
		Targets: []ProxyTarget{},
		Logger: logger,
	}

	// Set up each proxy target
	for i, targetConfig := range configs.Targets {
		// Parse the target URL
		parsedURL, err := url.Parse(targetConfig.TargetURL)
		if err != nil {
			logger.Fatalf("Error parsing target URL %s: %v", targetConfig.TargetURL, err)
		}

		// Check if target is HTTPS
		if parsedURL.Scheme != "https" {
			logger.Printf("Warning: Target URL scheme is %s, not https for path %s", 
				parsedURL.Scheme, targetConfig.Path)
		}

		// Create a custom transport with configurable TLS
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: targetConfig.Insecure,
			},
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     60 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		// Create a reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(parsedURL)

		// Custom director to modify the request before sending to the target
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			// Save the path for logging purposes
			originalPath := req.URL.Path

			// Optionally strip the prefix if configured
			if targetConfig.StripPrefix && targetConfig.Path != "/" {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, targetConfig.Path)
				if !strings.HasPrefix(req.URL.Path, "/") {
					req.URL.Path = "/" + req.URL.Path
				}
			}

			originalDirector(req)
			
			// Explicitly set the Host header to match the target's host
			req.Host = parsedURL.Host
			
			// Log request details if verbose
			if configs.Verbose {
				logRequest(logger, req, fmt.Sprintf("[Target %d]", i+1))
			} else {
				logger.Printf("[Target %d] %s %s -> %s (Host: %s, Original Path: %s)", 
					i+1, req.Method, req.URL.Path, parsedURL.String()+req.URL.Path, req.Host, originalPath)
			}
		}

		// Set the custom transport
		proxy.Transport = transport

		// Add error handler
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Printf("[Target %d] Error proxying request: %v", i+1, err)
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, "Error proxying request: %v", err)
		}

		// Add response modification through a custom handler
		proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For verbose logging, we use a custom response writer that captures the response
			if configs.Verbose {
				// Create a response recorder
				recorder := newResponseRecorder(w)
				
				// Process the request with our recorder
				proxy.ServeHTTP(recorder, r)
				
				// Log the response
				logResponse(logger, recorder, fmt.Sprintf("[Target %d]", i+1))
			} else {
				// Standard handling without verbose logging
				proxy.ServeHTTP(w, r)
			}
		})

		// Add this target to our router
		router.Targets = append(router.Targets, ProxyTarget{
			Config:  targetConfig,
			Handler: proxyHandler,
			Index:   i,
		})

		logger.Printf("Configured proxy for path %s (and subpaths) -> %s (Insecure: %v, StripPrefix: %v)", 
			targetConfig.Path, targetConfig.TargetURL, targetConfig.Insecure, targetConfig.StripPrefix)
	}

	// Start the server
	logger.Printf("Starting proxy server on %s with %d targets", configs.Listen, len(configs.Targets))
	
	server := &http.Server{
		Addr:         configs.Listen,
		Handler:      router,
		ReadTimeout:  time.Duration(configs.Timeout) * time.Second,
		WriteTimeout: time.Duration(configs.Timeout) * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("Error starting proxy server: %v", err)
	}
}

// logRequest logs detailed information about the request
func logRequest(logger *log.Logger, req *http.Request, prefix ...string) {
	// Get prefix if provided
	prefixStr := ""
	if len(prefix) > 0 {
		prefixStr = prefix[0] + " "
	}

	// Convert request to dump
	dump, err := httputil.DumpRequest(req, true)
	if err != nil {
		logger.Printf("%sError dumping request: %v", prefixStr, err)
		return
	}
	
	// Log with formatting
	logger.Printf("\n%s%s[REQUEST]%s\n%s\n%s[END REQUEST]%s\n",
		prefixStr,
		strings.Repeat("=", 30),
		strings.Repeat("=", 30),
		string(dump),
		strings.Repeat("=", 30),
		strings.Repeat("=", 30))
}

// logResponse logs detailed information about the response
func logResponse(logger *log.Logger, recorder *responseRecorder, prefix ...string) {
	// Get prefix if provided
	prefixStr := ""
	if len(prefix) > 0 {
		prefixStr = prefix[0] + " "
	}

	logger.Printf("\n%s%s[RESPONSE: %d]%s\n",
		prefixStr,
		strings.Repeat("=", 30),
		recorder.statusCode,
		strings.Repeat("=", 30))
	
	// Log headers
	logger.Println(prefixStr + "Headers:")
	for key, values := range recorder.Header() {
		for _, value := range values {
			logger.Printf("%s  %s: %s", prefixStr, key, value)
		}
	}
	
	// Log body if exists (truncate if too large)
	if len(recorder.body) > 0 {
		maxBodyLogSize := 1000
		body := recorder.body
		if len(body) > maxBodyLogSize {
			logger.Printf("\n%sBody (truncated to %d bytes):\n%s... [truncated]", 
				prefixStr, maxBodyLogSize, body[:maxBodyLogSize])
		} else {
			logger.Printf("\n%sBody:\n%s", prefixStr, body)
		}
	}
	
	logger.Printf("\n%s%s[END RESPONSE]%s\n",
		prefixStr,
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