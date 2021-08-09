Elastos ELA
===========
|Actions CI|Go Report Card|
|:-:|:-:|
|[![Build Status()](https://github.com/elastos/Elastos.ELA/workflows/Go/badge.svg?branch=release_v0.4.3)](https://github.com/elastos/Elastos.ELA/actions?query=branch:release_v0.4.3) |[![Code Report()](https://goreportcard.com/badge/github.com/elastos/Elastos.ELA)](https://goreportcard.com/report/github.com/elastos/Elastos.ELA)|

## Introduction

ELA is the digital currency solution within the Elastos ecosystem. It is merged mined with Bitcoin which means the existing bitcoin miners are able to merge mine both BTC and ELA at the same time without expending any additional resources or energy while also providing the enormous hashpower that comes with the bitcoin network.

This project is the source code that can build a full node of ELA blockchain(main chain).

## Table of Contents
- [Elastos ELA](#elastos-ela)
    - [Introduction](#introduction)
    - [Table of Contents](#table-of-contents)
- [Prerequisites on Mac](#prerequisites-on-mac)
    - [1. Check OS version](#1-check-os-version)
    - [2. Install Go distribution](#2-install-go-distribution)
    - [3. Check Go version](#3-check-go-version)
- [Prerequisites on Ubuntu](#prerequisites-on-ubuntu)
    - [1. Check Ubuntu version](#1-check-ubuntu-version)
    - [2. Install git](#2-install-git)
    - [3. Install Go distribution](#3-install-go-distribution)
- [Configure the node](#configure-the-node)
- [Build the node](#build-the-node)
    - [1. Check Go version](#1-check-go-version)
    - [2. Clone source code](#2-clone-source-code)
    - [3. Make](#3-make)
    - [4. Run the node on Ubuntu and Mac](#4-run-the-node-on-ubuntu-and-mac)
- [Build and Run using Docker](#build-and-run-using-docker)
    - [1. Build the Docker node](#1-build-the-docker-node)
    - [2. Run the node in the Docker container](#2-run-the-node-in-the-docker-container)
- [Interact with the node](#interact-with-the-node)
    - [1. Access the web UI of the node](#1-access-the-web-ui-of-the-node)
    - [2. REST API of the node](#2-rest-api-of-the-node)
    - [3. JSON-RPC API of the node](#3-json-rpc-api-of-the-node)
- [Contribution](#contribution)
- [Acknowledgments](#acknowledgments)
- [License](#license)

## Prerequisites on Mac

### 1. Check OS version

Make sure the OSX version is 16.7 or later

```bash
$ uname -srm
Darwin 16.7.0 x86_64
```

### 2. Install Go distribution

Use [Homebrew](https://brew.sh/) to install Golang 1.13.

```bash
$ brew install go@1.13
```

### 3. Check Go version
Check the golang version. Make sure they are the following version number or above.

```bash
$ go version
go version go1.13.15 darwin/amd64
```

## Prerequisites on Ubuntu

### 1. Check Ubuntu version

Make sure your ubuntu version is 18.04 or later

```bash
$ cat /etc/issue
Ubuntu 18.04.5 LTS \n \l
```

### 2. Install Git

```bash
$ sudo apt-get install -y git
```

### 3. Install Go distribution

```bash
$ curl -O https://golang.org/dl/go1.13.15.linux-amd64.tar.gz
$ tar -xvf go1.13.15.linux-amd64.tar.gz
$ sudo chown -R root:root ./go
$ sudo mv go /usr/local
$ export GOPATH=$HOME/go
$ export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
$ source ~/.profile
```

## Build and run step by step

### 1. Clone source code
Make sure you are in the folder
```bash
$ git clone https://github.com/elastos/Elastos.ELA.git
```

If clone works successfully, you should see folder structure like Elastos.ELA/Makefile

### 2. Make

Build the node.
```bash
$ cd Elastos.ELA
$ make
```

If you did not see any error message, congratulations, you have made the ELA full node.

### 3. Configure the node

You can just run a `ela` node without a `config.json` file, the `ela` node will use the main net configuration by default, and provide a JSON-RPC service on [http://localhost:20336](http://localhost:20336).

If you want to customize the node configuration, see the [`config.json`](./docs/config.json.md) to understand what each parameter means on the configuration file.

If you would like to connect to testnet, do the following:

```bash
$ cp -v docs/testnet_config.json.sample config.json
```

If you would like a simple config template, do the following:

```bash
$ cp -v docs/mainnet_config.json.sample config.json
```

Make sure to modify the parameters to what your own specification.

### 4. Run the node on Ubuntu and Mac

Run the node.
```bash
$ ./ela
```

## Build and run using Docker

Alternatively, if don't want to build it manually on Mac or Linux, we also provide a `Dockerfile` to help you (You need to have [Docker](https://www.docker.com/get-started) installed).

### 1. Build the Docker node

```bash
$ cd docker
$ docker build -t ela_node_run .
```

### 2. Run the node in the Docker container

```bash
$ docker run -p 20334:20334 -p 20335:20335 -p 20336:20336 -p 20338:20338 ela_node_run
```

> Note: Don't hit Ctrl-C to terminate the output; instead close this terminal and open another.

> Please note the dockerfile uses the default 'config.json' in the repository. If you're familiar with Docker, you can change the dockerfile to make it use your own ELA Node configuration file.

## Interact with the node

### 1. Web UI

If you would like to access the web UI of the node to get different stats about the node, go to the following URL on your browser: [http://localhost:21333/info](http://localhost:21333/info)

### 2. REST API

Once the node is running successfully, you can access ELA Node's REST APIs:

Example 1: Get the number of nodes to which the node is connected
```bash
$ curl http://localhost:21334/api/v1/node/connectioncount
{
    "Desc": "Success",
    "Error": 0,
    "Result": 5
}
```
Example 2: Get the block height of the node
```bash
$ curl http://localhost:21334/api/v1/block/height
{
    "Desc": "Success",
    "Error": 0,
    "Result": 1000
}
```

If you would like to learn more about what other REST APIs are available for the node, please check out the [Restful API](docs/Restful_API.md)

### 3. JSON-RPC API

Once the node is running successfully, you can access ELA Node's JSON-RPC APIs:

Example 1: Get the hash of the most recent block
```bash
$ curl -H 'Content-Type: application/json' -H 'Accept:application/json' \
  --data '{"method":"getbestblockhash"}' http://localhost:21336
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": "c4e72359cbb128bca244a800fb36d71f64b834e20d437c25de6c62edc46196c7"
}
```

Example 2: Get the hash of the specific blockchain height
```bash
$ curl -H 'Content-Type: application/json' -H 'Accept:application/json' \
  --data '{"method":"getblockhash","params":{"height":1}}' http://localhost:21336
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": "71b422e09dcd2f749d2adc0086735c210084cdb6b59bd4cd42e50455d024a662"
}
```

If you would like to learn more about what other JSON-RPC APIs are available for the node, please check out the [JSON-RPC API](docs/jsonrpc_apis.md)

## Contribution

We welcome contributions to the Elastos ELA Project.

## Acknowledgments

A sincere thank you to all teams and projects that we rely on directly or indirectly.

## License

This project is licensed under the terms of the [MIT license](https://github.com/elastos/Elastos.ELA/blob/master/LICENSE).
