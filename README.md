# go-geo-redirect
A simple redirect app written in go. It listens for HTTP requests with any URL. When one occurs, it determines geoip data of the original IP of the request. Based on the country from geoip response, it redirects the request(using HTTP 302 code) to the same URL and query of a preconfigured domain, based on the detected country.
If no ip was recognized, geoip fails or country not in the config, it will always redirect to the default domain (from the config).

## Config

When the server is launched, it looks for `config.toml` next to it. 

Example config:
```toml
[redirects]
default = "https://example.com"
DE = "https://de.example.com"
IT = "https://it.example.com"
```

## Example redirects
If you are using the config above and run this app on a domain example.**org**, here is some example redirects:
- [CountryCode] requested URL -> redirected URL
- [US] http://example.org -> https://example.com
- [UK] http://example.org -> https://example.com
- [JP] http://example.org -> https://example.com
- [AU] http://example.org -> https://example.com
- [undetected country] http://example.org -> https://example.com
- [US] http://example.org/index.html -> https://example.com/index.html
- [US] http://example.org/any_path/any_request?any=1&query=2 -> https://example.com/any_path/any_request?any=1&query=2
- [DE] http://example.org/any_path/any_request?any=1&query=2 -> https://de.example.com/any_path/any_request?any=1&query=2
- [IT] http://example.org/any_path/any_request?any=1&query=2 -> https://it.example.com/any_path/any_request?any=1&query=2

## Building

Build for your current platform
```bash
go build
```

Build for linux amd64 (see supported GOOS and GOARCH values [here](https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63))
```bash
env GOOS=linux GOARCH=amd64 go build -o webhook_linux_amd64
```

Build for all release platforms
```bash
./scripts/build-all.sh
```