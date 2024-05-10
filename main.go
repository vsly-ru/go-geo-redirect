package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/vsly-ru/go-geo-redirect/pkg/geoip"
)

type Config struct {
	Redirects map[string]string `toml:"redirects"`
}

func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
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

func getRedirectURL(geoData *geoip.GeoIPData, config *Config, originalURL *url.URL) (string, error) {
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
		return "", errors.New(msg)
	}

	// Append the original path (from the request)
	parsedURL.Path = originalPath

	// Append the original query string (if it exists)
	if query != "" {
		parsedURL.RawQuery = query
	}

	result := parsedURL.String()
	log.Printf("[\033[32m%s\033[0m] %s -> %s", CountryCode, originalURL.String(), result)
	return result, nil
}

func main() {
	// Initialize the GeoIP service
	service, err := geoip.NewGeoIPService(10000)
	if err != nil {
		log.Fatalf("Failed to initialize GeoIP service: %v", err)
	}

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

	// Define the HTTP server handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := getIPFromRequest(r)
		geoData, err := service.GetGeoIPData(ip)
		if err != nil {
			log.Printf("Error getting GeoIP data for IP %s: %v", ip, err)
		}

		redirectURL, err := getRedirectURL(geoData, config, r.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	// Start the HTTP server
	port := ":8302"
	log.Printf("Running server. Test url: http://127.0.0.1%s/example?q=42", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
