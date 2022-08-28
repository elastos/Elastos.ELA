// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package jsonrpc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"sync"

	elaErr "github.com/elastos/Elastos.ELA/servers/errors"
	htp "github.com/elastos/Elastos.ELA/utils/http"
)

const (
	// JSON-RPC protocol version.
	Version = "2.0"
	// JSON-RPC protocol error codes.
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
	//-32000 to -32099	Server error, waiting for defining
)

//if  we want to run server_test.go, set this to be true (add one test action)

// Handler is the registered method to handle a http request.
type Handler func(htp.Params) (interface{}, error)

// Request represent the standard JSON-RPC request data structure.
type Request struct {
	ID      interface{} `json:"id"`
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// Response represent the standard JSON-RPC Response data structure.
type Response struct {
	ID      interface{} `json:"id"`
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
}

// error returns an error response to the http client.
func (r *Response) error(w http.ResponseWriter, httpStatus, code int, message string) {
	r.Error = &htp.Error{
		Code:    code,
		Message: message,
	}
	r.write(w, httpStatus)
}

// write returns a normal response to the http client.
func (r *Response) write(w http.ResponseWriter, httpStatus int) {
	r.Version = Version
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(httpStatus)
	data, _ := json.Marshal(r)
	w.Write(data)
}

// Config is the configuration of the JSON-RPC server.
type Config struct {
	Path      string
	ServePort uint16
	User      string
	Pass      string
	WhiteList []string
	NetListen func(port uint16) (net.Listener, error)
}

// Server is the JSON-RPC server instance class.
type Server struct {
	cfg    Config
	server *http.Server

	mutex     sync.Mutex
	paramsMap map[string][]string
	handlers  map[string]Handler
}

// RegisterAction register a service handler method by it's name and parameters. When a
// JSON-RPC client's request method matches the registered handler name, it will be invoked.
// This method is safe for concurrency access.
func (s *Server) RegisterAction(name string, handler Handler, params ...string) {
	s.mutex.Lock()
	s.paramsMap[name] = params
	s.handlers[name] = handler
	s.mutex.Unlock()
}

func (s *Server) Start() error {
	if s.cfg.ServePort == 0 {
		return fmt.Errorf("jsonrpc ServePort not configured")
	}

	var err error
	var listener net.Listener
	if s.cfg.NetListen != nil {
		listener, err = s.cfg.NetListen(s.cfg.ServePort)
	} else {
		listener, err = net.Listen("tcp", fmt.Sprint(":", s.cfg.ServePort))
	}
	if err != nil {
		fmt.Printf("Start error err %v \n", err)
		return err
	}

	if s.cfg.Path == "" {
		s.server = &http.Server{Handler: s}
	} else {
		http.Handle(s.cfg.Path, s)
		s.server = &http.Server{}
	}
	return s.server.Serve(listener)
}

func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}
	return fmt.Errorf("server not started")
}

func (s *Server) parseParams(method string, array []interface{}) htp.Params {
	s.mutex.Lock()
	fields := s.paramsMap[method]
	s.mutex.Unlock()

	params := make(htp.Params)
	count := min(len(array), len(fields))
	for i := 0; i < count; i++ {
		params[fields[i]] = array[i]
	}
	return params
}

func (s *Server) clientAllowed(r *http.Request) bool {
	//this ipAbbr  may be  ::1 when request is localhost
	ipAbbr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fmt.Printf("RemoteAddr clientAllowed SplitHostPort failure %s \n", r.RemoteAddr)
		return false

	}
	//after ParseIP ::1 chg to 0:0:0:0:0:0:0:1 the true ip
	remoteIP := net.ParseIP(ipAbbr)

	if remoteIP == nil {
		fmt.Printf("clientAllowed ParseIP ipAbbr %s failure  \n", ipAbbr)
		return false
	}

	if remoteIP.IsLoopback() {
		//log.Debugf("remoteIP %s IsLoopback\n", remoteIP)
		return true
	}

	for _, cfgIP := range s.cfg.WhiteList {
		//WhiteIPList have 0.0.0.0  allow all ip in
		if cfgIP == "0.0.0.0" {
			return true
		}
		if cfgIP == remoteIP.String() {
			return true
		}

	}

	return false
}

func (s *Server) checkAuth(r *http.Request) bool {
	User := s.cfg.User
	Pass := s.cfg.Pass

	if (User == Pass) && (len(User) == 0) {
		return true
	}
	authHeader := r.Header["Authorization"]

	if len(authHeader) <= 0 {
		return false
	}

	authSha256 := sha256.Sum256([]byte(authHeader[0]))

	login := User + ":" + Pass
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(login))
	cfgAuthSha256 := sha256.Sum256([]byte(auth))

	resultCmp := subtle.ConstantTimeCompare(authSha256[:], cfgAuthSha256[:])
	if resultCmp == 1 {

		return true
	}

	// Request's auth doesn't match  user
	return false
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isClientAllowed := s.clientAllowed(r)
	if !isClientAllowed {
		http.Error(w, "Client ip is not allowd", http.StatusForbidden)
		return
	}
	//JSON RPC commands should be POSTs
	if r.Method != "POST" {
		http.Error(w, "JSON RPC protocol only allows POST method",
			http.StatusMethodNotAllowed)
		return
	}

	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentType != "application/json" && contentType != "text/plain" {
		RPCError(w, http.StatusUnsupportedMediaType, InternalError, "JSON-RPC need content type to be application/json or text/plain")
		return
	}

	isCheckAuthOk := s.checkAuth(r)

	if !isCheckAuthOk {
		RPCError(w, http.StatusUnauthorized, InternalError, "Client authenticate failed")
		return
	}
	// read the body of the request
	// TODO: add Max-Length check
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		RPCError(w, http.StatusBadRequest, InvalidRequest, "JSON-RPC request reading error:"+err.Error())
		return
	}

	var requestArray []Request
	var request Request
	err = json.Unmarshal(body, &request)
	if err != nil {
		errArray := json.Unmarshal(body, &requestArray)
		if errArray != nil {
			RPCError(w, http.StatusBadRequest, ParseError, "JSON-RPC request parsing error:"+err.Error())
			return
		}
	}
	var data []byte
	if len(requestArray) == 0 {
		response := s.getResponse(request)
		data, _ = json.Marshal(response)
	} else {
		var responseArray []Response
		for _, req := range requestArray {
			response := s.getResponse(req)
			responseArray = append(responseArray, response)
		}

		data, _ = json.Marshal(responseArray)
	}

	w.Header().Set("Content-type", "application/json")
	w.Write(data)
}

// NewServer creates and return a JSON-RPC server instance.
func NewServer(cfg *Config) *Server {
	return &Server{
		cfg:       *cfg,
		paramsMap: make(map[string][]string),
		handlers:  make(map[string]Handler),
	}
}

func min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}

// RPCError constructs an RPC access error
func RPCError(w http.ResponseWriter, httpStatus int, code elaErr.ServerErrCode, message string) {
	data, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  nil,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"id":      nil,
		},
	})
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(httpStatus)
	w.Write(data)
}

func (s *Server) getResponse(request Request) Response {
	var resp Response
	requestMethod := request.Method
	if len(requestMethod) == 0 {
		resp = Response{
			Version: "2.0",
			Result:  nil,
			ID:      request.ID,
			Error: map[string]interface{}{
				"id":      request.ID,
				"code":    InvalidRequest,
				"message": "JSON-RPC need a method",
			},
		}
		return resp
	}
	handler, ok := s.handlers[requestMethod]
	if !ok {
		resp = Response{
			Version: "2.0",
			Result:  nil,
			ID:      request.ID,
			Error: map[string]interface{}{
				"id":      request.ID,
				"code":    MethodNotFound,
				"message": "JSON-RPC method " + requestMethod + " not found",
			},
		}
		return resp
	}

	requestParams := request.Params
	// Json rpc 1.0 support positional parameters while json rpc 2.0 support named parameters.
	// positional parameters: { "requestParams":[1, 2, 3....] }
	// named parameters: { "requestParams":{ "a":1, "b":2, "c":3 } }
	// Here we support both of them.
	var params htp.Params
	switch requestParams := requestParams.(type) {
	case nil:
	case []interface{}:
		params = s.parseParams(requestMethod, requestParams)
	case map[string]interface{}:
		params = htp.Params(requestParams)
	default:
		resp = Response{
			Version: "2.0",
			Result:  nil,
			ID:      request.ID,
			Error: map[string]interface{}{
				"id":      request.ID,
				"code":    InvalidRequest,
				"message": "params format error, must be an array or a map",
			},
		}
		return resp
	}

	result, err := handler(params)
	if err != nil {
		code := InternalError
		message := fmt.Sprintf("internal error: %s", err)
		resp = Response{
			Version: "2.0",
			Result:  nil,
			ID:      request.ID,
			Error: map[string]interface{}{
				"id":      request.ID,
				"code":    code,
				"message": message,
			},
		}
	} else {
		resp = Response{
			Version: "2.0",
			Result:  result,
			ID:      request.ID,
			Error:   nil,
		}
	}
	return resp
}
