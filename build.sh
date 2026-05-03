#!/bin/sh

if [ ! -d "bin" ]; then
  mkdir bin
fi

go build -o bin/client cmd/client/main.go