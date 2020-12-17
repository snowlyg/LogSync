package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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

type getRestful struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    []*Restful `json:"data"`
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

type Restful struct {
	Id  int64  `json:"id"`
	Url string `json:"url"`
}

//http://fyxt.t.chindeo.com/platform/report/getService  获取服务
func GetServices() ([]*Server, error) {
	re := &getServer{}
	result := Request("GET", "platform/report/getService", "", true)
	if len(result) == 0 {
		return nil, errors.New(fmt.Sprintf("GetServices 获取服务请求没有返回数据"))
	}
	err := json.Unmarshal(result, re)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("GetServices 获取服务解析返回内容报错 :%v", err))
	}

	if re.Code == 200 {
		return re.Data, nil
	} else if re.Code == 401 {
		return nil, errors.New("token 验证失败")
	} else {
		return nil, errors.New(fmt.Sprintf("GetServices 获取服务返回错误信息 :%v", re.Message))
	}
}

//http://fyxt.t.chindeo.com/platform/report/getRestful  获取接口列表
func GetRestfuls() ([]*Restful, error) {
	var re getRestful
	result := Request("GET", "platform/report/getRestful", "", true)
	if len(result) == 0 {
		return nil, errors.New(fmt.Sprintf("GetRestfuls 获取接口请求没有返回数据"))
	}
	err := json.Unmarshal(result, &re)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("GetRestfuls 获取接口解析返回内容报错 :%v", err))
	}

	if re.Code == 200 {
		return re.Data, nil
	} else if re.Code == 401 {
		return nil, errors.New("token 验证失败")
	} else {
		return nil, errors.New(fmt.Sprintf("GetRestfuls 获取接口返回错误信息 :%v", re.Message))
	}
}

//http://fyxt.t.chindeo.com/platform/report/device  发送设备日志信息
//http://fyxt.t.chindeo.com/platform/report/service  提交服务监控信息
func SyncServices(path, data string) (interface{}, error) {
	var re Req
	result := Request("POST", path, data, true)
	if len(result) == 0 {
		return nil, errors.New(fmt.Sprintf("SyncServices 同步数据请求没有返回数据"))
	}
	err := json.Unmarshal(result, &re)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("SyncServices dopost: %s json.Unmarshal error：%v ,with result: %v", path, err, string(result)))
	}
	return re, nil
}

//http://fyxt.t.chindeo.com/platform/application/login
//http://fyxt.t.chindeo.com/platform/report/device
func GetToken() error {
	token := GetCacheToken()
	if token != "" {
		return nil
	}

	var re getToken
	appid := Conf().Section("config").Key("appid").MustString("")
	appsecret := Conf().Section("config").Key("appsecret").MustString("")
	result := Request("POST", "platform/application/login", fmt.Sprintf("appid=%s&appsecret=%s&apptype=%s", appid, appsecret, "hospital"), false)
	if len(result) == 0 {
		return errors.New("请求没有返回数据")
	}

	err := json.Unmarshal(result, &re)
	if err != nil {
		return err
	}

	if re.Code == 200 {
		SetCacheToken(re.Data.XToken)
		return nil
	} else {
		return errors.New(re.Message)
	}
}

func Request(method, url, data string, auth bool) []byte {
	timeout := 3
	timeover := 3
	host := Conf().Section("config").Key("host").MustString("")
	T := time.Tick(time.Duration(timeover) * time.Second)
	var result = make(chan []byte, 10)
	t := time.Duration(timeout) * time.Second
	Client := http.Client{Timeout: t}
	go func() {
		req, _ := http.NewRequest(method, fmt.Sprintf("http://%s/%s", host, url), strings.NewReader(data))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
		if auth {
			req.Header.Set("X-Token", GetCacheToken())
		}
		resp, err := Client.Do(req)
		if err != nil {
			fmt.Println(fmt.Sprintf("%s: %+v", url, err))
			return
		}
		defer resp.Body.Close()
		//if resp.StatusCode == 200 || resp.StatusCode == 500 || resp.StatusCode == 502 {
		b, _ := ioutil.ReadAll(resp.Body)
		result <- b
		//} else {
		//	b, _ := ioutil.ReadAll(resp.Body)
		//	logging.Norm.Info(string(b))
		//}
	}()

	for {
		select {
		case x := <-result:
			return x
		case <-T:
			return nil
		}
	}

}
