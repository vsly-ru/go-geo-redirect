package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/vsly-ru/go-geo-redirect/pkg/geoip"
)

type Config struct {
	Main      map[string]string `toml:"main"`
	Redirects map[string]string `toml:"redirects"`
}

var (
	geoipService *geoip.GeoIPService
	config       Config
)

func LoadConfig(path string) (*Config, error) {
	log.Println("Reading config", path)
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	if config.Redirects == nil {
		return nil, errors.New("missing [redirects] section")
	}
	if config.Redirects["default"] == "" {
		return nil, errors.New("missing default url")
	}
	log.Println("Loaded redirects from the config:")
	for countryCode := range config.Redirects {
		log.Printf(`%-8s -> %s`, countryCode, config.Redirects[countryCode])
	}
	return &config, nil
}

func getIPFromRequest(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}
	ra := r.RemoteAddr
	return strings.Split(ra, ":")[0]
}

func getRedirectURL(geoData *geoip.GeoIPData, config *Config, originalURL *url.URL) (*url.URL, error) {
	originalPath := originalURL.Path
	query := originalURL.RawQuery
	var CountryCode string = "default"
	if geoData != nil {
		CountryCode = geoData.CountryCode
	}
	targetURL, ok := config.Redirects[CountryCode]
	if !ok {
		targetURL = config.Redirects["default"]
	}

	// Parse the target URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		msg := fmt.Sprintf("failed to parse target URL '%s': %v", targetURL, err)
		return nil, errors.New(msg)
	}

	// Append the original path (from the request)
	parsedURL.Path = originalPath

	// Append the original query string (if it exists)
	if query != "" {
		parsedURL.RawQuery = query
	}

	log.Printf("[\033[32m%s\033[0m] \033[33m%s\033[0m -> %s", CountryCode, originalURL.String(), parsedURL.String())
	return parsedURL, nil
}

func main() {
	// Initialize the GeoIP service
	service, err := geoip.NewGeoIPService(10000)
	if err != nil {
		log.Fatalf("Failed to initialize GeoIP service: %v", err)
	}
	geoipService = service

	// Load the configuration file
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to determine executable path: %v", err)
	}
	configPath := filepath.Join(filepath.Dir(execPath), "config.toml")
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	mode := "redirect"
	if u := config.Main["use"]; u != "" {
		mode = u
	}

	if mode == "redirect" {
		http.HandleFunc("/", redirectHandler)
	}
	if mode == "proxy" {
		proxy := &httputil.ReverseProxy{}
		http.HandleFunc("/", proxyHandler(proxy))
	}

	// Start the HTTP server
	addr := ":8302"
	if a := config.Main["addr"]; a != "" {
		addr = a
	}

	log.Printf("Running \033[32m%s\033[0m server on %s", mode, addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	ip := getIPFromRequest(r)
	geoData, err := geoipService.GetGeoIPData(ip)
	if err != nil {
		log.Printf("Error getting GeoIP data for IP %s: %v", ip, err)
	}

	redirectURL, err := getRedirectURL(geoData, &config, r.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func proxyHandler(proxy *httputil.ReverseProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getIPFromRequest(r)
		geoData, err := geoipService.GetGeoIPData(ip)
		if err != nil {
			log.Printf("Error getting GeoIP data for IP %s: %v", ip, err)
		}

		proxyURL, err := getRedirectURL(geoData, &config, r.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update the proxy target with the new URL
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = proxyURL.Scheme
			req.URL.Host = proxyURL.Host
			req.URL.Path = proxyURL.Path
			req.URL.RawQuery = proxyURL.RawQuery
			req.Host = req.URL.Host
			// log.Printf("Proxying request to %s", req.URL.String())
		}

		// Serve the proxied request
		proxy.ServeHTTP(w, r)
	}
}
