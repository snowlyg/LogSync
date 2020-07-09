package sync

import (
	"encoding/base64"
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

//'device_code.require'       => '设备编码不能为空！',  string
//'fault_msg.require'         => '故障信息不能为空！',  string
//'create_at.require'         => '创建时间不能为空！' 时间格式
//'dir_name.require'          => '目录名称' 时间格式

var LogCount int         // 扫描设备数量
var DeviceCodes []string // 扫描设备名称

var ServiceCount int      // 扫描设备数量
var ServiceNames []string // 扫描设备名称

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

// 循环扫描日志目录，最多层级为4层
func getDirs(c *ftp.ServerConn, path string, logMsg models.LogMsg, index int) {

	var faultMsgs []*FaultMsg
	location, err := time.LoadLocation("Local")
	if err != nil {
		logger.Println(fmt.Sprintf("时区设置错误 %v", err))
	}
	if location == nil {
		logger.Println(fmt.Sprintf("时区设置为空"))
	}

	ss, err := c.List(path)
	if err != nil {
		logger.Println(fmt.Sprintf("获取文件/文件夹列表出错：%v", err))
	}

	getCurrentDir(c)

	for mun, s := range ss {
		// 设备规则
		switch index {
		case 0:
			extStr := utils.Conf().Section("config").Key("root").MustString("log")
			if s.Name == extStr {
				Next(c, s.Name, logMsg, index)
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
				logger.Println(fmt.Sprintf("错误设备类型：%v", s.Name))
			}

			Next(c, s.Name, logMsg, index)
		case 2:
			// 设备编码
			logMsg.DeviceCode = s.Name
			Next(c, s.Name, logMsg, index)
			LogCount++
			DeviceCodes = append(DeviceCodes, s.Name)
		case 3:
			if s.Name == time.Now().Format("2006-01-02") {
				Next(c, s.Name, logMsg, index)
			} else if mun == len(ss)-1 { // 没有当天日志
				logMsg.FaultMsg = "没有当天日志"
				//checkLogOverFive(logMsg, location)
			}
			continue
		case 4:
			// 文件后缀
			extStr := utils.Conf().Section("config").Key("exts").MustString("")
			exts := strings.Split(extStr, ",")
			// 图片后缀
			imgExtStr := utils.Conf().Section("config").Key("img_exts").MustString("")
			imgExts := strings.Split(imgExtStr, ",")

			// 文件修改时间时区调整
			logMsg.LogAt = s.Time.In(location).Format("2006-01-02 15:04:05")
			logMsg.UpdateAt = s.Time.In(location)
			if utils.InStrArray(s.Name, exts) { // 设备日志文件
				faultMsg := new(FaultMsg)
				faultMsg.Name = s.Name
				faultMsg.Content = string(getFileContent(c, s.Name))
				faultMsgs = append(faultMsgs, faultMsg)
			} else if utils.InStrArray(s.Name, imgExts) { // 设备截屏图片
				logMsg.DeviceImg = "data:image/png;base64," + base64.StdEncoding.EncodeToString(getFileContent(c, s.Name))
			}

		default:
			logger.Println(fmt.Sprintf("进入错误层级：%v", s.Name))
			continue
		}
	}

	var oldMsg models.LogMsg
	utils.SQLite.Where("dir_name = ?", logMsg.DirName).
		Where("device_code = ?", logMsg.DeviceCode).
		Order("created_at desc").
		First(&oldMsg)

	if faultMsgs != nil {
		faultMsgsJson, err := json.Marshal(faultMsgs)
		if err != nil {
			logger.Println(fmt.Sprintf("json 化数据错误 ：%v", err))
		}

		logMsg.FaultMsg = string(faultMsgsJson)
		if oldMsg.ID == 0 { //如果信息有更新就存储，并推送
			utils.SQLite.Save(&logMsg)
			sendDevice(logMsg)
			logger.Println(fmt.Sprintf("%s: 初次记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
		} else {
			utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg})
			sendDevice(logMsg)
		}
	} else if len(logMsg.DeviceCode) > 0 { //没有日志异常
		subT := time.Now().Sub(oldMsg.UpdateAt)
		if subT.Minutes() >= 15 {
			checkLogOverFive(logMsg, location) // 日志超时
		}

		//logMsg.FaultMsg = "没有当天日志"
		//go checkLogOverFive(logMsg, location) // 日志超时

		//logMsg.FaultMsg = "设备今日没有上传日志文件"
		//logMsg.Status = "设备今日没有上传日志文件"
		//logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
		//utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
		//sendDevice(logMsg)
		//logger.Printf("设备:%s恢复正常", logMsg.DeviceCode)
	}

	err = c.ChangeDirToParent()
	if err != nil && !strings.Contains(err.Error(), "200 CDUP successful") {
		logger.Println(fmt.Sprintf("返回上级目录出错 ：%v", err))
	}

	index--
}

// 当前路径
func getCurrentDir(c *ftp.ServerConn) {
	dir, err := c.CurrentDir()
	if err != nil {
		logger.Println(fmt.Sprintf("获取当前文件夹出错：%v", err))
	}

	logger.Println(fmt.Sprintf("当前路径 >>> %v", dir))
}

// 日志超时未上传
func checkLogOverFive(logMsg models.LogMsg, location *time.Location) {
	logger.Println(fmt.Sprintf(">>> 日志记录超时,开始排查错误"))
	defer logger.Println(fmt.Sprintf(" "))
	defer logger.Println(fmt.Sprintf("日志记录超时,排查错误完成"))
	if logMsg.DirName == _NIS.String() { // 大屏
		logger.Println(fmt.Sprintf(">>> 开始排查大屏"))
		defer logger.Println(fmt.Sprintf(" "))
		defer logger.Println(fmt.Sprintf(">>> 大屏排查结束"))
		webIp := utils.Conf().Section("web").Key("ip").MustString("")
		webAccount := utils.Conf().Section("web").Key("account").MustString("administrator")
		webPassword := utils.Conf().Section("web").Key("password").MustString("chindeo888")
		for _, ip := range strings.Split(webIp, ",") {
			func(ip string) {
				if len(strings.TrimSpace(ip)) > 0 {

					conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, 80))
					if err != nil {
						logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
						if len(logMsg.FaultMsg) == 0 {
							logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						}
						logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
						sendDevice(logMsg)
						logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
						return
					} else {
						if conn == nil {
							logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
							if len(logMsg.FaultMsg) == 0 {
								logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
							}
							logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
							utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
							sendDevice(logMsg)
							logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
							return
						}
					}
					defer conn.Close()

					// tasklist /s \\10.0.0.149 /u administrator  /p chindeo888 | findstr "App"
					args := []string{"/s", fmt.Sprintf("\\\\%s", ip), "/u", webAccount, "/p", webPassword}
					cmd := exec.Command("tasklist", args...)
					stdout, err := cmd.StdoutPipe()
					if err != nil {
						logger.Println(fmt.Sprintf("Command 执行出错 %v", err))
						logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
						if len(logMsg.FaultMsg) == 0 {
							logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						}
						logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
						sendDevice(logMsg)
						logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
						return
					}
					defer stdout.Close()

					if err := cmd.Start(); err != nil {
						logger.Println(fmt.Sprintf("tasklist 执行出错 %v", err))
						if len(logMsg.FaultMsg) == 0 {
							logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						}
						logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
						utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
						sendDevice(logMsg)
						logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
						return
					}

					if opBytes, err := ioutil.ReadAll(stdout); err != nil {
						logger.Println(fmt.Sprintf("ReadAll 执行出错 %v", err))
						if len(logMsg.FaultMsg) == 0 {
							logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						}
						logMsg.Status = "设备异常"
						logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
						utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
						sendDevice(logMsg)
						logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
						return
					} else {
						logger.Println(fmt.Sprintf("tasklist couts： %v", string(opBytes)))
						logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
						if strings.Count(string(opBytes), "exe") == 0 {
							logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:程序未启动"
						} else if strings.Count(string(opBytes), "App.exe ") != 4 {
							logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:程序未启动"
						}
						logMsg.FaultMsg = string(opBytes)
						if len(logMsg.FaultMsg) == 0 {
							logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
						}
						utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
						sendDevice(logMsg)
						logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
						return
					}
				}
			}(ip)
		}

		// 安卓设备
	} else if logMsg.DirName == _BIS.String() {
		logger.Println(fmt.Sprintf(">>> 开始排查安卓设备"))
		defer logger.Println(fmt.Sprintf(" "))
		defer logger.Println(fmt.Sprintf(">>> 安卓设备排查结束"))
		androidAccount := utils.Conf().Section("android").Key("account").MustString("root")
		androidPassword := utils.Conf().Section("android").Key("password").MustString("Chindeo")
		var device models.CfDevice
		utils.SQLite.Where("dev_code = ?", logMsg.DeviceCode).Find(&device)
		if len(strings.TrimSpace(device.DevIp)) > 0 {
			logger.Println(fmt.Sprintf("dev_id : %s /dev_code : %s", device.DevIp, logMsg.DeviceCode))
			var faultMags []*FaultMsg
			cli := utils.New(device.DevIp, androidAccount, androidPassword, 22)
			if cli != nil {
				shell := fmt.Sprintf("ps -ef")
				output, err := cli.Run(shell)
				if err != nil {
					logger.Println(fmt.Sprintf(" cli.Run 错误 %s ", err))
					logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
					logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
					if len(logMsg.FaultMsg) == 0 {
						logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
					}
					utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
					sendDevice(logMsg)
					logger.Println(fmt.Sprintf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
					return
				}

				shell = fmt.Sprintf("cd /sdcard/chindeo_app/log/%s && ls", time.Now().Format("2006-01-02"))
				output, err = cli.Run(shell)
				if err != nil {
					logger.Println(fmt.Sprintf(" cli.Run 错误 %s ", err))
					logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
					logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
					if len(logMsg.FaultMsg) == 0 {
						logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
					}
					utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
					sendDevice(logMsg)
					logger.Println(fmt.Sprintf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
					return
				}

				logFiles := strings.Split(output, "\n")
				for _, name := range logFiles {
					shell := fmt.Sprintf("cat /sdcard/chindeo_app/log/%s/%s", time.Now().Format("2006-01-02"), name)
					text, err := cli.Run(shell)
					if err != nil {
						logger.Println(fmt.Sprintf("Command 执行出错 %v", err))
						continue
					}

					faultMsg := new(FaultMsg)
					faultMsg.Name = name
					faultMsg.Content = text
					faultMags = append(faultMags, faultMsg)
					if faultMags != nil {
						faultMsgsJson, err := json.Marshal(faultMags)
						if err != nil {
							logger.Println(fmt.Sprintf("JSON 化数据出错 %v", err))
						}

						logMsg.FaultMsg = string(faultMsgsJson)
					}
				}

				if logMsg.FaultMsg != "" {
					logMsg.Status = ""
					logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
					utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
					sendDevice(logMsg)
					logger.Println(fmt.Sprintf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
					return
				}

			} else {
				logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
				logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备连接不上"
				if len(logMsg.FaultMsg) == 0 {
					logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
				}
				utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
				sendDevice(logMsg)
				logger.Println(fmt.Sprintf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
				return
			}
		} else {
			logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
			logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备ip不存在"
			if len(logMsg.FaultMsg) == 0 {
				logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
			}
			utils.SQLite.Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status})
			sendDevice(logMsg)
			logger.Println(fmt.Sprintf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
			return
		}

	}

}

// 获取文件内容
func getFileContent(c *ftp.ServerConn, name string) []byte {
	r, err := c.Retr(name)
	if err != nil {
		logger.Error(err)
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		logger.Println(fmt.Sprintf("获取文件内容出错 %s  错误信息成功", err))
	}

	return buf
}

// sendDevice 发送请求
func sendDevice(logMsg models.LogMsg) {
	data := fmt.Sprintf("dir_name=%s&device_code=%s&fault_msg=%s&create_at=%s&status=%s&device_img=%s", logMsg.DirName, logMsg.DeviceCode, logMsg.FaultMsg, logMsg.LogAt, logMsg.Status, logMsg.DeviceImg)
	res := utils.SyncServices("platform/report/device", data)
	logger.Println(fmt.Sprintf("提交日志信息返回数据 :%v", res))
}

// 进入下级目录
func Next(c *ftp.ServerConn, name string, logMsg models.LogMsg, index int) {
	if !strings.Contains(name, ".") {
		err := c.ChangeDir(name)
		if err != nil {
			logger.Println(fmt.Sprintf("进入下级目录出错：%v", err))
		}

		index++

		getDirs(c, ".", logMsg, index)
	}
}

// 扫描设备日志
func SyncDeviceLog() {
	logger.Println("<========================>")
	logger.Println("日志监控开始")
	defer logger.Println("日志监控结束")
	defer logger.Println(fmt.Sprintf("扫描 %d 个设备 ：%v", LogCount, DeviceCodes))
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
		logger.Println(fmt.Sprintf("ftp 退出错误：%v", err))
	}

}

// 监控服务
func CheckDevice() {
	// 监控服务
	// platform_service_id ，service_type_id，create_at，fault_msg
	// http://fyxt.t.chindeo.com/platform/report/service  服务故障上报url

	logger.Println("服务监控开始")

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
				conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", server.Account, server.Pwd, server.Ip, server.Port, "dois")
				sqlDb, err := gorm.Open("mysql", conn)
				if err != nil {
					logger.Printf("MYSQL 连接错误: %v ", err)
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
				} else {
					logger.Println("MYSQL 连接成功")
					serverMsg.Status = true
				}
				defer sqlDb.Close()
			}()
			setServiceCountAndNames(server)
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
					logger.Printf("MQTT 客户端创建失败: %v ", err)
				} else {

					if mqttClient == nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = "连接失败"
						logger.Printf("MQTT 连接失败")
					} else {
						//建立连接
						err = mqttClient.Connect()
						if err != nil {
							serverMsg.Status = false
							serverMsg.FaultMsg = err.Error()
							logger.Printf("MQTT 连接出错: %v ", err)
						} else {
							serverMsg.Status = true
							logger.Println("MQTT 连接成功")
						}
					}
				}

				//断开连接
				defer mqttClient.Disconnect()

			}()
			setServiceCountAndNames(server)
		case "RabbitMQ":
			func() {
				mqurl := fmt.Sprintf("amqp://%s:%s@%s:%d/shop", server.Account, server.Pwd, server.Ip, server.Port)
				rabbitmq, err := NewRabbitMQSimple("imoocSimple", mqurl)
				if err != nil {
					if err.Error() == "Exception (403) Reason: \"no access to this vhost\"" {
						serverMsg.Status = true
						logger.Println("RabbitMq conn success")
					} else {
						serverMsg.Status = false
						serverMsg.FaultMsg = err.Error()
						logger.Printf("RabbitMq 连接错误: %v ", err)
					}
				} else {
					if rabbitmq == nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = "连接失败"
						logger.Printf("RabbitMq 连接失败: 连接失败 ")
					} else {
						serverMsg.Status = true
						logger.Println("RabbitMq 连接成功")
					}
				}
				defer rabbitmq.Destory()
			}()
			setServiceCountAndNames(server)
		case "FileZilla Server":
			func() {
				c, err := ftp.Dial(fmt.Sprintf("%s:%d", server.Ip, server.Port), ftp.DialWithTimeout(5*time.Second))
				if err != nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("FTP 连接错误: %v ", err)
				} else {
					if c == nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = "连接失败"
						logger.Printf("FTP 连接失败")
					} else {
						err = c.Login(server.Account, server.Pwd)
						if err != nil {
							serverMsg.Status = false
							serverMsg.FaultMsg = err.Error()
							logger.Printf("FTP 连接错误: %v ", err)
						} else {
							serverMsg.Status = true
							logger.Println("FTP 连接成功")
						}
					}
				}
				defer c.Quit()
			}()
			setServiceCountAndNames(server)
		default:
			func() {
				conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", server.Ip, server.Port))
				if err != nil {
					serverMsg.Status = false
					serverMsg.FaultMsg = err.Error()
					logger.Printf("%s连接错误: %v ", server.ServiceName, err)
				} else {
					if conn == nil {
						serverMsg.Status = false
						serverMsg.FaultMsg = "连接失败"
						logger.Printf("%s 连接失败", server.ServiceName)
					} else {
						serverMsg.Status = true
						logger.Printf("%s conn success", server.ServiceName)
					}

				}
				defer conn.Close()

			}()
			setServiceCountAndNames(server)
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
	res := utils.SyncServices("platform/report/service", data)

	logger.Printf("推送返回信息: %v", res)
	logger.Printf("%d 个服务监控推送完成 : %v", ServiceCount, ServiceNames)
	logger.Println("服务监控结束")
}

func setServiceCountAndNames(server *utils.Server) {
	ServiceCount++
	ServiceNames = append(ServiceNames, server.ServiceName)
}
