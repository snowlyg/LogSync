package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type getToken struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *Token `json:"data"`
}

type getServer struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    []*Server `json:"data"`
}

type Req struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type Token struct {
	XToken string `json:"X-Token"`
}

type Server struct {
	Id            int64  `json:"id"`
	Ip            string `json:"ip"`
	Port          int64  `json:"port"`
	Account       string `json:"account"`
	Pwd           string `json:"pwd"`
	ServiceTypeId int64  `json:"service_type_id"`
	ServiceName   string `json:"service_name"`
	ServiceTitle  string `json:"service_title"`
}

func GetRequestHref(r *http.Request) string {
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	return strings.Join([]string{scheme, r.Host, r.RequestURI}, "")
}

func GetRequestHostname(r *http.Request) (hostname string) {
	if _url, err := url.Parse(GetRequestHref(r)); err == nil {
		hostname = _url.Hostname()
	}
	return
}

//http://fyxt.t.chindeo.com/platform/report/getService  获取服务
func GetServices() []*Server {
	var re getServer
	result := DoGET("platform/report/getService")
	_ = json.Unmarshal(result, &re)

	if re.Code == 200 {
		return re.Data
	} else {
		log.Printf("Get Service error：%v", re.Message)
		return nil
	}
}

//http://fyxt.t.chindeo.com/platform/report/device  发送设备日志信息
func PostDevices(data string) interface{} {
	var re getServer

	result := DoPOST("platform/report/device", data)
	_ = json.Unmarshal(result, &re)

	return re.Message
}

//http://fyxt.t.chindeo.com/platform/report/service  提交服务监控信息
func PostServices(data string) interface{} {
	var re getServer
	result := DoPOST("platform/report/service", data)
	_ = json.Unmarshal(result, &re)

	return re.Message
}

func DoGET(url string) []byte {
	client, req := getClient("GET", url, "")

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	if !strings.Contains(url, "login") {
		token := GetToken()
		log.Printf("token：%v", token)
		req.Header.Set("X-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求出错：%v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
		result, _ := ioutil.ReadAll(resp.Body)
		return result
	}

	return nil
}

//http://fyxt.t.chindeo.com/platform/application/login
//http://fyxt.t.chindeo.com/platform/report/device
func GetToken() string {
	var re getToken

	appid := Conf().Section("config").Key("appid").MustString("")
	appsecret := Conf().Section("config").Key("appsecret").MustString("")

	result := DoPOST("platform/application/login", fmt.Sprintf("appid=%s&appsecret=%s&apptype=%s", appid, appsecret, "hospital"))
	_ = json.Unmarshal(result, &re)

	if re.Code == 200 {
		return re.Data.XToken
	} else {
		return re.Message
	}
}

func DoPOST(url string, data string) []byte {
	client, req := getClient("POST", url, data)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	if !strings.Contains(url, "login") {
		token := GetToken()
		log.Printf("token：%v", token)
		req.Header.Set("X-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("请求出错：%v", err)
	}
	if resp != nil {
		defer resp.Body.Close()
		result, _ := ioutil.ReadAll(resp.Body)
		return result
	}

	return nil
}

func getClient(reType string, url string, data string) (*http.Client, *http.Request) {
	host := Conf().Section("config").Key("host").MustString("")
	port := Conf().Section("config").Key("port").MustString("80")

	// 超时时间：5秒
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(reType, fmt.Sprintf("http://%s:%s/%s", host, port, url), strings.NewReader(data))
	if err != nil {
		log.Printf("请求出错：%v", err)
	}
	return client, req
}
