#!/bin/bash
mkdir -p build

echo Building linux amd64
env GOOS=linux GOARCH=amd64 go build -o ./build/go-geo-redirect_linux_amd64

echo Building linux arm64
env GOOS=linux GOARCH=arm64 go build -o ./build/go-geo-redirect_linux_arm64

echo Building darwin amd64
env GOOS=darwin GOARCH=amd64 go build -o ./build/go-geo-redirect_darwin_amd64

echo Building darwin arm64
env GOOS=darwin GOARCH=arm64 go build -o ./build/go-geo-redirect_darwin_arm64

echo Building windows amd64
env GOOS=windows GOARCH=amd64 go build -o ./build/go-geo-redirect_windows_amd64.exe

echo Building windows arm64
env GOOS=windows GOARCH=arm64 go build -o ./build/go-geo-redirect_windows_arm64.exe

echo Done ✅