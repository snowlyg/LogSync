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
		err = GetToken()
		if err != nil {
			fmt.Println(fmt.Sprintf("get token err %v", err))
		}
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
		err = GetToken()
		if err != nil {
			fmt.Println(fmt.Sprintf("get token err %v", err))
		}
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

	if re.Code == 200 {
		return re.Data, nil
	} else if re.Code == 401 {
		err = GetToken()
		if err != nil {
			fmt.Println(fmt.Sprintf("get token err %v", err))
		}
		return nil, errors.New("token 验证失败")
	} else {
		return nil, errors.New(fmt.Sprintf("SyncServices 获取接口返回错误信息 :%v", re.Message))
	}
}

//http://fyxt.t.chindeo.com/platform/application/login
//http://fyxt.t.chindeo.com/platform/report/device
func GetToken() error {
	if GetCacheToken() != "" {
		return nil
	}
	var re getToken
	appid := Config.Appid
	appsecret := Config.Appsecret
	result := Request("POST", "platform/application/login", fmt.Sprintf("appid=%s&appsecret=%s&apptype=%s", appid, appsecret, "hospital"), false)
	if len(result) == 0 {
		return errors.New("请求没有返回数据")
	}

	err := json.Unmarshal(result, &re)
	if err != nil {
		return err
	}

	if re.Code == 200 {
		fmt.Println(re.Data.XToken)
		SetCacheToken(re.Data.XToken)
		return nil
	} else {
		return errors.New(re.Message)
	}
}

func Request(method, url, data string, auth bool) []byte {
	timeout := Config.Timeout
	timeover := Config.Timeover
	host := Config.Host
	T := time.Tick(time.Duration(timeover) * time.Second)
	var result = make(chan []byte, 10)
	t := time.Duration(timeout) * time.Second
	Client := http.Client{Timeout: t}
	go func() {
		fullUrl := fmt.Sprintf("http://%s/%s", host, url)
		if strings.Contains(url, "http") {
			fullUrl = url
		}
		req, err := http.NewRequest(method, fullUrl, strings.NewReader(data))
		if err != nil {
			fmt.Println(fmt.Sprintf("%s: %+v", url, err))
			result <- nil
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
		if auth {
			req.Header.Set("X-Token", GetCacheToken())
			phpSessionId := GetSessionId()
			if phpSessionId != nil {
				req.AddCookie(phpSessionId)
			}
		}
		var resp *http.Response
		resp, err = Client.Do(req)
		if err != nil {
			fmt.Println(fmt.Sprintf("%s: %+v", url, err))
			result <- nil
			return
		}
		defer resp.Body.Close()

		if !auth {
			SetSessionId(resp.Cookies())
		}

		b, _ := ioutil.ReadAll(resp.Body)
		result <- b

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
