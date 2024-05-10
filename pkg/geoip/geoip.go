package geoip

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

type GeoIPData struct {
	IP               string `json:"ip"`
	Country          string `json:"country"`
	CountryCode      string `json:"country_code"`
	Region           string `json:"region"`
	City             string `json:"city"`
	PostalCode       string `json:"postal_code"`
	Timezone         string `json:"timezone"`
	Organization     string `json:"organization"`
	OrganizationName string `json:"organization_name"`
}

type GeoIPService struct {
	cache *lru.Cache
	mu    sync.RWMutex
}

// NewGeoIPService creates a new GeoIP service with a given cache size
func NewGeoIPService(cacheSize int) (*GeoIPService, error) {
	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}
	return &GeoIPService{
		cache: cache,
	}, nil
}

// GetGeoIPData fetches the geographical information for a given IP address
func (s *GeoIPService) GetGeoIPData(ip string) (*GeoIPData, error) {
	if ip == "" {
		return nil, errors.New("invalid IP address")
	}
	// Check if the data is in the cache
	s.mu.RLock()
	if data, ok := s.cache.Get(ip); ok {
		return data.(*GeoIPData), nil
	}
	s.mu.RUnlock()

	// Data is not in the cache, fetch from the API
	url := fmt.Sprintf("https://get.geojs.io/v1/ip/geo/%s.json", ip)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var geoData GeoIPData
	if err := json.NewDecoder(resp.Body).Decode(&geoData); err != nil {
		return nil, err
	}
	if geoData.CountryCode == "" {
		return nil, errors.New("received empty country code")
	}

	// Add the fetched data to the cache
	go func() {
		s.mu.Lock()
		s.cache.Add(ip, &geoData)
		s.mu.Unlock()
	}()

	return &geoData, nil
}
