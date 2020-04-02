#!/bin/bash

#mac:
GOOS=darwin GOARCH=amd64 go build -buildmode=exe -o ./bin/mac/amd64/lcdash main.go
GOOS=darwin GOARCH=386 go build -buildmode=exe -o ./bin/mac/386/lcdash main.go
#linux
GOOS=linux GOARCH=amd64 go build -buildmode=exe -o ./bin/linux/amd64/lcdash main.go
GOOS=linux GOARCH=386 go build -buildmode=exe -o ./bin/linux/386/lcdash main.go

#windows
GOOS=windows GOARCH=amd64 go build -buildmode=exe -o ./bin/windows/amd64/lcdash main.go
GOOS=windows GOARCH=386 go build -buildmode=exe -o ./bin/windows/386/lcdash main.go