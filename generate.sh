#!/bin/bash

GOPATH=$(go env GOPATH)
PATH=$PATH:$GOPATH/bin
export PATH

protoc --proto_path=proto --go_out=. \
  --go_opt=Mplatform.proto=github.com/tknie/ecoflow \
  --go_opt=Mpowerstream.proto=github.com/tknie/ecoflow \
  --go_opt=Mecopacket.proto=github.com/tknie/ecoflow \
  --go_opt=paths=source_relative  proto/platform.proto proto/powerstream.proto proto/ecopacket.proto
