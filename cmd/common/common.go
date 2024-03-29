// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package common

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/elastos/Elastos.ELA/utils/http"
	"github.com/elastos/Elastos.ELA/utils/http/jsonrpc"

	"github.com/urfave/cli"
)

var (
	rpcUrl      = ""
	rpcIp       = "127.0.0.1"
	rpcPort     = "20336"
	rpcUser     = ""
	rpcPassword = ""
)

func SetRpcConfig(c *cli.Context) {
	rpcUrl = c.String("rpcurl")
	serverIp := c.String("rpcip")
	if serverIp != "" {
		rpcIp = serverIp
	}
	port := c.String("rpcport")
	if port != "" {
		rpcPort = port
	}
	user := c.String("rpcuser")
	if user != "" {
		rpcUser = user
	}
	password := c.String("rpcpassword")
	if password != "" {
		rpcPassword = password
	}
}

func localServer() string {
	if rpcUrl != "" {
		return rpcUrl
	}
	return "http://" + rpcIp + ":" + rpcPort
}

func RPCCall(method string, params http.Params) (interface{}, error) {
	req := jsonrpc.Request{
		Method: method,
		Params: params,
	}
	return jsonrpc.Call(localServer(), req, rpcUser, rpcPassword)
}

func ReadFile(filePath string) (string, error) {
	if _, err := os.Stat(filePath); err != nil {
		return "", errors.New("invalid transaction file path")
	}
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0400)
	if err != nil {
		return "", errors.New("open transaction file failed")
	}
	rawData, err := ioutil.ReadAll(file)
	if err != nil {
		return "", errors.New("read transaction file failed")
	}

	content := strings.TrimSpace(string(rawData))
	if content == "" {
		return "", errors.New("transaction file is empty")
	}
	return content, nil
}
