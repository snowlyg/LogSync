package sync

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/antlinker/go-mqtt/client"
	"github.com/jander/golog/logger"
	"github.com/jinzhu/gorm"
	"github.com/jlaffaye/ftp"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
	"io/ioutil"
	"net"
	"os/exec"
	"strings"
	"time"
)

// 日志信息同步接口
//http://fyxt.t.chindeo.com/platform/report/device
//

//'device_code.require'          => '设备编码不能为空！',  string
//'fault_msg.require'          => '故障信息不能为空！',  string
//'create_at.require'          => '创建时间不能为空！' 时间格式
//'dir_name.require'          => '目录名称' 时间格式

type FaultMsg struct {
	Name    string
	Content string
}

// bis 床头交互系统
// nis 护理交互系统，大屏
// nws 护理工作站
// webapp 前端产品
type DirName int

const (
	_BIS DirName = iota
	_NIS
	_NWS
	_WEBAPP
)

func (d DirName) String() string {
	switch d {
	case _BIS:
		return "bis"
	case _NIS:
		return "nis"
	case _NWS:
		return "nws"
	case _WEBAPP:
		return "webapp"
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
	query := "select ct_loc.loc_desc as loc_desc,pac_room.room_desc as room_desc, pac_bed.bed_code as bed_code, dev_id ,dev_code ,dev_desc ,dev_position ,dev_type,dev_active ,dev_create_time,mm.ipaddr as dev_ip from cf_device"
	query += " left join mqtt.mqtt_device as mm on mm.username = cf_device.dev_code"
	query += " left join ct_loc on ct_loc.loc_id = cf_device.ct_loc_id"
	query += " left join pac_room on pac_room.room_id = cf_device.pac_room_id"
	query += " left join pac_bed on pac_bed.bed_id = cf_device.pac_bed_id"

	rows, err := sqlDb.Raw(query).Rows()
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
		if utils.SQLite != nil {
			utils.SQLite.Exec("DELETE FROM t_cf_devices;")
			for _, cfD := range cfDevices {
				utils.SQLite.Create(&cfD)
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
		if utils.SQLite != nil {
			utils.SQLite.Exec("DELETE FROM t_telphones;")
			for _, cfD := range telphones {
				utils.SQLite.Create(&cfD)
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
		if utils.SQLite != nil {
			utils.SQLite.Exec("DELETE FROM t_telphone_groups;")
			for _, cfD := range telphoneGroups {
				utils.SQLite.Create(&cfD)
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
			case _WEBAPP.String():
				logMsg.DirName = _WEBAPP.String()
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
				logMsg.LogAt = s.Time.Format("2006-01-02 15:04:05")
				logMsg.UpdateAt = s.Time
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
		utils.SQLite.Where("dir_name = ?", logMsg.DirName).
			Where("device_code = ?", logMsg.DeviceCode).
			Order("created_at desc").
			First(&oldMsg)

		fmt.Println(fmt.Sprintf("dir_name :%s ,device_code :%s oldMsgid :%d", logMsg.DirName, logMsg.DeviceCode, oldMsg.ID))

		if oldMsg.ID == 0 { //如果信息有更新就存储，并推送
			utils.SQLite.Save(&logMsg)
			sendDevice(logMsg)
			logger.Printf("%s: 初次记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode)

		} else {
			subT := time.Now().Sub(logMsg.UpdateAt)
			if subT.Minutes() > 0 && subT.Minutes() < 15 { // ftp 正常
				sendDevice(logMsg)
				// 大屏
			} else if subT.Minutes() >= 15 && logMsg.DirName == _NIS.String() {
				logger.Error("日志记录超时,开始排查错误")
				webIp := utils.Conf().Section("web").Key("ip").MustString("")
				webAccount := utils.Conf().Section("web").Key("account").MustString("administrator")
				webPassword := utils.Conf().Section("web").Key("password").MustString("chindeo888")
				for _, ip := range strings.Split(webIp, ",") {
					func(ip string) {
						if len(ip) > 0 {
							// tasklist /s \\10.0.0.149 /u administrator  /p chindeo888 | findstr "App"
							args := []string{"/s", fmt.Sprintf("\\\\%s", ip), "/u", webAccount, "/p", webPassword}
							cmd := exec.Command("tasklist", args...)
							stdout, err := cmd.StdoutPipe()
							if err != nil {
								logger.Printf("Command 执行出错 %v", err)
							}
							defer stdout.Close()

							if err := cmd.Start(); err != nil {
								logger.Printf("tasklist 执行出错 %v", err)
							}

							if opBytes, err := ioutil.ReadAll(stdout); err != nil {
								logger.Printf("ReadAll 执行出错 %v", err)
							} else {
								logger.Printf("tasklist couts： %v", string(opBytes))
								if strings.Count(string(opBytes), "exe") == 0 {
									logMsg.Status = "设备异常"
								} else if strings.Count(string(opBytes), "App.exe ") != 4 {
									logMsg.Status = "程序异常"
								}
							}
							sendDevice(logMsg)
							logger.Printf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode)
						}
					}(ip)
				}

				// 安卓设备
			} else if subT.Minutes() >= 15 && logMsg.DirName == _BIS.String() {

				androidAccount := utils.Conf().Section("android").Key("account").MustString("root")
				androidPassword := utils.Conf().Section("android").Key("password").MustString("Chindeo")
				var device models.CfDevice
				utils.SQLite.Where("dev_code = ?", logMsg.DeviceCode).Find(&device)
				logger.Printf("dev_code : %s", device.DevIp)
				if len(device.DevIp) > 0 {

					var faultMags []*FaultMsg
					cli := utils.New(device.DevIp, androidAccount, androidPassword, 22)
					if cli != nil {

						shell := fmt.Sprintf("ps -ef")
						output, err := cli.Run(shell)

						shell = fmt.Sprintf("cd /sdcard/chindeo_app/log/%s && ls", time.Now().Format("2006-01-02"))
						output, err = cli.Run(shell)

						logFiles := strings.Split(output, "\n")
						for _, name := range logFiles {
							shell := fmt.Sprintf("cat /sdcard/chindeo_app/log/%s/%s", time.Now().Format("2006-01-02"), name)
							text, err := cli.Run(shell)
							if err != nil {
								logger.Printf("Command 执行出错 %v", err)
							}

							faultMsg := new(FaultMsg)
							faultMsg.Name = name
							faultMsg.Content = text
							faultMags = append(faultMags, faultMsg)
							if faultMags != nil {
								faultMsgsJson, err := json.Marshal(faultMags)
								if err != nil {
									logger.Error(err)
								}

								logMsg.FaultMsg = string(faultMsgsJson)
								utils.SQLite.Save(&logMsg)

								sendDevice(logMsg)
								logger.Printf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode)
							}
						}
						if err != nil {
							logger.Printf("Command 执行出错 %v", err)
						}
					}

				}

			}
		}

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

// sendDevice 发送请求
func sendDevice(logMsg models.LogMsg) {
	data := fmt.Sprintf("dir_name=%s&device_code=%s&fault_msg=%s&create_at=%s&status=%s", logMsg.DirName, logMsg.DeviceCode, logMsg.FaultMsg, logMsg.LogAt, logMsg.Status)
	res := utils.PostServices("platform/report/device", data)
	logger.Error("PostLogMsg:%s", res)
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
	c, err := ftp.Dial(fmt.Sprintf("%s:21", ip), ftp.DialWithTimeout(30*time.Second))
	if err != nil {
		logger.Println(fmt.Sprintf("ftp 连接错误 %v", err))
	}
	// 登录ftp
	err = c.Login(username, password)
	if err != nil {
		logger.Println(fmt.Sprintf("ftp 登录错误 %v", err))
	}

	// 扫描日志目录，记录日志信息
	var logMsg models.LogMsg
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
		utils.SQLite.Where("service_type_id = ?", server.Id).First(&oldServerMsg)
		if oldServerMsg.ID > 0 {
			oldServerMsg.Status = serverMsg.Status
			oldServerMsg.FaultMsg = serverMsg.FaultMsg
			utils.SQLite.Save(&oldServerMsg)
		} else {
			utils.SQLite.Save(&serverMsg)
		}

		serverMsgs = append(serverMsgs, &serverMsg)
	}
	serverMsgJson, _ := json.Marshal(&serverMsgs)
	data := fmt.Sprintf("fault_data=%s", string(serverMsgJson))
	res := utils.PostServices("platform/report/service", data)

	logger.Printf("serverMsgs: %s", res)

}
