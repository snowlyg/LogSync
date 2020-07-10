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

// 循环扫描日志目录，最多层级为4层
func getDirs(c *ftp.ServerConn, logMsg models.LogMsg) {

	var faultMsgs []*FaultMsg
	location := getLocation()

	ss, err := c.List(getCurrentDir(c))
	if err != nil {
		logger.Println(fmt.Sprintf("%s 获取文件/文件夹列表出错：%v", getCurrentDir(c), err))
	}

	for _, s := range ss {
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
			sendDevice(&logMsg)
			logger.Println(fmt.Sprintf("%s: 初次记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
		} else {
			subT := time.Now().Sub(oldMsg.UpdateAt)
			if subT.Minutes() >= 15 {
				checkLogOverFive(logMsg, oldMsg, location) // 日志超时
			} else {
				utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "update_at": logMsg.UpdateAt})
				sendDevice(&logMsg)
			}
		}

	} else {
		sendEmptyMsg(&logMsg, location, "设备今日没有上传日志文件")
	}

}

func getLocation() *time.Location {
	location, err := time.LoadLocation("Local")
	if err != nil {
		logger.Println(fmt.Sprintf("时区设置错误 %v", err))
	}
	if location == nil {
		logger.Println(fmt.Sprintf("时区设置为空"))
	}
	return location
}

// 日志为空，或者目录不存在
func sendEmptyMsg(logMsg *models.LogMsg, location *time.Location, msg string) {
	var oldMsg models.LogMsg
	utils.SQLite.Where("dir_name = ?", logMsg.DirName).
		Where("device_code = ?", logMsg.DeviceCode).
		Order("created_at desc").
		First(&oldMsg)

	logMsg.FaultMsg = msg
	logMsg.Status = msg
	logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
	logMsg.UpdateAt = time.Now().In(location)
	if oldMsg.ID == 0 { //如果信息有更新就存储，并推送
		utils.SQLite.Save(&logMsg)
	} else {
		utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
	}
	sendDevice(logMsg)
}

// 当前路径
func getCurrentDir(c *ftp.ServerConn) string {
	dir, err := c.CurrentDir()
	if err != nil {
		logger.Println(fmt.Sprintf("获取当前文件夹出错：%v", err))
		return ""
	}
	logger.Println(fmt.Sprintf("当前路径 >>> %v", dir))
	return dir
}

// 日志超时未上传
func checkLogOverFive(logMsg, oldMsg models.LogMsg, location *time.Location) {
	logger.Println(fmt.Sprintf(">>> 日志记录超时,开始排查错误"))
	defer logger.Println(fmt.Sprintf(" "))
	defer logger.Println(fmt.Sprintf("日志记录超时,排查错误完成"))
	if logMsg.DirName == utils.NIS.String() { // 大屏
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
						utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
						sendDevice(&logMsg)
						logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
						return
					} else {
						if conn == nil {
							logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
							if len(logMsg.FaultMsg) == 0 {
								logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
							}
							logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备无法连接"
							utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
							sendDevice(&logMsg)
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
						utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
						sendDevice(&logMsg)
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
						utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
						sendDevice(&logMsg)
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
						utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
						sendDevice(&logMsg)
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
						utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
						sendDevice(&logMsg)
						logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
						return
					}
				}
			}(ip)
		}

		// 安卓设备
	} else if logMsg.DirName == utils.BIS.String() {
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
					utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
					sendDevice(&logMsg)
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
					utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
					sendDevice(&logMsg)
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
					utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
					sendDevice(&logMsg)
					logger.Println(fmt.Sprintf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
					return
				}

			} else {
				logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
				logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备连接不上"
				if len(logMsg.FaultMsg) == 0 {
					logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备连接不上"
				}
				utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
				sendDevice(&logMsg)
				logger.Println(fmt.Sprintf("%s: 扫描安卓记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
				return
			}
		} else {
			logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
			logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:设备ip不存在"
			if len(logMsg.FaultMsg) == 0 {
				logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且PING不通:设备ip不存在"
			}
			utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
			sendDevice(&logMsg)
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
func sendDevice(logMsg *models.LogMsg) {
	data := fmt.Sprintf("dir_name=%s&device_code=%s&fault_msg=%s&create_at=%s&status=%s&device_img=%s", logMsg.DirName, logMsg.DeviceCode, logMsg.FaultMsg, logMsg.LogAt, logMsg.Status, logMsg.DeviceImg)
	res := utils.SyncServices("platform/report/device", data)
	logger.Println(fmt.Sprintf("提交日志信息返回数据 :%v", res))
}

// 扫描设备日志
func SyncDeviceLog() {
	logger.Println("<========================>")
	logger.Println("日志监控开始")
	defer logger.Println("日志监控结束")
	defer logger.Println(fmt.Sprintf("扫描 %d 个设备 ：%v", LogCount, DeviceCodes))
	ip := utils.Conf().Section("ftp").Key("ip").MustString("10.0.0.23")

	LogCount = 0
	DeviceCodes = nil

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

	root := utils.Conf().Section("config").Key("root").MustString("log")
	err = cmdDir(c, root)
	if err != nil {
		return
	}

	devices := utils.GetDevices()
	if len(devices) > 0 {
		for _, device := range devices {

			LogCount++
			DeviceCodes = append(DeviceCodes, device.DeviceCode)

			logger.Println(fmt.Sprintf("当前设备 >>> %v：%v", device.DeviceTypeId, device.DeviceCode))
			deviceDir := getDeviceDir(device.DeviceTypeId)
			// 扫描日志目录，记录日志信息
			var logMsg models.LogMsg
			logMsg.DeviceCode = device.DeviceCode
			logMsg.DirName = deviceDir
			if deviceDir == "" {
				continue
			}
			err = cmdDir(c, deviceDir)
			if err != nil {
				continue
			}
			err = cmdDir(c, device.DeviceCode)
			if err != nil {
				cmdDir(c, "../")
				sendEmptyMsg(&logMsg, getLocation(), "设备志目录不存在")
				continue
			}
			pName := time.Now().Format("2006-01-02")
			err = cmdDir(c, pName)
			if err != nil {
				sendEmptyMsg(&logMsg, getLocation(), "没有创建设备当天日志目录")
				cmdDir(c, "../../")
				continue
			}

			getDirs(c, logMsg)

			cmdDir(c, "../../../")
		}
	}

	if err := c.Quit(); err != nil {
		logger.Println(fmt.Sprintf("ftp 退出错误：%v", err))
	}

}

// 进入下级目录
func cmdDir(c *ftp.ServerConn, root string) error {
	err := c.ChangeDir(root)
	if err != nil {
		logger.Println(fmt.Sprintf("进入下级目录出错：%v", err))
		return err
	}
	getCurrentDir(c)
	return nil
}

// 获取日志类型目录
func getDeviceDir(deviceTypeId utils.DirName) string {
	dirStr := utils.Conf().Section("config").Key("dirs").MustString("bis,nis,nws,webapp")
	dirs := strings.Split(dirStr, ",")
	if len(dirs) > 0 {
		for _, dir := range dirs {
			if dir == deviceTypeId.String() {
				return dir
			}
		}
	}
	return ""
}

// 监控服务
func CheckDevice() {
	// 监控服务
	// platform_service_id ，service_type_id，create_at，fault_msg
	// http://fyxt.t.chindeo.com/platform/report/service  服务故障上报url

	logger.Println("服务监控开始")
	defer logger.Println("服务监控结束")
	defer logger.Println(fmt.Sprintf("%d 个服务监控推送完成 : %v", ServiceCount, ServiceNames))

	ServiceCount = 0
	ServiceNames = nil

	serverList := utils.GetServices()
	logger.Println(fmt.Sprintf("服务： %v", serverList))
	if len(serverList) > 0 {
		var serverMsgs []*models.ServerMsg
		for _, server := range serverList {
			setServiceCountAndNames(server)
			logger.Println(fmt.Sprintf("服务名称： %v", server.ServiceName))
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
						defer sqlDb.Close()
						logger.Println("MYSQL 连接成功")
						serverMsg.Status = true
					}

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
								defer mqttClient.Disconnect()
								serverMsg.Status = true
								logger.Println("MQTT 连接成功")
							}
						}
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
							defer rabbitmq.Destory()
							serverMsg.Status = true
							logger.Println("RabbitMq 连接成功")
						}
					}

				}()
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
								defer c.Quit()
								serverMsg.Status = true
								logger.Println("FTP 连接成功")
							}
						}
					}

				}()
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
							defer conn.Close()
							serverMsg.Status = true
							logger.Printf("%s conn success", server.ServiceName)
						}
					}
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
		res := utils.SyncServices("platform/report/service", data)
		logger.Printf("推送返回信息: %v", res)
	}

}

func setServiceCountAndNames(server *utils.Server) {
	ServiceCount++
	ServiceNames = append(ServiceNames, server.ServiceName)
}
