package sync

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jander/golog/logger"
	"github.com/jlaffaye/ftp"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
)

// 日志信息同步接口
//http://fyxt.t.chindeo.com/platform/report/device
//

//'device_code.require'       => '设备编码不能为空！',  string
//'fault_msg.require'         => '故障信息不能为空！',  string
//'create_at.require'         => '创建时间不能为空！' 时间格式
//'dir_name.require'          => '目录名称' 时间格式
var NotFirst bool // 为 true 提交错误日志数据

var logMsgs []*models.LogMsg // 日志
var logCodes []string        // 日志

type FaultMsg struct {
	Name    string
	Content string
}

// fault.log 文件
type FaultLog struct {
	AppType        string `json:"appType"`
	Call           string `json:"call"`
	Face           string `json:"face"`
	Interf         string `json:"interf"`
	Iptv           string `json:"iptv"`
	IsBackground   string `json:"isBackground"`
	IsEmptyBed     string `json:"isEmptyBed"`
	IsMainActivity string `json:"isMainActivity"`
	Mqtt           string `json:"mqtt"`
	Timestamp      string `json:"timestamp"`
}

// fault.txt 文件
type FaultTxt struct {
	Reason    string `json:"reason"`
	Mqtt      string `json:"mqtt"`
	Timestamp string `json:"timestamp"`
}

// 循环扫描日志目录，最多层级为4层
func getDirs(c *ftp.ServerConn, logMsg models.LogMsg) {

	var faultMsgs []*FaultMsg
	ss, err := c.List(getCurrentDir(c))
	if err != nil {
		logger.Println(fmt.Sprintf("获取文件/文件夹列表出错：%v", err))
	}

	for _, s := range ss {
		// 文件后缀
		extStr := utils.Conf().Section("config").Key("exts").MustString("")
		exts := strings.Split(extStr, ",")
		// 图片后缀
		imgExtStr := utils.Conf().Section("config").Key("img_exts").MustString("")
		imgExts := strings.Split(imgExtStr, ",")

		if utils.InStrArray(s.Name, exts) { // 设备日志文件
			faultMsg := new(FaultMsg)
			faultMsg.Name = s.Name
			faultMsg.Content = string(getFileContent(c, s.Name))
			faultMsgs = append(faultMsgs, faultMsg)
		} else if utils.InStrArray(s.Name, imgExts) { // 设备截屏图片
			imgDir := utils.Conf().Section("config").Key("img_dir").MustString("D:/Svr/logSync/")
			path := fmt.Sprintf("%simg/%s.png", imgDir, logMsg.DeviceCode)
			newPath := fmt.Sprintf("%simg/%s_%s.png", imgDir, logMsg.DeviceCode, "new")
			imgContent := getFileContent(c, s.Name)
			isResizeImg := utils.Conf().Section("config").Key("is_resize_img").MustBool(false)

			if isResizeImg && s.Size/1024 > 500 {
				if len(imgContent) > 0 {
					err := utils.Create(path, imgContent)
					if err != nil {
						logger.Println(fmt.Sprintf("%s 图片生成失败：%v", s.Name, err))
					}
					err = utils.ResizePng(path, newPath)
					if err != nil {
						logger.Println(fmt.Sprintf("%s 图片重置失败：%v", s.Name, err))
					}
					if file, err := utils.OpenFile(newPath); err == nil {
						logMsg.DeviceImg = "data:image/png;base64," + base64.StdEncoding.EncodeToString(file)
					}
				}
			} else {
				logMsg.DeviceImg = "data:image/png;base64," + base64.StdEncoding.EncodeToString(imgContent)
			}

			// 删除文件
			os.Remove(path)
			os.Remove(newPath)
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
		location := getLocation()
		if oldMsg.ID == 0 { //如果信息有更新就存储，并推送
			logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
			logMsg.UpdateAt = time.Now().In(location)
			utils.SQLite.Save(&logMsg)
			addLogs(&logMsg)
			logger.Println(fmt.Sprintf("%s: 初次记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
		} else {
			subT := time.Now().Sub(oldMsg.UpdateAt)
			if subT.Minutes() >= 15 && time.Now().Hour() != 0 && NotFirst {
				checkLogOverFive(logMsg, oldMsg) // 日志超时
			} else {
				logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
				logMsg.UpdateAt = time.Now().In(location)
				utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "update_at": logMsg.UpdateAt})
				addLogs(&logMsg)
			}
		}

	} else {
		if oldMsg.ID > 0 { //如果信息有更新就存储，并推送
			subT := time.Now().Sub(oldMsg.UpdateAt)
			if subT.Minutes() >= 15 && time.Now().Hour() != 0 && NotFirst {
				checkLogOverFive(logMsg, oldMsg) // 日志超时
			}
		}
	}
}

// 获取时区
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
	saveOrUpdate(logMsg, oldMsg)
	addLogs(logMsg)
}

// 更新或者新建
func saveOrUpdate(logMsg *models.LogMsg, oldMsg models.LogMsg) {
	location := getLocation()
	logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
	logMsg.UpdateAt = time.Now().In(location)
	if oldMsg.ID == 0 { //如果信息有更新就存储，并推送
		utils.SQLite.Save(&logMsg)
	} else {
		utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
	}
}

// 当前路径
func getCurrentDir(c *ftp.ServerConn) string {
	dir, err := c.CurrentDir()
	if err != nil {
		logger.Println(fmt.Sprintf("获取当前文件夹出错：%v", err))
		return ""
	}

	if utils.Conf().Section("config").Key("loglevel").String() == "debug" {
		logger.Println(fmt.Sprintf("当前路径 >>> %v", dir))
	}

	return dir
}

// 日志超时未上传
func checkLogOverFive(logMsg, oldMsg models.LogMsg) {
	logger.Println(fmt.Sprintf(">>> 日志记录超时,开始排查错误"))
	defer logger.Println(fmt.Sprintf(" "))
	defer logger.Println(fmt.Sprintf("日志记录超时,排查错误完成"))
	if logMsg.DirName == utils.NIS.String() { // 大屏
		logger.Println(fmt.Sprintf(">>> 开始排查大屏"))
		defer logger.Println(fmt.Sprintf(" "))
		defer logger.Println(fmt.Sprintf(">>> 大屏排查结束"))
		var device models.CfDevice
		utils.SQLite.Where("dev_code = ?", logMsg.DeviceCode).Find(&device)
		webAccount := utils.Conf().Section("web").Key("account").MustString("administrator")
		webPassword := utils.Conf().Section("web").Key("password").MustString("chindeo888")
		func(ip string) {
			if len(strings.TrimSpace(ip)) > 0 {
				// pscp -scp -r -pw Chindeo root@10.0.0.202:/www/ D:/
				inDir := utils.Conf().Section("web").Key("inDir").MustString("D:/App/data/log")
				idir := fmt.Sprintf("%s/%s/%s/%s/", inDir, logMsg.DirName, logMsg.DeviceCode, time.Now().Format("2006-01-02"))
				pscpDevice(logMsg, oldMsg, webPassword, webAccount, idir, ip)

				// TODO 检查程序是否运行，但是效率太低
				//// tasklist /s \\10.0.0.149 /u administrator  /p chindeo888 | findstr "App"
				//args := []string{"/s", fmt.Sprintf("\\\\%s", ip), "/u", webAccount, "/p", webPassword}
				//cmd := exec.Command("tasklist", args...)
				//stdout, err := cmd.StdoutPipe()
				//if err != nil {
				//	logger.Println(fmt.Sprintf("tasklist %v  执行出错 %v", args, err))
				//	logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
				//	if len(logMsg.FaultMsg) == 0 {
				//		logMsg.FaultMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通; tasklist:%s", err)
				//	}
				//	logMsg.Status = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通; tasklist:%s", err)
				//	utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
				//	addLogs(&logMsg)
				//	logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功 %s", time.Now().String(), logMsg.DeviceCode, logMsg.Status))
				//	return
				//}
				//defer stdout.Close()
				//
				//if err := cmd.Start(); err != nil {
				//	logger.Println(fmt.Sprintf("tasklist start 执行出错 %v", err))
				//	if len(logMsg.FaultMsg) == 0 {
				//		logMsg.FaultMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;tasklist start :%s", err)
				//	}
				//	logMsg.Status = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;tasklist start :%s", err)
				//	logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
				//	utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
				//	addLogs(&logMsg)
				//	logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功 %s", time.Now().String(), logMsg.DeviceCode, logMsg.Status))
				//	return
				//}
				//
				//if opBytes, err := ioutil.ReadAll(stdout); err != nil {
				//	logger.Println(fmt.Sprintf("ReadAll 执行出错 %v", err))
				//	if len(logMsg.FaultMsg) == 0 {
				//		logMsg.FaultMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通; 读取日志内容：%s", err)
				//	}
				//	logMsg.Status = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通; 读取日志内容：%s", err)
				//	logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
				//	utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
				//	addLogs(&logMsg)
				//	logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功 %s", time.Now().String(), logMsg.DeviceCode, logMsg.Status))
				//	return
				//} else {
				//	logger.Println(fmt.Sprintf("tasklist couts： %d content:%v", strings.Count(string(opBytes), "App.exe "), string(opBytes)))
				//	logMsg.LogAt = time.Now().In(location).Format("2006-01-02 15:04:05")
				//	if strings.Count(string(opBytes), "exe") == 0 || strings.Count(string(opBytes), "App.exe ") != 4 {
				//		logMsg.Status = "设备超过15分钟未上报日志到FTP,并且PING不通:程序未启动"
				//	}else{
				//
				//	}
				//
				//	logMsg.FaultMsg = logMsg.Status
				//	utils.SQLite.Model(&oldMsg).Updates(map[string]interface{}{"log_at": logMsg.LogAt, "fault_msg": logMsg.FaultMsg, "device_img": logMsg.DeviceImg, "status": logMsg.Status, "update_at": logMsg.UpdateAt})
				//	addLogs(&logMsg)
				//	logger.Println(fmt.Sprintf("%s: 扫描大屏记录设备 %s  错误信息成功 %s", time.Now().String(), logMsg.DeviceCode, logMsg.Status))
				//	return
				//}
			}
		}(device.DevIp)

		// 安卓设备
	} else if utils.InStrArray(logMsg.DirName, []string{utils.BIS.String(), utils.NWS.String(), utils.WEBAPP.String()}) {
		logger.Println(fmt.Sprintf(">>> 开始排查安卓设备"))
		defer logger.Println(fmt.Sprintf(" "))
		defer logger.Println(fmt.Sprintf(">>> 安卓设备排查结束"))
		androidAccount := utils.Conf().Section("android").Key("account").MustString("root")
		androidPassword := utils.Conf().Section("android").Key("password").MustString("Chindeo")
		var device models.CfDevice
		utils.SQLite.Where("dev_code = ?", logMsg.DeviceCode).Find(&device)
		if len(strings.TrimSpace(device.DevIp)) > 0 {
			logger.Println(fmt.Sprintf("dev_id : %s /dev_code : %s", device.DevIp, logMsg.DeviceCode))

			// pscp -scp -r -pw Chindeo root@10.0.0.202:/www/ D:/
			inDir := utils.Conf().Section("android").Key("inDir").MustString("/sdcard/chindeo_app/log")
			idir := fmt.Sprintf("%s/%s/", inDir, time.Now().Format("2006-01-02"))

			pscpDevice(logMsg, oldMsg, androidPassword, androidAccount, idir, device.DevIp)

		} else {
			logMsg.Status = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;设备ip：%s 错误", device.DevIp)
			if len(logMsg.FaultMsg) == 0 {
				logMsg.FaultMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;设备ip：%s 错误", device.DevIp)
			}
			saveOrUpdate(&logMsg, oldMsg)
			addLogs(&logMsg)
			logger.Println(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
			return
		}
	}
}

// 使用 pscp 获取设备上的日志
func pscpDevice(logMsg, oldMsg models.LogMsg, password, account, idir, ip string) {
	odir := createOutDir(logMsg)

	err := os.RemoveAll(odir)
	if err != nil {
		logger.Println(fmt.Sprintf("%s: RemoveAll %s ", odir, err))
	}

	args := []string{"-scp", "-r", "-pw", password, "-P", "22", fmt.Sprintf("%s@%s:%s", account, ip, idir), odir}
	cmd := exec.Command("pscp", args...)
	logger.Println(fmt.Sprintf("cmd： %v", cmd))
	cmd.Stdin = strings.NewReader("y")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Println(fmt.Sprintf("pscp %v  执行出错 %v", args, err))
		if len(logMsg.FaultMsg) == 0 {
			logMsg.FaultMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通; pscp:%s", err)
		}
		logMsg.Status = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通; pscp:%s", err)
		saveOrUpdate(&logMsg, oldMsg)
		addLogs(&logMsg)
		logger.Println(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		logger.Println(fmt.Sprintf("%v 执行出错 %v", cmd, err))
		if len(logMsg.FaultMsg) == 0 {
			logMsg.FaultMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;%v :%s", cmd, err)
		}
		logMsg.Status = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;%v :%s", cmd, err)
		saveOrUpdate(&logMsg, oldMsg)
		addLogs(&logMsg)
		logger.Println(fmt.Sprintf("%s: 扫描设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
		return
	} else {
		time.Sleep(2 * time.Second)
		logFiles, err := utils.ListDir(odir, "log")
		if err != nil {
			logger.Println(fmt.Sprintf("%s 获取日志文件 出错： %v ", time.Now().Format("2006-01-02 15:04:05"), err))
		}

		var faultMags []*FaultMsg
		for _, fileName := range logFiles {
			if file, err := utils.OpenFile(fileName); err == nil {
				faultMsg := new(FaultMsg)
				faultMsg.Name = fileName
				faultMsg.Content = string(file)
				faultMags = append(faultMags, faultMsg)

				if strings.ContainsAny(fileName, "fault.log") {
					var faultLog FaultLog
					err := json.Unmarshal(file, &faultLog)
					if err != nil {
						log.Printf("FaultLog json.Unmarshal error：%v", err)
					}

					timestamp, err := time.Parse("2006-01-02 15:04:05", faultLog.Timestamp)
					if err != nil {
						log.Printf(" time.Parse error：%v", err)
					}

					subT := time.Now().Sub(timestamp)
					if subT.Minutes() >= 10 {
						emptyLogRe(logMsg, oldMsg)
						return
					}

				} else if strings.ContainsAny(fileName, "fault.txt") {
					var faultTxt FaultTxt
					err := json.Unmarshal(file, &faultTxt)
					if err != nil {
						log.Printf("FaultLog json.Unmarshal error：%v", err)
					}

					timestamp, err := time.Parse("2006-01-02 15:04:05", faultTxt.Timestamp)
					if err != nil {
						log.Printf(" time.Parse error：%v", err)
					}

					subT := time.Now().Sub(timestamp)
					if subT.Minutes() >= 10 {
						emptyLogRe(logMsg, oldMsg)
						return
					}
				}
			}
		}

		if faultMags != nil {
			faultMsgsJson, err := json.Marshal(faultMags)
			if err != nil {
				logger.Println(fmt.Sprintf("JSON 化数据出错 %v", err))
			}

			logMsg.FaultMsg = string(faultMsgsJson)
		}

		if logMsg.FaultMsg != "" {
			logMsg.Status = "设备有正常生成了日志,但是设备超过15分钟未上报日志到FTP"
			saveOrUpdate(&logMsg, oldMsg)
			addLogs(&logMsg)
			logger.Println(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
		} else {
			emptyLogRe(logMsg, oldMsg)
		}
	}
}

// 没有生成日志的逻辑
func emptyLogRe(logMsg models.LogMsg, oldMsg models.LogMsg) {
	logMsg.FaultMsg = "设备超过15分钟未上报日志到FTP,并且设备上也没有生成日志"
	logMsg.Status = "设备超过15分钟未上报日志到FTP,并且设备上也没有生成日志"
	saveOrUpdate(&logMsg, oldMsg)
	addLogs(&logMsg)
	logger.Println(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
}

// 创建目录
func createOutDir(logMsg models.LogMsg) string {
	outDir := utils.Conf().Section("android").Key("outDir").MustString("D:Svr/logSync")
	odir := fmt.Sprintf("%s/other_logs/%s/%s/%s", outDir, logMsg.DirName, logMsg.DeviceCode, time.Now().Format("2006-01-02"))

	if !utils.Exist(odir) {
		err := utils.CreateDir(odir)
		if err != nil {
			logger.Println(fmt.Sprintf("%s 文件夹创建错误： %v", odir, err))
		}
	}
	return odir
}

// 获取文件内容
func getFileContent(c *ftp.ServerConn, name string) []byte {
	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		logger.Println("++++++++++++++++Retr 程序异常退出++++++++++++++++")
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容，55
		}
		logger.Println("++++++++++++++++Retr 程序 recover ++++++++++++++++")
	}()
	r, err := c.Retr(name)
	if err != nil {
		logger.Panicln(fmt.Sprintf("Retr 文件内容出错 Error: %s  ", err))
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		logger.Println(fmt.Sprintf("获取文件内容出错  Error: %s  ", err))
	}

	return buf
}

// addLogs 添加日志
func addLogs(logMsg *models.LogMsg) {
	logMsgs = append(logMsgs, logMsg)
	logCodes = append(logCodes, logMsg.DeviceCode)
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

// 扫描设备日志
func SyncDeviceLog() {
	logMsgs = nil
	logCodes = nil
	logger.Println("<========================>")
	logger.Println("日志监控开始")

	ip := utils.Conf().Section("ftp").Key("ip").MustString("10.0.0.23")

	username := utils.Conf().Section("ftp").Key("username").MustString("admin")
	password := utils.Conf().Section("ftp").Key("password").MustString("Chindeo")
	// 扫描错误日志，设备监控
	defer func() { // 必须要先声明defer，否则不能捕获到panic异常
		logger.Println("++++++++++++++++ftp 程序异常退出++++++++++++++++")
		if err := recover(); err != nil {
			fmt.Println(err) // 这里的err其实就是panic传入的内容，55
		}
		logger.Println("++++++++++++++++ftp 程序 recover ++++++++++++++++")
	}()
	c, err := ftp.Dial(fmt.Sprintf("%s:21", ip), ftp.DialWithTimeout(15*time.Second))
	if err != nil {
		logger.Panicln(fmt.Sprintf("ftp 连接错误 %v", err))
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

	devices, err := models.GetCfDevice()
	if err == nil && len(devices) > 0 {
		for _, device := range devices {

			logger.Println(fmt.Sprintf("当前设备 >>> %v：%v", device.DevType, device.DevCode))
			deviceDir := getDeviceDir(device.DevType)
			// 扫描日志目录，记录日志信息
			var logMsg models.LogMsg
			logMsg.DeviceCode = device.DevCode
			logMsg.DirName = deviceDir
			if deviceDir == "" {
				continue
			}

			// 进入设备类型目录
			err = cmdDir(c, deviceDir)
			if err != nil {
				continue
			}

			// 进入设备编码目录
			err = cmdDir(c, device.DevCode)
			if err != nil {
				cmdDir(c, "../")
				sendEmptyMsg(&logMsg, getLocation(), "设备志目录不存在")
				continue
			}

			pName := time.Now().Format("2006-01-02")
			err = cmdDir(c, pName)
			if err != nil {
				// 进入当天目录,跳过 23点45 当天凌晨 0点15 分钟，给设备创建目录的时间
				if !(time.Now().Hour() == 0 && time.Now().Minute() < 15) || !(time.Now().Hour() == 23 && time.Now().Minute() > 45) {
					sendEmptyMsg(&logMsg, getLocation(), "没有创建设备当天日志目录")
				}
				cmdDir(c, "../../")
				continue
			}

			getDirs(c, logMsg)

			cmdDir(c, "../../../")
		}
	}

	serverMsgJson, _ := json.Marshal(logMsgs)
	logger.Println(fmt.Sprintf("日志文件大小：%d", len(serverMsgJson)/1024/1024))
	data := fmt.Sprintf("log_msgs=%s", string(serverMsgJson))
	var res interface{}
	res, err = utils.SyncServices("platform/report/device", data)
	if err != nil {
		logger.Println(err)
	}
	logger.Println(fmt.Sprintf("提交日志信息返回数据 :%v", res))

	if err := c.Quit(); err != nil {
		logger.Println(fmt.Sprintf("ftp 退出错误：%v", err))
	}

	logger.Println(fmt.Sprintf("扫描 %d 个设备 ：%v", len(logMsgs), logCodes))
	logger.Println("日志监控结束")

}
