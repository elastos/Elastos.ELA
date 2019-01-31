VERSION := $(shell git describe --abbrev=4 --dirty --always --tags)
BUILD =go build -ldflags "-X main.Version=$(VERSION) -X 'main.GoVersion=`go version`'" #-race

all:
	$(BUILD) -o distributor config.go main.go
