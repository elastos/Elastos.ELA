package common

import (
	. "Elastos.ELA/common/config"
	. "Elastos.ELA/errors"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"time"
)

var pushBlockFlag bool = true

func CheckPushBlock() bool {
	return pushBlockFlag
}
func GetNoticeServerUrl(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	resp["Result"] = Parameters.NoticeServerUrl
	return resp
}
func SetPushBlockFlag(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)
	open, ok := cmd["Open"].(bool)
	if !ok {
		resp["Error"] = InvalidParams
		return resp
	}
	pushBlockFlag = open
	resp["Result"] = pushBlockFlag
	return resp
}
func SetNoticeServerUrl(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(Success)

	addr, ok := cmd["Url"].(string)
	if !ok || len(addr) == 0 {
		resp["Error"] = InvalidParams
		return resp
	}
	var reg *regexp.Regexp
	pattern := `((http|https)://)(([a-zA-Z0-9\._-]+\.[a-zA-Z]{2,6})|([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}))(:[0-9]{1,4})*(/[a-zA-Z0-9\&%_\./-~-]*)?`
	reg = regexp.MustCompile(pattern)
	if !reg.Match([]byte(addr)) {
		resp["Error"] = InvalidParams
		return resp
	}
	Parameters.NoticeServerUrl = addr
	resp["Result"] = Parameters.NoticeServerUrl
	return resp
}

func PostRequest(cmd map[string]interface{}, url string) (map[string]interface{}, error) {

	var repMsg = make(map[string]interface{})

	data, err := json.Marshal(cmd)
	if err != nil {
		return repMsg, err
	}
	reqData := bytes.NewBuffer(data)
	transport := http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, time.Second*10)
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(time.Now().Add(time.Second * 10))
			return conn, nil
		},
		DisableKeepAlives: false,
	}
	client := &http.Client{Transport: &transport}
	request, err := http.NewRequest("POST", url, reqData)
	if err != nil {
		return repMsg, err
	}
	request.Header.Set("Content-type", "application/json")

	response, err := client.Do(request)
	if response != nil {
		defer response.Body.Close()
		if response.StatusCode == 200 {
			body, _ := ioutil.ReadAll(response.Body)
			if err := json.Unmarshal(body, &repMsg); err == nil {
				return repMsg, err
			}
		}
	}

	if err != nil {
		return repMsg, err
	}

	return repMsg, err
}
