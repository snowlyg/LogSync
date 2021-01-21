package sync

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/snowlyg/LogSync/utils/logging"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

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

var logMsgs []*models.LogMsg // 日志
var logCodes []string        // 日志
var loggerD *logging.Logger

type FaultMsg struct {
	Name    string
	Content string
	Time    string
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

// 扫描设备日志
func SyncDeviceLog() {
	loggerD = logging.GetMyLogger("device")
	logMsgs = nil
	logCodes = nil
	loggerD.Infof("<========================>")
	loggerD.Infof("日志监控开始")

	devices, err := models.GetCfDevice()
	if err != nil {
		loggerD.Infof("GetCfDevice", err)
		return
	}
	if len(devices) == 0 {
		loggerD.Infof("devices len is 0")
		return
	}

	remoteDevices, _ := utils.GetDevices()
	for _, device := range devices {
		remoteDevice := getDeviceByCode(remoteDevices, device.DevCode)
		if remoteDevice == nil {
			loggerD.Errorf("当前设备 >>> %v ", device.DevCode, "跳过日志检查")
			continue
		}
		deviceDir, err := utils.GetDeviceDir(device.DevType)
		if err != nil {
			loggerD.Errorf("GetDeviceDir", err)
		}
		loggerD.Infof(fmt.Sprintf("当前设备 >>> %v ：%v：%v", device.DevType, deviceDir, device.DevCode))

		var logMsg models.LogMsg
		logMsg.DeviceCode = device.DevCode
		logMsg.DirName = deviceDir
		logMsg.Status = true
		logMsg.DevStatus = device.DevStatus
		logMsg.LogAt = time.Now().In(getLocation()).Format("2006-01-02 15:04:05")
		if remoteDevice != nil {
			logMsg.UpdateAt = remoteDevice.LogAt.In(getLocation())
		}
		// 设备类型不在日志扫描范围内
		// PDA 走廊屏 墨水瓶等设备
		if deviceDir == "other" || deviceDir == "" {
			addLogs(&logMsg)
			continue
		}
		// 扫描日志
		getDirs(device.DevIp, logMsg, device)
	}

	var loop = 0
	devicesize := utils.Config.Devicesize
	for loop < len(logMsgs)/devicesize+1 {
		var logMsgSubs []*models.LogMsg
		var index = 0
		for index < devicesize && index+loop*devicesize < len(logMsgs) {
			msg := logMsgs[index+loop*devicesize]
			logMsgSubs = append(logMsgSubs, msg)
			index++
		}

		if len(logMsgSubs) > 0 {
			serverMsgJson, _ := json.Marshal(logMsgSubs)
			data := fmt.Sprintf("log_msgs=%s", string(serverMsgJson))
			var res interface{}
			res, err = utils.SyncServices("platform/report/device", data)
			if err != nil {
				loggerD.Error(err)
			}
			logCodes = nil
			for _, logMsgSub := range logMsgSubs {
				logCodes = append(logCodes, logMsgSub.DeviceCode)
			}
			loggerD.Infof(fmt.Sprintf("提交日志信息返回数据 :%v", res))
			loggerD.Infof(fmt.Sprintf("扫描 %d 个设备 ：%v", len(logMsgSubs), logCodes))
		}
		loop++
	}
	loggerD.Infof("日志监控结束")
}

// 循环扫描日志目录，最多层级为4层
func getDirs(devIp string, logMsg models.LogMsg, device *models.CfDevice) {

	ip := utils.Config.Ftp.Ip
	username := utils.Config.Ftp.Username
	password := utils.Config.Ftp.Password
	c, err := ftp.Dial(fmt.Sprintf("%s:21", ip), ftp.DialWithTimeout(15*time.Second))
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("ftp 连接错误 %v", err))
		addLogs(&logMsg)
		return
	}
	defer c.Quit()
	// 登录ftp
	err = c.Login(username, password)
	if err != nil {
		loggerD.Infof(fmt.Sprintf("ftp 登录错误 %v", err))
		addLogs(&logMsg)
		return
	}

	logPath := fmt.Sprintf("%s/%s/%s/%s", utils.Config.Root, logMsg.DirName, logMsg.DeviceCode, time.Now().Format("2006-01-02"))
	err = cmdDir(c, logPath)
	if err != nil {
		loggerD.Infof("没有创建设备当天日志目录 ", logPath, " 错误： ", err)
		// 进入当天目录,跳过 23点45 当天凌晨 0点15 分钟，给设备创建目录的时间
		if !(time.Now().Hour() == 0 && time.Now().Minute() < 15) || !(time.Now().Hour() == 23 && time.Now().Minute() > 45) {
			pingMsg := utils.GetPingMsg(devIp)
			sendEmptyMsg(&logMsg, fmt.Sprintf("没有创建设备当天日志目录: %s", pingMsg))
			return
		}
	}

	var faultMsgs []*FaultMsg
	ss, err := c.List(getCurrentDir(c))
	if err != nil {
		loggerD.Infof(fmt.Sprintf("获取文件/文件夹列表出错：%v", err))
		addLogs(&logMsg)
		return
	}

	for _, s := range ss {
		// 文件后缀
		extStr := utils.Config.Exts
		names := strings.Split(extStr, ",")
		// 图片后缀
		imgExtStr := utils.Config.Imgexts
		imgExts := strings.Split(imgExtStr, ",")

		if utils.InStrArray(s.Name, names) { // 设备日志文件
			faultMsg := new(FaultMsg)
			faultMsg.Name = s.Name
			faultMsg.Content = string(getFileContent(c, s.Name))
			faultMsg.Time = s.Time.In(getLocation()).Format("2006-01-02 15:04:05")
			faultMsgs = append(faultMsgs, faultMsg)
		} else if utils.InStrArray(s.Name, imgExts) { // 设备截屏图片
			imgDir := utils.Config.Outdir
			path := fmt.Sprintf("%simg/%s.png", imgDir, logMsg.DeviceCode)
			newPath := fmt.Sprintf("%simg/%s_%s.png", imgDir, logMsg.DeviceCode, "new")
			imgContent := getFileContent(c, s.Name)
			isResizeImg := utils.Config.Isresizeimg

			if isResizeImg && s.Size/1024 > 500 {
				if len(imgContent) > 0 {
					err = utils.Create(path, imgContent)
					if err != nil {
						loggerD.Infof(fmt.Sprintf("%s 图片生成失败：%v", s.Name, err))
					}
					err = utils.ResizePng(path, newPath)
					if err != nil {
						loggerD.Infof(fmt.Sprintf("%s 图片重置失败：%v", s.Name, err))
					}
					var file []byte
					if file, err = utils.OpenFile(newPath); err == nil {
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

	if faultMsgs != nil {
		faultMsgsJson, err := json.Marshal(faultMsgs)
		if err != nil {
			loggerD.Errorf(fmt.Sprintf("json 化数据错误 ：%v", err))
		}

		logMsg.FaultMsg = string(faultMsgsJson)
		if logMsg.UpdateAt.IsZero() { //如果信息有更新就存储，并推送
			addLogs(&logMsg)
			loggerD.Infof(fmt.Sprintf("%s: 初次记录设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
			return
		} else {
			subT := time.Now().Sub(logMsg.UpdateAt)
			loggerD.Infof(fmt.Sprintf("日志超时时间：%v", subT))
			if subT.Minutes() >= 15 && time.Now().Hour() != 0 {
				checkLogOverFive(logMsg, device) // 日志超时
			} else {
				addLogs(&logMsg)
			}

			return
		}

	} else {
		if !logMsg.UpdateAt.IsZero() { //如果信息有更新就存储，并推送
			subT := time.Now().Sub(logMsg.UpdateAt)
			loggerD.Infof(fmt.Sprintf("日志超时时间_1：%v", subT))
			if subT.Minutes() >= 15 && time.Now().Hour() != 0 {
				checkLogOverFive(logMsg, device) // 日志超时
				return
			}
		}
	}

}

// 获取时区
func getLocation() *time.Location {
	location, err := time.LoadLocation("Local")
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("时区设置错误 %v", err))
	}
	if location == nil {
		loggerD.Errorf(fmt.Sprintf("时区设置为空"))
	}
	return location
}

// 日志为空，或者目录不存在
func sendEmptyMsg(logMsg *models.LogMsg, msg string) {
	logMsg.StatusMsg = msg
	logMsg.Status = false
	addLogs(logMsg)
}

// 当前路径
func getCurrentDir(c *ftp.ServerConn) string {
	dir, err := c.CurrentDir()
	if err != nil {
		loggerD.Infof(fmt.Sprintf("获取当前文件夹出错：%v", err))
		return ""
	}

	loggerD.Infof(fmt.Sprintf("当前路径 >>> %v", dir))

	return dir
}

// 日志超时未上传
func checkLogOverFive(logMsg models.LogMsg, device *models.CfDevice) {
	loggerD.Infof(fmt.Sprintf(">>> 日志记录超时,开始排查错误"))
	if logMsg.DirName == "nis" { // 大屏
		loggerD.Infof(fmt.Sprintf(">>> 开始排查大屏"))
		webAccount := utils.Config.Web.Account
		webPassword := utils.Config.Web.Password
		if len(strings.TrimSpace(device.DevIp)) == 0 {
			logMsg.Status = false
			if len(logMsg.FaultMsg) == 0 {
				logMsg.StatusMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;设备ip：%s 错误", device.DevIp)
			}
			//saveOrUpdate(&logMsg, oldMsg)
			addLogs(&logMsg)
			loggerD.Infof(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
			return
		}
		// pscp -scp -r -pw Chindeo root@10.0.0.202:/www/ D:/
		inDir := utils.Config.Web.Indir
		idir := fmt.Sprintf("%s/%s/%s/%s/", inDir, logMsg.DirName, logMsg.DeviceCode, time.Now().Format("2006-01-02"))
		pscpDevice(logMsg, webPassword, webAccount, idir, device.DevIp)
		loggerD.Infof(fmt.Sprintf(">>> 大屏排查结束"))
		// 安卓设备
	} else if utils.InStrArray(logMsg.DirName, []string{"nis", "nws", "webapp"}) {
		loggerD.Infof(fmt.Sprintf(">>> 开始排查安卓设备"))
		androidAccount := utils.Config.Android.Account
		androidPassword := utils.Config.Android.Password
		if len(strings.TrimSpace(device.DevIp)) == 0 {
			logMsg.Status = false
			if len(logMsg.FaultMsg) == 0 {
				logMsg.StatusMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;设备ip：%s 错误", device.DevIp)
			}
			//saveOrUpdate(&logMsg, oldMsg)
			addLogs(&logMsg)
			loggerD.Infof(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
			return
		}
		if len(strings.TrimSpace(device.DevIp)) > 0 {
			loggerD.Infof(fmt.Sprintf("dev_id : %s /dev_code : %s", device.DevIp, logMsg.DeviceCode))

			// pscp -scp -r -pw Chindeo root@10.0.0.202:/www/ D:/
			inDir := utils.Config.Android.Indir
			idir := fmt.Sprintf("%s/%s/", inDir, time.Now().Format("2006-01-02"))

			pscpDevice(logMsg, androidPassword, androidAccount, idir, device.DevIp)

		}
		loggerD.Infof(fmt.Sprintf(">>> 安卓设备排查结束"))
	}
	loggerD.Infof(fmt.Sprintf("日志记录超时,排查错误完成"))
}

// 使用 pscp 获取设备上的日志
func pscpDevice(logMsg models.LogMsg, password, account, idir, ip string) {
	odir := createOutDir(logMsg)

	err := os.RemoveAll(odir)
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("%s: RemoveAll %s ", odir, err))
	}

	args := []string{"-scp", "-r", "-pw", password, "-P", "22", fmt.Sprintf("%s@%s:%s", account, ip, idir), odir}
	cmd := exec.Command("pscp", args...)
	loggerD.Infof(fmt.Sprintf("cmd： %v", cmd))
	cmd.Stdin = strings.NewReader("y")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		loggerD.Infof(fmt.Sprintf("pscp %v  执行出错 %v", args, err))
		if len(logMsg.FaultMsg) == 0 {
			logMsg.FaultMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通; pscp:%s", err)
		}
		logMsg.Status = false
		//saveOrUpdate(&logMsg, oldMsg)
		addLogs(&logMsg)
		loggerD.Infof(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
		return
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		loggerD.Infof(fmt.Sprintf("%v 执行出错 %v", cmd, err))
		if len(logMsg.FaultMsg) == 0 {
			logMsg.StatusMsg = fmt.Sprintf("设备超过15分钟未上报日志到FTP,并且PING不通;%v :%s", cmd, err)
		}
		logMsg.Status = false
		//saveOrUpdate(&logMsg, oldMsg)
		addLogs(&logMsg)
		loggerD.Infof(fmt.Sprintf("%s: 扫描设备 %s  错误信息成功", time.Now().String(), logMsg.DeviceCode))
		return
	} else {
		time.Sleep(2 * time.Second)
		logFiles, err := utils.ListDir(odir, "log")
		if err != nil {
			loggerD.Infof(fmt.Sprintf("%s 获取日志文件 出错： %v ", time.Now().Format("2006-01-02 15:04:05"), err))
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
						loggerD.Errorf("FaultLog json.Unmarshal error：%v", err)
					}

					timestamp, err := time.Parse("2006-01-02 15:04:05", faultLog.Timestamp)
					if err != nil {
						loggerD.Errorf(" time.Parse error：%v", err)
					}

					subT := time.Now().Sub(timestamp)
					if subT.Minutes() >= 10 {
						emptyLogRe(logMsg)
						return
					}

				} else if strings.ContainsAny(fileName, "fault.txt") {
					var faultTxt FaultTxt
					err := json.Unmarshal(file, &faultTxt)
					if err != nil {
						loggerD.Errorf("FaultLog json.Unmarshal error：%v", err)
					}

					timestamp, err := time.Parse("2006-01-02 15:04:05", faultTxt.Timestamp)
					if err != nil {
						loggerD.Errorf(" time.Parse error：%v", err)
					}

					subT := time.Now().Sub(timestamp)
					if subT.Minutes() >= 10 {
						emptyLogRe(logMsg)
						return
					}
				}
			}
		}

		if faultMags != nil {
			faultMsgsJson, err := json.Marshal(faultMags)
			if err != nil {
				loggerD.Errorf(fmt.Sprintf("JSON 化数据出错 %v", err))
			}

			logMsg.FaultMsg = string(faultMsgsJson)
		}

		if logMsg.FaultMsg != "" {
			logMsg.Status = true
			//saveOrUpdate(&logMsg, oldMsg)
			addLogs(&logMsg)
			loggerD.Infof(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
		} else {
			emptyLogRe(logMsg)
		}
	}
}

// 没有生成日志的逻辑
func emptyLogRe(logMsg models.LogMsg) {
	logMsg.StatusMsg = "设备超过15分钟未上报日志到FTP,并且设备上也没有生成日志"
	logMsg.Status = false
	//saveOrUpdate(&logMsg, oldMsg)
	addLogs(&logMsg)
	loggerD.Infof(fmt.Sprintf("%s: 扫描设备 %s  错误信息完成", time.Now().String(), logMsg.DeviceCode))
}

// 创建目录
func createOutDir(logMsg models.LogMsg) string {
	outDir := utils.Config.Outdir
	odir := fmt.Sprintf("%s/other_logs/%s/%s/%s", outDir, logMsg.DirName, logMsg.DeviceCode, time.Now().Format("2006-01-02"))

	if !utils.Exist(odir) {
		err := utils.CreateDir(odir)
		if err != nil {
			loggerD.Errorf(fmt.Sprintf("%s 文件夹创建错误： %v", odir, err))
		}
	}
	return odir
}

// 获取文件内容
func getFileContent(c *ftp.ServerConn, name string) []byte {
	r, err := c.Retr(name)
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("Retr 文件内容出错 Error: %s  ", err))
	}
	defer r.Close()

	var buf []byte
	buf, err = ioutil.ReadAll(r)
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("获取文件内容出错  Error: %s  ", err))
	}

	return buf
}

// addLogs 添加日志
func addLogs(logMsg *models.LogMsg) {
	logMsgs = append(logMsgs, logMsg)
}

// 进入下级目录
func cmdDir(c *ftp.ServerConn, root string) error {
	err := c.ChangeDir(root)
	if err != nil {
		loggerD.Infof(fmt.Sprintf("进入下级目录出错：%v", err))
		return err
	}
	getCurrentDir(c)
	return nil
}

func getDeviceByCode(remoteDevices []*utils.Device, code string) *utils.Device {
	for _, device := range remoteDevices {
		if device.DevCode == code {
			return device
		}
	}

	return nil
}
