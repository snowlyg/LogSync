package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/jander/golog/logger"
	"github.com/jlaffaye/ftp"
	"github.com/kardianos/service"
	"github.com/snowlyg/LogSync/db"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
)

// 日志信息同步接口
//http://fyxt.t.chindeo.com/platform/report/device
//

//'hospital_code.require'          => '医院编码不能为空！',  string
//'device_code.require'          => '设备编码不能为空！',  string
//'fault_msg.require'          => '故障信息不能为空！',  string
//'create_at.require'          => '创建时间不能为空！' 时间格式
//'dir_name.require'          => '目录名称' 时间格式

type FaultMsg struct {
	Name    string
	Content string
}

// bis 床头交互系统
// nis 护理交互系统
// nws 护理工作站
type DirName int

const (
	_BIS DirName = iota
	_NIS
	_NWS
)

func (d DirName) String() string {
	switch d {
	case _BIS:
		return "bis"
	case _NIS:
		return "nis"
	case _NWS:
		return "nws"
	}

	return "错误设备类型"
}

func getDirs(c *ftp.ServerConn, path string, logMsg models.LogMsg, index int) {

	var faultMsgs []*FaultMsg
	ss, err := c.List(path)
	if err != nil {
		logger.Error(err)
	}

	for _, s := range ss {

		_, err := c.CurrentDir()
		if err != nil {
			logger.Error(err)
		}

		// 设备规则
		switch index {
		case 0:
			//logger.Printf("当前路径0：%s ,当前层级：%d", cDir, index)
			extStr := utils.Conf().Section("config").Key("root").MustString("log")
			if s.Name == extStr {
				err = Next(c, s.Name, logMsg, index)
				if err != nil {
					logger.Error(err)
				}
			} else {
				continue
			}

		case 1:

			// 设备类型
			switch s.Name {
			case _BIS.String():
				logMsg.DirName = _BIS.String()
			case _NIS.String():
				logMsg.DirName = _NIS.String()
			case _NWS.String():
				logMsg.DirName = _NWS.String()
			default:
				logger.Error("错误设备类型")
			}

			//logger.Printf("当前路径1：%s ,当前层级：%d , 设备类型：%s ", cDir, index, logMsg.DirName)

			err = Next(c, s.Name, logMsg, index)
			if err != nil {
				logger.Error(err)
			}
		case 2:
			// 设备编码
			logMsg.DeviceCode = s.Name
			err = Next(c, s.Name, logMsg, index)
			if err != nil {
				logger.Error(err)
			}

			//logger.Printf("当前路径2：%s ,当前层级：%d , 设备编码：%s ", cDir, index, logMsg.DeviceCode)
		case 3:
			//logger.Printf("当前路径3：%s ,当前层级：%d", cDir, index)

			// 日期
			timeStr := time.Now().Format("2006-01-02 15:04:05")
			logMsg.LogAt = timeStr
			if s.Name == timeStr {
				err = Next(c, s.Name, logMsg, index)
				if err != nil {
					logger.Error(err)
				}
			} else {
				continue
			}

		case 4:

			extStr := utils.Conf().Section("config").Key("exts").MustString("")
			exts := strings.Split(extStr, ",")

			//logger.Printf("当前路径4：%s ,当前层级：%d,文件后缀：%v", cDir, index, exts)

			if utils.InStrArray(s.Name, exts) {
				r, err := c.Retr(s.Name)
				if err != nil {
					logger.Error(err)
				}
				//defer r.Close()

				buf, err := ioutil.ReadAll(r)

				faultMsg := new(FaultMsg)
				faultMsg.Name = s.Name
				faultMsg.Content = string(buf)
				faultMsgs = append(faultMsgs, faultMsg)

				r.Close()
			}

		default:
			logger.Error("进入错误层级")
		}
	}

	if faultMsgs != nil {
		faultMsgsJson, err := json.Marshal(faultMsgs)
		if err != nil {
			logger.Error(err)
		}

		logMsg.FaultMsg = string(faultMsgsJson)

		var oldMsg models.LogMsg
		db.SQLite.Where("dir_name = ?", logMsg.DirName).
			Where("hospital_code = ?", logMsg.HospitalCode).
			Where("device_code = ?", logMsg.DeviceCode).
			Where("log_at = ?", logMsg.LogAt).
			Order("created_at desc").
			First(&oldMsg)
		if oldMsg.ID == 0 || oldMsg.FaultMsg != logMsg.FaultMsg { //如果信息有更新就存储，并推送
			db.SQLite.Save(&logMsg)
		}
		data := fmt.Sprintf("dir_name=%s&hospital_code=%s&device_code=%s&fault_msg=%s&create_at=%s", logMsg.DirName, logMsg.HospitalCode, logMsg.DeviceCode, logMsg.FaultMsg, logMsg.LogAt)
		res := utils.Post("platform/report/device", data)
		logger.Error("PostLogMsg:%s", res)

		logger.Printf("%s: 记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode)
	}

	err = c.ChangeDirToParent()
	if err != nil && !strings.Contains(err.Error(), "200 CDUP successful") {
		logger.Error(err)
	}

	index--

	_, err = c.CurrentDir()
	if err != nil {
		logger.Error(err)
	}
	//logger.Printf("上级当前路径：%s ,当前层级：%d", cDir, index)
}

// 进入下级目录
func Next(c *ftp.ServerConn, name string, logMsg models.LogMsg, index int) error {
	if !strings.Contains(name, ".") {
		err := c.ChangeDir(name)
		if err != nil {
			return err
		}

		index++

		getDirs(c, ".", logMsg, index)

		return err
	}
	return nil
}

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	err := models.Init()
	if err != nil {
		panic(err)
	}
	defer models.Close()
	ip := utils.Conf().Section("ftp").Key("ip").MustString("10.0.0.23")
	username := utils.Conf().Section("ftp").Key("username").MustString("admin")
	password := utils.Conf().Section("ftp").Key("password").MustString("Chindeo")

	var ch chan int
	ticker := time.NewTicker(time.Minute * 4)
	go func() {
		for range ticker.C {
			c, err := ftp.Dial(fmt.Sprintf("%s:21", ip), ftp.DialWithTimeout(5*time.Second))
			if err != nil {
				log.Println(err)
			}

			err = c.Login(username, password)
			if err != nil {
				log.Println(err)
			}

			var logMsg models.LogMsg
			logMsg.HospitalCode = getHospitalCode()

			getDirs(c, "/", logMsg, 0)

			if err := c.Quit(); err != nil {
				log.Println(err)
			}
		}
		ch <- 1
	}()
	<-ch
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "LogSync",  //服务显示名称
		DisplayName: "LogSync",  //服务名称
		Description: "同步错误日志信息", //服务描述
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Fatal(err)
	}

	if err != nil {
		logger.Fatal(err)
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "install" {
			s.Install()
			logger.Println("服务安装成功")
			return
		}

		if os.Args[1] == "remove" {
			s.Uninstall()
			logger.Println("服务卸载成功")
			return
		}
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
	}

}

func getHospitalCode() string {
	hospitalCode := utils.Conf().Section("config").Key("hospital_code").MustString("")
	if len(hospitalCode) == 0 {
		log.Fatal(errors.New("医院编码"))
	}
	return hospitalCode
}
