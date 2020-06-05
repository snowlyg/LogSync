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

type getRe struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *Token `json:"data"`
}

type Token struct {
	XToken string `json:"X-Token"`
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

func Get(path string) []byte {
	ip := Conf().Section("config").Key("ip").MustString("")
	port := Conf().Section("config").Key("port").MustString("")
	response, err := http.Get(fmt.Sprintf("http://%s:%s/%s", ip, port, path))
	if err != nil {
		log.Println(err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
	}

	return body
}

//http://fyxt.t.chindeo.com/platform/application/login
//http://fyxt.t.chindeo.com/platform/report/device
func GetToken() string {
	appid := Conf().Section("config").Key("appid").MustString("")
	appsecret := Conf().Section("config").Key("appsecret").MustString("")

	return Post("platform/application/login", fmt.Sprintf("appid=%s&appsecret=%s&apptype=%s", appid, appsecret, "hospital"))
}

func Post(url string, data string) string {

	var re getRe
	host := Conf().Section("config").Key("host").MustString("")
	port := Conf().Section("config").Key("port").MustString("80")

	// 超时时间：5秒
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s/%s", host, port, url), strings.NewReader(data))
	if err != nil {
		log.Printf("请求出错：%v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	if !strings.Contains(url, "login") {
		token := GetToken()
		log.Printf("token：%v", token)
		req.Header.Set("X-Token", token)
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	_ = json.Unmarshal(result, &re)

	if re.Code == 200 {
		if !strings.Contains(url, "login") {
			return re.Message
		}
		return re.Data.XToken
	} else {
		return re.Message
	}
}
