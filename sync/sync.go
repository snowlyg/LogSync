package sync

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/antlinker/go-mqtt/client"
	"github.com/jander/golog/logger"
	"github.com/jinzhu/gorm"
	"github.com/jlaffaye/ftp"
	"github.com/snowlyg/LogSync/db"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
	"io/ioutil"
	"net"
	"strings"
	"time"
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

func SyncDevice() {
	// 同步设备和通讯录
	//http://fyxt.t.chindeo.com/platform/report/syncdevice 同步设备 post
	//http://fyxt.t.chindeo.com/platform/report/synctelgroup   同步通讯录组 post
	//http://fyxt.t.chindeo.com/platform/report/synctel  同步通讯录 post
	serverList := utils.GetServices()
	for _, server := range serverList {
		if server.ServiceName == "MySQL" {
			func() {
				conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", server.Account, server.Pwd, server.Ip, server.Port, "dois")
				sqlDb, err := gorm.Open("mysql", conn)
				if err != nil {
					logger.Printf("mysql conn error: %v ", err)
					return
				}
				defer sqlDb.Close()
				sqlDb.DB().SetMaxOpenConns(1)
				sqlDb.SetLogger(utils.DefaultGormLogger)
				sqlDb.LogMode(false)

				createDevices(sqlDb)
				createTelphones(sqlDb)
				createTelphoneGroups(sqlDb)

			}()
		}
	}
}

// 同步设备
func createDevices(sqlDb *gorm.DB) {
	var cfDevices []*models.CfDevice
	rows, err := sqlDb.Raw("select dev_id ,dev_code ,dev_desc ,dev_position ,dev_type ,dev_ip  ,dev_active ,dev_create_time  from cf_device").Rows()
	if err != nil {
		logger.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var cfDevice models.CfDevice
		// ScanRows 扫描一行记录到 user
		sqlDb.ScanRows(rows, &cfDevice)

		cfDevices = append(cfDevices, &cfDevice)
	}

	if len(cfDevices) > 0 {
		if db.SQLite != nil {
			db.SQLite.Exec("DELETE FROM t_cf_devices;")
			for _, cfD := range cfDevices {
				db.SQLite.Create(&cfD)
			}

			cfDeviceJson, _ := json.Marshal(&cfDevices)
			data := fmt.Sprintf("data=%s", cfDeviceJson)
			res := utils.PostServices("platform/report/syncdevice", data)
			logger.Error("PostDevice:%s", res)

		} else {
			logger.Println("db.SQLite is null")
		}
	}

}

// 同步通讯录
func createTelphones(sqlDb *gorm.DB) {
	var telphones []*models.Telphone

	rows, err := sqlDb.Raw("select *  from ss_telephone").Rows()
	if err != nil {
		logger.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var telphone models.Telphone
		// ScanRows 扫描一行记录到 user
		sqlDb.ScanRows(rows, &telphone)

		telphones = append(telphones, &telphone)
	}

	if len(telphones) > 0 {
		if db.SQLite != nil {
			db.SQLite.Exec("DELETE FROM t_telphones;")
			for _, cfD := range telphones {
				db.SQLite.Create(&cfD)
			}

			telphoneJson, _ := json.Marshal(&telphones)
			data := fmt.Sprintf("data=%s", telphoneJson)
			res := utils.PostServices("platform/report/synctel", data)
			logger.Error("PostTel:%s", res)
		} else {
			logger.Println("db.SQLite is null")
		}
	}
}

// 同步电话组
func createTelphoneGroups(sqlDb *gorm.DB) {
	var telphoneGroups []*models.TelphoneGroup
	//var telphoneGroups []*models.TelphoneGroup

	rows, err := sqlDb.Raw("select *  from ss_telephone_group").Rows()
	if err != nil {
		logger.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var telphoneGroup models.TelphoneGroup
		// ScanRows 扫描一行记录到 user
		sqlDb.ScanRows(rows, &telphoneGroup)

		telphoneGroups = append(telphoneGroups, &telphoneGroup)
	}

	if len(telphoneGroups) > 0 {
		if db.SQLite != nil {
			db.SQLite.Exec("DELETE FROM t_telphone_groups;")
			for _, cfD := range telphoneGroups {
				db.SQLite.Create(&cfD)
			}

			telphoneGroupJson, _ := json.Marshal(&telphoneGroups)
			data := fmt.Sprintf("data=%s", telphoneGroupJson)
			res := utils.PostServices("platform/report/synctelgroup", data)

			logger.Error("PostTelGroup:%s", res)
		} else {
			logger.Println("db.SQLite is null")
		}
	}
}

// 获取医院编码
func getHospitalCode() string {
	hospitalCode := utils.Conf().Section("config").Key("hospital_code").MustString("")
	if len(hospitalCode) == 0 {
		logger.Error(errors.New("医院编码"))
	}
	return hospitalCode
}

// 循环扫描日志目录，最多层级为4层
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
			logMsg.LogAt = time.Now().Format("2006-01-02 15:04:05")
			if s.Name == time.Now().Format("2006-01-02") {
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

				_ = r.Close()
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
		//utils.MD5(oldMsg.FaultMsg) != utils.MD5(oldMsg.FaultMsg
		if oldMsg.ID == 0 { //如果信息有更新就存储，并推送
			db.SQLite.Save(&logMsg)
			data := fmt.Sprintf("dir_name=%s&hospital_code=%s&device_code=%s&fault_msg=%s&create_at=%s", logMsg.DirName, logMsg.HospitalCode, logMsg.DeviceCode, logMsg.FaultMsg, logMsg.LogAt)
			res := utils.PostServices("platform/report/device", data)
			logger.Error("PostLogMsg:%s", res)

			logger.Printf("%s: 记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode)
		}
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

func SyncDeviceLog() {
	ip := utils.Conf().Section("ftp").Key("ip").MustString("10.0.0.23")
	username := utils.Conf().Section("ftp").Key("username").MustString("admin")
	password := utils.Conf().Section("ftp").Key("password").MustString("Chindeo")

	// 扫描错误日志，设备监控
	c, err := ftp.Dial(fmt.Sprintf("%s:21", ip), ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		logger.Println(err)
	}
	// 登录ftp
	err = c.Login(username, password)
	if err != nil {
		logger.Println(err)
	}

	// 扫描日志目录，记录日志信息
	var logMsg models.LogMsg
	logMsg.HospitalCode = getHospitalCode()
	getDirs(c, "/", logMsg, 0)

	if err := c.Quit(); err != nil {
		logger.Println(err)
	}

	// 监控服务
	// platform_service_id ，service_type_id，create_at，fault_msg
	//http://fyxt.t.chindeo.com/platform/report/service  服务故障上报url
	serverList := utils.GetServices()
	var serverMsgs []*models.ServerMsg

	for _, server := range serverList {

		var serverMsg models.ServerMsg
		serverMsg.ServiceTypeId = server.ServiceTypeId
		serverMsg.ServiceName = server.ServiceName
		serverMsg.ServiceTitle = server.ServiceTitle
		serverMsg.PlatformServiceId = server.Id
		serverMsg.CreatedAt = time.Now()

		switch server.ServiceName {
		case "MySQL":
			func() {
				conn := fmt.Sprintf("%s:%s@%s:%d/mysql?charset=utf8", server.Account, server.Pwd, server.Ip, server.Port)
				sqlDb, err := sql.Open("mysql", conn)
				if err != nil {
					logger.Printf("mysql conn error: %v ", err)
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					return
				} else {
					logger.Println("mysql conn success")
					serverMsg.Status = true
				}
				defer sqlDb.Close()
			}()

		case "EMQX":
			func() {
				addr := fmt.Sprintf("tcp://%s:%d", server.Ip, server.Port)
				mqttClient, err := client.CreateClient(client.MqttOption{
					Addr:               addr,
					ReconnTimeInterval: 1,
					UserName:           server.Account,
					Password:           server.Pwd,
				})

				if err != nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("mqtt client create error: %v ", err)
					return
				}
				//断开连接
				defer mqttClient.Disconnect()

				if mqttClient == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("mqtt conn error: 连接失败 ")
					return
				} else {
					//建立连接
					err = mqttClient.Connect()
					if err != nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("mqtt conn error: %v ", err)
						return
					}

					serverMsg.Status = true
					logger.Println("mqtt conn success")

					return
				}
			}()

		case "RabbitMQ":
			func() {
				mqurl := fmt.Sprintf("amqp://%s:%s@%s:%d/shop", server.Account, server.Pwd, server.Ip, server.Port)
				rabbitmq, err := NewRabbitMQSimple("imoocSimple", mqurl)
				if err != nil {
					if err.Error() == "Exception (403) Reason: \"no access to this vhost\"" {
						serverMsg.Status = true
						logger.Println("RabbitMq conn success")
						return
					}

					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("RabbitMq conn error: %v ", err)
					return
				}

				defer rabbitmq.Destory()

				if rabbitmq == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("RabbitMq conn error: 连接失败 ")
					return
				} else {
					serverMsg.Status = true
					logger.Println("RabbitMq conn success")
					//断开连接
					return
				}

			}()

		case "FileZilla Server":
			func() {
				// 扫描错误日志
				c, err := ftp.Dial(fmt.Sprintf("%s:%d", server.Ip, server.Port), ftp.DialWithTimeout(5*time.Second))
				if err != nil {
					if err.Error() == "Exception (403) Reason: \"no access to this vhost\"" {
						serverMsg.Status = true
						logger.Println("FTP conn success")
						return
					}
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("FTP conn error: %v ", err)
					return
				}

				defer c.Quit()

				if c == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("FTP conn error: 连接失败 ")
					return
				} else {
					err = c.Login(server.Account, server.Pwd)
					if err != nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("FTP conn error: %v ", err)
						return
					} else {
						serverMsg.Status = true
						logger.Println("FTP conn success")
						return
					}
				}
			}()
		default:
			func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", server.Ip, server.Port))
				if err != nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("FTP conn error: %v ", err)
					return
				}
				defer conn.Close()
				if conn == nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = "连接失败"
					logger.Printf("FTP conn error: 连接失败 ")
					return
				}

				serverMsg.Status = true
				logger.Printf("%s conn success", server.ServiceName)
				return

			}()
		}

		// 本机存储数据
		var oldServerMsg models.ServerMsg
		db.SQLite.Where("service_type_id = ?", server.Id).First(&oldServerMsg)
		if oldServerMsg.ID > 0 {
			oldServerMsg.Status = serverMsg.Status
			oldServerMsg.FaultMsg = serverMsg.FaultMsg
			db.SQLite.Save(&oldServerMsg)
		} else {
			db.SQLite.Save(&serverMsg)
		}

		serverMsgs = append(serverMsgs, &serverMsg)
	}
	serverMsgJson, _ := json.Marshal(&serverMsgs)
	data := fmt.Sprintf("fault_data=%s", string(serverMsgJson))
	res := utils.PostServices("platform/report/service", data)

	logger.Printf("serverMsgs: %s", res)

}
