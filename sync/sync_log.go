package sync

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
	"github.com/snowlyg/LogSync/utils/logging"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type FileInfo struct {
	Name    string
	Content string
	Time    string
}

type Plugin struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

type LogMsg struct {
	DevIp          string `json:"device_ip"` // 服务id
	DevType        int64  `json:"device_type_id"`
	DirName        string `json:"dir_name"`    //系统类型，bis/nis/nws/webapp
	DeviceCode     string `json:"device_code"` //设备编码
	FaultMsg       string `json:"fault_msg"`   //故障信息
	StatusMsg      string `json:"wechat_msg"`  //状态信息
	DeviceImg      string `json:"device_img"`  //设备截图
	Status         bool   `json:"status"`
	StatusType     string `json:"status_type"` //故障类型，设备异常，插件异常，日志异常
	InterfaceError int    `json:"interface_error"`
	Call           string `json:"call"`
	Face           string `json:"face"`
	Interf         string `json:"interf"`
	Iptv           string `json:"iptv"`
	IsBackground   string `json:"is_background"`
	IsEmptyBed     string `json:"is_empty_bed"`
	IsMainActivity string `json:"is_main_activity"`
	Mqtt           string `json:"mqtt"`
	Timestamp      string `json:"timestamp"`
}

// fault.log 文件内容 bis:床旁 nws: 护士站主机  webapp:门旁
type FaultLog struct {
	AppType        string `json:"appType"`
	Call           Plugin `json:"call"`
	Face           Plugin `json:"face"`
	Interf         Plugin `json:"interf"`
	Iptv           Plugin `json:"iptv"`
	Mqtt           Plugin `json:"mqtt"`
	IsBackground   bool   `json:"isBackground"`
	IsEmptyBed     bool   `json:"isEmptyBed"`
	IsMainActivity bool   `json:"isMainActivity"`
	Timestamp      string `json:"timestamp"`
}

// interface.log
//requestType为0表示get请求，为1表示post请求，
//postParamType为0表示post请求的参数为Map，
//postParamType为1表示post请求的参数为Json，code为-1表示返回体为空，code为-2表示JSON解析出错
type InterfaceLog struct {
	Msg           string `json:"msg"`
	PostParamJson string `json:"postParamJson"`
	PostParamType int    `json:"postParamType"`
	Remark        string `json:"remark"`
	RequestType   int    `json:"requestType"`
	Timestamp     string `json:"timestamp"`
	Url           string `json:"url"`
}

// fault.txt 文件内容  nis : 护理大屏
type FaultTxt struct {
	Reason    string `json:"reason"`
	Mqtt      bool   `json:"mqtt"`
	Timestamp string `json:"timestamp"`
}

// 扫描设备日志
func SyncDeviceLog() {
	loggerD := logging.GetMyLogger("device")
	var logMsgs []*LogMsg
	var logCodes []string
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

	// 远程设备信息
	remoteDevices, _ := utils.GetDevices()
	for _, device := range devices {
		// 是否在远程监控范围，远程可设置监控范围
		remoteDevice := getDeviceByCode(remoteDevices, device.DevCode)
		if remoteDevice == nil {
			loggerD.Errorf("当前设备 >>> %v ", device.DevCode, "跳过日志检查")
			continue
		}
		// 获取设备类型编码
		deviceDir, err := utils.GetDeviceDir(device.DevType)
		if err != nil {
			loggerD.Errorf("GetDeviceDir", err)
		}

		loggerD.Infof(fmt.Sprintf("当前设备 >>> %v ：%v：%v", device.DevType, deviceDir, device.DevCode))

		logMsg := &LogMsg{
			DevIp:      device.DevIp,
			DevType:    device.DevType,
			DeviceCode: device.DevCode,
			DirName:    deviceDir,
			Status:     true,
		}

		// 设备类型不在日志扫描范围内
		//  PDA 走廊屏 墨水瓶等设备
		if deviceDir == "other" || deviceDir == "" {
			logMsgs = append(logMsgs, logMsg)
			continue
		}
		// 扫描日志
		getDirs(logMsg, loggerD)
		logMsgs = append(logMsgs, logMsg)
	}

	// 分批发送
	var loop = 0
	devicesize := utils.Config.Devicesize
	for loop < len(logMsgs)/devicesize+1 {
		var logMsgSubs []*LogMsg
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
				for _, logMsgSub := range logMsgSubs {
					loggerD.Infof(
						logMsgSub.DeviceCode,
						logMsgSub.StatusMsg,
						logMsgSub.FaultMsg,
						logMsgSub.Status,
						logMsgSub.InterfaceError,
						logMsgSub.StatusType,
						logMsgSub.Call,
						logMsgSub.Face,
						logMsgSub.Iptv,
						logMsgSub.Interf,
						logMsgSub.Mqtt,
						logMsgSub.Timestamp,
					)
				}
				loggerD.Errorf("提交日志信息", "错误", err)
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

// 循环扫描日志目录
func getDirs(logMsg *LogMsg, loggerD *logging.Logger) {
	c, err := ftp.Dial(fmt.Sprintf("%s:21", utils.Config.Ftp.Ip), ftp.DialWithTimeout(15*time.Second))
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("ftp 连接错误 %v", err))
		return
	}
	defer c.Quit()
	// 登录ftp
	err = c.Login(utils.Config.Ftp.Username, utils.Config.Ftp.Password)
	if err != nil {
		loggerD.Infof(fmt.Sprintf("ftp 登录错误 %v", err))
		return
	}

	logPath := fmt.Sprintf("%s/%s/%s/%s", utils.Config.Root, logMsg.DirName, logMsg.DeviceCode, time.Now().Format(utils.DateLayout))
	err = cmdDir(c, logPath)
	if err != nil {
		loggerD.Infof("ftp 进入路径 ", logPath, " 出错 ", err)
		// 进入当天目录,跳过 23点45 当天凌晨 0点15 分钟，给设备创建目录的时间
		if !(time.Now().Hour() == 0 && time.Now().Minute() < 15) || !(time.Now().Hour() == 23 && time.Now().Minute() > 45) {
			if ok, pingMsg := utils.GetPingMsg(logMsg.DevIp); !ok { // ping 不通
				msg := fmt.Sprintf("【%s】ftp日志路径 %s 访问失败; %s", utils.Config.Faultmsg.Device, logPath, pingMsg)
				logMsg.StatusMsg = msg
				logMsg.StatusType = utils.Config.Faultmsg.Device
				logMsg.Status = false
				loggerD.Infof(fmt.Sprintf("设备%s;%s", logMsg.DeviceCode, msg))
				return
			} else {
				msg := fmt.Sprintf("设备 %s 日志路径 %s 不存在;", logMsg.DeviceCode, logPath)
				loggerD.Infof(msg)
				logMsg.StatusMsg = fmt.Sprintf("【%s】%s", utils.Config.Faultmsg.Logsync, msg)
				checkLogOverFive(logMsg, loggerD)
				return
			}
		}
	}

	dir, err := getCurrentDir(c)
	if err != nil {
		loggerD.Infof(fmt.Sprintf("获取文件当前路径出错：%v", err))
		return
	}

	loggerD.Infof(fmt.Sprintf("当前路径 >>> %v", logPath))

	ss, err := c.List(dir)
	if err != nil {
		loggerD.Infof(fmt.Sprintf("获取文件/文件夹列表出错：%v", err))
		return
	}

	if len(ss) == 0 {
		if ok, pingMsg := utils.GetPingMsg(logMsg.DevIp); !ok { // ping 不通
			msg := fmt.Sprintf("【%s】服务器日志没有文件 %s", utils.Config.Faultmsg.Device, pingMsg)
			logMsg.StatusMsg = msg
			logMsg.Status = false
			logMsg.StatusType = utils.Config.Faultmsg.Device
			loggerD.Infof(fmt.Sprintf("设备%s;%s", logMsg.DeviceCode, msg))
			return
		} else {
			msg := fmt.Sprintf("设备 %s 没有日志文件;", logMsg.DeviceCode)
			loggerD.Infof(msg)
			logMsg.StatusMsg = fmt.Sprintf("【%s】%s", utils.Config.Faultmsg.Logsync, msg)
			checkLogOverFive(logMsg, loggerD)
			return
		}
	}

	location, err := utils.GetLocation()
	if err != nil {
		loggerD.Errorf("get location ", err)
	}
	isOverTime := false
	isSyncTime := true
	syncTimeMsg := ""
	overTimeMsg := ""
	for _, s := range ss {
		// 需要扫描的文件
		extStr := utils.Config.Exts
		names := strings.Split(extStr, ",")
		// 需要扫描的图片
		imgExtStr := utils.Config.Imgexts
		imgExts := strings.Split(imgExtStr, ",")
		if utils.InStrArray(s.Name, names) { // 设备日志文件
			if s.Name == "interface.log" {
				interfaceLogs, err := readInterfaceLogLine(c, s.Name)
				if err != nil {
					loggerD.Infof(fmt.Sprintf("获取 interface.log 内容 %s 错误 %+v", s.Name, err))
					continue
				}
				logMsg.InterfaceError = len(interfaceLogs)
			}

			if s.Name == "error.txt" {
				errorTxt, err := readErrorTxtLine(c, s.Name)
				if err != nil {
					loggerD.Infof(fmt.Sprintf("获取 error.txt 内容 %s 错误 %+v", s.Name, err))
					continue
				}
				logMsg.InterfaceError = errorTxt
			}
			if s.Name == "fault.log" || s.Name == "fault.txt" {
				fileData, err := getFileContent(c, s.Name)
				if err != nil {
					loggerD.Infof(fmt.Sprintf("获取日志文件内容 %s 错误 %+v", s.Name, err))
					continue
				}
				logMsg.FaultMsg = string(fileData)
				err = getPluginsInfo(s.Name, fileData, logMsg)
				if err != nil {
					loggerD.Infof(fmt.Sprintf("解析日志文件 %s 错误 %+v", s.Name, err))
					continue
				}
				// 服务器时间是否同步
				if isSyncTime {
					var subT int64
					if isSyncTime, subT, err = checkSyncTime(logMsg.Timestamp, s.Time); err != nil {
						loggerD.Infof(fmt.Sprintf("检查时间同步错误 %+v", err))
						continue
					}
					syncTimeMsg = fmt.Sprintf("日志记录时间 %s ;服务器时间 %s ;偏差 %d 分钟", logMsg.Timestamp, s.Time.In(location).Format(utils.DateTimeLayout), subT)
				}
				//有超时跳过检查
				if !isOverTime && isSyncTime {
					var subT int64
					if isOverTime, subT, err = checkOverTime(logMsg.Timestamp); err != nil {
						loggerD.Infof(fmt.Sprintf("检查时间超时错误 %+v", err))
						continue
					}
					overTimeMsg = fmt.Sprintf("日志记录时间 %s ;当前时间 %s ;日志已经超时 %d 分钟未更新", logMsg.Timestamp, time.Now().In(location).Format(utils.DateTimeLayout), subT)
				}
			}

		} else if utils.InStrArray(s.Name, imgExts) { // 设备截屏图片
			fileData, err := getFileContent(c, s.Name)
			if err != nil {
				continue
			}
			imgDir := utils.Config.Outdir
			path := fmt.Sprintf("%simg/%s.png", imgDir, logMsg.DeviceCode)
			newPath := fmt.Sprintf("%simg/%s_%s.png", imgDir, logMsg.DeviceCode, "new")
			imgContent := fileData
			isResizeImg := utils.Config.Isresizeimg

			if isResizeImg && s.Size/1024 > 500 {
				if len(imgContent) > 0 {
					err = utils.CreateFile(path, imgContent)
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

	// 设备与服务器时间不一致
	// 设备异常
	if !isSyncTime {
		msg := fmt.Sprintf("【%s】服务器时间和日志记录时间不一致; %s", utils.Config.Faultmsg.Device, syncTimeMsg)
		logMsg.StatusMsg = msg
		logMsg.Status = false
		logMsg.StatusType = utils.Config.Faultmsg.Device
		loggerD.Infof(fmt.Sprintf("设备%s;%s", logMsg.DeviceCode, msg))
		return
	}

	// 超时尝试 ping 设备
	// 日志异常
	if isOverTime {
		if ok, pingMsg := utils.GetPingMsg(logMsg.DevIp); !ok { // ping 不通
			msg := fmt.Sprintf("【%s】%s;%s", utils.Config.Faultmsg.Device, overTimeMsg, pingMsg)
			logMsg.StatusMsg = msg
			logMsg.Status = false
			logMsg.StatusType = utils.Config.Faultmsg.Device
			loggerD.Infof(fmt.Sprintf("设备%s;%s", logMsg.DeviceCode, msg))
			return
		} else {
			msg := fmt.Sprintf("设备 %s 日志超时 %s;", logMsg.DeviceCode, overTimeMsg)
			loggerD.Infof(msg)
			logMsg.StatusMsg = fmt.Sprintf("【%s】%s", utils.Config.Faultmsg.Logsync, msg)
			checkLogOverFive(logMsg, loggerD)
			return
		}
	}

	if logMsg.FaultMsg == "" {
		// 日志异常
		if ok, pingMsg := utils.GetPingMsg(logMsg.DevIp); !ok { // ping 不通
			msg := fmt.Sprintf("【%s】没有生成插件日志;%s", utils.Config.Faultmsg.Device, pingMsg)
			logMsg.StatusMsg = msg
			logMsg.Status = false
			logMsg.StatusType = utils.Config.Faultmsg.Device
			loggerD.Infof(fmt.Sprintf("设备%s;%s", logMsg.DeviceCode, msg))
			return
		} else {
			msg := fmt.Sprintf("设备 %s 没有生成插件日志;", logMsg.DeviceCode)
			loggerD.Infof(msg)
			logMsg.StatusMsg = fmt.Sprintf("【%s】%s", utils.Config.Faultmsg.Logsync, msg)
			checkLogOverFive(logMsg, loggerD)
		}
	}

}

// 当前路径
func getCurrentDir(c *ftp.ServerConn) (string, error) {
	dir, err := c.CurrentDir()
	if err != nil {
		return "", err
	}
	return dir, err
}

// 日志超时未上传
func checkLogOverFive(logMsg *LogMsg, loggerD *logging.Logger) {
	loggerD.Infof(fmt.Sprintf(">>> 日志记录超时,开始排查错误"))
	loggerD.Infof(fmt.Sprintf("dev_id : %s /dev_code : %s", logMsg.DevIp, logMsg.DeviceCode))
	logMsg.Status = false
	logMsg.StatusMsg += fmt.Sprintf("设备 %s(%s) 可以访问;", logMsg.DeviceCode, logMsg.DevIp)
	logMsg.StatusType = utils.Config.Faultmsg.Logsync
	if logMsg.DirName == "nis" { // 大屏
		loggerD.Infof(fmt.Sprintf(">>> 开始排查大屏"))
		if tasklistDevice(logMsg, loggerD, utils.Config.Web.Password, utils.Config.Web.Account, logMsg.DevIp) {
			// pscp -scp -r -pw Chindeo root@10.0.0.202:/www/ D:/
			dir := fmt.Sprintf("%s/%s/%s/%s/", utils.Config.Web.Indir, logMsg.DirName, logMsg.DeviceCode, time.Now().Format(utils.DateLayout))
			pscpDevice(logMsg, loggerD, utils.Config.Web.Password, utils.Config.Web.Account, dir, logMsg.DevIp)
		}

		loggerD.Infof(fmt.Sprintf(">>> 大屏排查结束"))
		// 安卓设备
	} else if utils.InStrArray(logMsg.DirName, []string{"bis", "nws", "webapp"}) {
		loggerD.Infof(fmt.Sprintf(">>> 开始排查安卓设备"))
		// pscp -scp -r -pw Chindeo root@10.0.0.202:/www/ D:/
		dir := fmt.Sprintf("%s/%s/", utils.Config.Android.Indir, time.Now().Format(utils.DateLayout))
		pscpDevice(logMsg, loggerD, utils.Config.Android.Password, utils.Config.Android.Account, dir, logMsg.DevIp)
		loggerD.Infof(fmt.Sprintf(">>> 安卓设备排查结束"))
	} else {
		loggerD.Infof(fmt.Sprintf("未设置排查类型 %s", logMsg.DirName))
	}

	loggerD.Infof(fmt.Sprintf("日志记录超时,排查错误完成"))
}

func tasklistDevice(logMsg *LogMsg, loggerD *logging.Logger, password, account, ip string) bool {
	loggerD.Infof(fmt.Sprintf("开始执行远程查看进程操作"))
	defer loggerD.Infof(fmt.Sprintf("结束执行远程查看进程操作"))
	if runtime.GOOS == "windows" {
		// Tasklist /s 218.22.123.26 /u jtdd /p 12345678
		// /FI "USERNAME ne NT AUTHORITY\SYSTEM" /FI "STATUS eq running"
		args := []string{"/C", "tasklist.exe", "/S", ip, "/U", account, "/P", password, "/FI", "IMAGENAME eq App.exe"}
		cmd := exec.Command("cmd.exe", args...)
		loggerD.Infof(fmt.Sprintf("%+v", cmd))
		var out bytes.Buffer
		var outErr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &outErr
		if err := cmd.Run(); err != nil {
			utf8Data := outErr.Bytes()
			if utils.IsGBK(outErr.Bytes()) {
				utf8Data, _ = simplifiedchinese.GBK.NewDecoder().Bytes(outErr.Bytes())
			}
			loggerD.Infof(fmt.Sprintf("tasklist get %+v", string(utf8Data)))
			logMsg.StatusMsg += "执行 Tasklist 失败，请确认应用程序是否已经开启;"
			return false
		}

		loggerD.Infof(fmt.Sprintf("out put %+v", out.String()))
		if strings.Count(out.String(), "App.exe") == 5 {
			logMsg.StatusMsg += "设备 App应用进程在运行中；"
			return true
		} else {
			logMsg.StatusMsg += "设备 App应用进程未运行，请确认应用程序是否已经开启;"
			return false
		}
	}

	return false
}

// 使用 pscp 获取设备上的日志
func pscpDevice(logMsg *LogMsg, loggerD *logging.Logger, password, account, iDir, ip string) {
	loggerD.Infof(fmt.Sprintf("开始执行远程复制日志操作"))
	defer loggerD.Infof(fmt.Sprintf("结束执行远程复制日志操作"))
	oDir, err := createOutDir(logMsg.DirName, logMsg.DeviceCode)
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("createOutDir %s error %s ", oDir, err))
		return
	}
	err = os.RemoveAll(oDir)
	if err != nil {
		loggerD.Errorf(fmt.Sprintf("%s: RemoveAll %s ", oDir, err))
		return
	}

	if runtime.GOOS == "windows" {
		args := []string{"/C", "pscp.exe", "-scp", "-r", "-pw", password, "-P", "22", fmt.Sprintf("%s@%s:%s", account, ip, iDir), oDir}
		cmd := exec.Command("cmd.exe", args...)
		loggerD.Infof(fmt.Sprintf("%+v", cmd))
		var out bytes.Buffer
		var outErr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &outErr
		if err := cmd.Run(); err != nil {
			utf8Data := outErr.Bytes()
			if utils.IsGBK(outErr.Bytes()) {
				utf8Data, _ = simplifiedchinese.GBK.NewDecoder().Bytes(outErr.Bytes())
			}
			loggerD.Infof(fmt.Sprintf("pscp copy files get %+v", string(utf8Data)))
			logMsg.StatusMsg += fmt.Sprintf("执行 pscp 失败 【%s】;", string(utf8Data))
			return
		}
		loggerD.Infof(fmt.Sprintf("out put %+v", out.String()))
	}

	logFiles, err := utils.ListDir(oDir, "log")
	if err != nil {
		loggerD.Infof(fmt.Sprintf("从路径 %s 获取日志文件出错 %v ", oDir, err))
		logMsg.StatusMsg += "执行 pscp 没有获取到日志;"
		return
	}

	// 没有文件
	if len(logFiles) == 0 {
		msg := "设备内没有生成新的日志"
		logMsg.StatusMsg += msg
		loggerD.Infof(msg)
		return
	}

	isOverTime := false
	overTimeMsg := ""
	var faultMags []*FileInfo
	for _, fileName := range logFiles {
		file, err := utils.OpenFile(fileName)
		if err != nil {
			continue
		}
		err = getPluginsInfo(fileName, file, logMsg)
		if err != nil {
			continue
		}
		if !isOverTime {
			var subT int64
			isOverTime, subT, err = checkOverTime(logMsg.Timestamp)
			if err != nil {
				continue
			}
			location, err := utils.GetLocation()
			if err != nil {
				loggerD.Errorf("get location ", err)
			}
			overTimeMsg = fmt.Sprintf("日志记录时间 %s ;当前时间 %s ;日志已经超时 %d 分钟未更新", logMsg.Timestamp, time.Now().In(location).Format(utils.DateTimeLayout), subT)
		}

		faultMsg := new(FileInfo)
		faultMsg.Name = fileName
		faultMsg.Content = string(file)
		faultMags = append(faultMags, faultMsg)
	}

	if isOverTime {
		logMsg.StatusMsg += overTimeMsg
		logMsg.Status = false
		return
	}

	logMsg.StatusMsg += "设备内正常生成了日志"
}

// 创建目录
func createOutDir(dirName, deviceCode string) (string, error) {
	outDir := utils.Config.Outdir
	oDir := fmt.Sprintf("%s/other_logs/%s/%s/%s", outDir, dirName, deviceCode, time.Now().Format(utils.DateLayout))
	if !utils.Exist(oDir) {
		err := utils.CreateDir(oDir)
		if err != nil {
			return "", err
		}
	}
	return oDir, nil
}

// 获取文件内容
func getFileContent(c *ftp.ServerConn, name string) ([]byte, error) {
	r, err := c.Retr(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var buf []byte
	buf, err = ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// 进入目录
func cmdDir(c *ftp.ServerConn, root string) error {
	err := c.ChangeDir(root)
	if err != nil {
		return err
	}
	return nil
}

// 获取远程设备信息
func getDeviceByCode(remoteDevices []*utils.Device, code string) *utils.Device {
	for _, device := range remoteDevices {
		if device.DevCode == code {
			return device
		}
	}
	return nil
}

// 判断日志是否超时
func checkOverTime(timeTxt string) (bool, int64, error) {
	location, err := utils.GetLocation()
	if err != nil {
		return true, 0, err
	}
	timestamp, err := getTimestamp(timeTxt)
	if err != nil {
		return true, 0, err
	}
	subT := time.Now().In(location).Sub(timestamp)
	ceil := int64(math.Ceil(subT.Minutes()))
	if ceil > utils.Config.Log.Overtime {
		return true, ceil, nil
	}

	return false, ceil, nil
}

// 判断日志是否超时
func checkSyncTime(timetxt string, txtTime time.Time) (bool, int64, error) {
	location, err := utils.GetLocation()
	if err != nil {
		return true, 0, err
	}
	timestamp, err := getTimestamp(timetxt)
	if err != nil {
		return true, 0, err
	}
	subT := txtTime.In(location).Sub(timestamp)
	abs := int64(math.Abs(math.Ceil(subT.Minutes())))
	if abs > utils.Config.Log.Synctime {
		return false, abs, nil
	}

	return true, abs, nil
}

//1.isEmptyBed 空城就不报错，
//2.code：状态码 1 200,999 都算正常 ，
//3.1 设备类型是护士站主机 没有插件是face 或 iptv  或 interf  ，
//3.2 设备类型是门旁 没有插件是iptv 或mqtt 或 interf
//4.1 code：3， 插件是call 正常
//4.2 code:0 , 插件是face 正常
//4.3 code:-1 , interf 正常
func getPluginsInfo(fileName string, file []byte, logMsg *LogMsg) error {
	if !strings.Contains(fileName, "fault.log") && !strings.Contains(fileName, "fault.txt") {
		return nil
	}
	if strings.Contains(fileName, "fault.log") {
		faultLog, err := getFaultLog(file)
		if err != nil {
			return err
		}

		logMsg.Call = faultLog.Call.Reason
		logMsg.Face = faultLog.Face.Reason
		logMsg.Interf = faultLog.Interf.Reason
		logMsg.Iptv = faultLog.Iptv.Reason
		logMsg.IsBackground = getBoolToString(faultLog.IsBackground)
		logMsg.IsEmptyBed = getBoolToString(faultLog.IsEmptyBed)
		logMsg.IsMainActivity = getBoolToString(faultLog.IsMainActivity)
		logMsg.Mqtt = faultLog.Mqtt.Reason
		logMsg.Timestamp = faultLog.Timestamp
		if logMsg.DevType == 0 {
			deviceTypeId, err := utils.GetDeviceTypeId(faultLog.AppType)
			if err != nil {
				return err
			}
			logMsg.DevType = deviceTypeId
		}
		if logMsg.DevType >= 5 && logMsg.DevType <= 10 {
			return nil
		}

		pluginError := true
		statusMsg := fmt.Sprintf("【%s】", utils.Config.Faultmsg.Plugin)
		// 门旁 没有 mqtt
		if logMsg.DevType != 3 {
			if codeIsError(faultLog.Mqtt.Code) {
				pluginError = false
				statusMsg += fmt.Sprintf("插件(mqtt): (%s)%s;", faultLog.Mqtt.Code, faultLog.Mqtt.Reason)
			}
		}

		// 护士站主机,门旁没有iptv,interf
		if logMsg.DevType != 4 && logMsg.DevType != 3 {
			if codeIsError(faultLog.Interf.Code) && faultLog.Face.Code != "-1" {
				pluginError = false
				statusMsg += fmt.Sprintf("插件(interf): (%s)%s;", faultLog.Interf.Code, faultLog.Interf.Reason)
			}
			if codeIsError(faultLog.Iptv.Code) {
				pluginError = false
				statusMsg += fmt.Sprintf("插件(iptv): (%s)%s;", faultLog.Iptv.Code, faultLog.Iptv.Reason)
			}
		}

		// 护士站主机没有face
		if logMsg.DevType != 4 {
			if codeIsError(faultLog.Face.Code) && faultLog.Face.Code != "0" {
				pluginError = false
				statusMsg += fmt.Sprintf("插件(face): (%s)%s;", faultLog.Face.Code, faultLog.Face.Reason)
			}
		}

		if codeIsError(faultLog.Call.Code) && faultLog.Call.Code != "3" {
			pluginError = false
			statusMsg += fmt.Sprintf("插件(call): (%s)%s;", faultLog.Call.Code, faultLog.Call.Reason)
		}

		// 插件异常
		if !pluginError {
			logMsg.StatusType = utils.Config.Faultmsg.Plugin
		}

		logMsg.Status = pluginError
		logMsg.StatusMsg += statusMsg

	} else if strings.Contains(fileName, "fault.txt") {
		if logMsg.DevType == 0 {
			logMsg.DevType = 1
		}
		faultTxt, err := getFaultTxt(file)
		if err != nil {
			return err
		}

		if !faultTxt.Mqtt {
			_, pingMsg := utils.GetPingMsg(logMsg.DevIp)
			logMsg.Status = false
			logMsg.StatusMsg = fmt.Sprintf("【%s】插件(mqtt): %s;%s", utils.Config.Faultmsg.Plugin, faultTxt.Reason, pingMsg)
			logMsg.StatusType = utils.Config.Faultmsg.Plugin
		}

		logMsg.Mqtt = faultTxt.Reason
		logMsg.Timestamp = faultTxt.Timestamp
	}
	return nil
}

//设备状态码不是:1,200,999 的时候报错
func codeIsError(code string) bool {
	if code != "1" && code != "200" && code != "999" {
		return true
	}
	return false
}

// bool to string
func getBoolToString(b bool) string {
	if b {
		return "1"
	} else {
		return "2"
	}
}

// 获取 error.txt 文件内容，解析错误数据
// 提取 http 错误
func readErrorTxtLine(c *ftp.ServerConn, name string) (int, error) {
	var count int
	if !strings.Contains(name, "error.txt") {
		return count, nil
	}
	r, err := c.Retr(name)
	if err != nil {
		return count, err
	}
	defer r.Close()

	lineReader := bufio.NewReader(r)
	for {
		// 相同使用场景下可以采用的方法
		// func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error)
		// func (b *Reader) ReadBytes(delim byte) (line []byte, err error)
		// func (b *Reader) ReadString(delim byte) (line string, err error)
		line, err := lineReader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if len(line) == 0 {
			continue
		}
		if strings.Contains(line, "http://") {
			count++
		}
	}
	return count, nil
}

// 获取 interface.log 文件内容，解析错误数据
func readInterfaceLogLine(c *ftp.ServerConn, name string) ([]*InterfaceLog, error) {
	if !strings.Contains(name, "interface.log") {
		return nil, nil
	}
	r, err := c.Retr(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	lineReader := bufio.NewReader(r)
	var interfaceLogs []*InterfaceLog
	for {
		// 相同使用场景下可以采用的方法
		// func (b *Reader) ReadLine() (line []byte, isPrefix bool, err error)
		// func (b *Reader) ReadBytes(delim byte) (line []byte, err error)
		// func (b *Reader) ReadString(delim byte) (line string, err error)
		line, _, err := lineReader.ReadLine()
		if err == io.EOF {
			break
		}
		if len(line) == 0 {
			continue
		}
		interfaceLog, err := createInterfaceLog(line)
		if err != nil {
			continue
		}
		interfaceLogs = append(interfaceLogs, interfaceLog)
	}

	return interfaceLogs, nil
}

// 获取 interface.log 文件内容
func createInterfaceLog(file []byte) (*InterfaceLog, error) {
	var interfaceLog InterfaceLog
	err := json.Unmarshal(file, &interfaceLog)
	if err != nil {
		return nil, err
	}
	return &interfaceLog, nil
}

// 获取 fault.log 文件内容
func getFaultLog(file []byte) (*FaultLog, error) {
	var faultLog FaultLog
	err := json.Unmarshal(file, &faultLog)
	if err != nil {
		return nil, err
	}
	return &faultLog, nil
}

// 获取 fault.txt 文件内容
func getFaultTxt(file []byte) (*FaultTxt, error) {
	var faultTxt FaultTxt
	err := json.Unmarshal(file, &faultTxt)
	if err != nil {
		return nil, err
	}

	return &faultTxt, nil
}

// string to time
func getTimestamp(ts string) (time.Time, error) {
	if ts == "" {
		return time.Time{}, errors.New("时间为空")
	}
	location, err := utils.GetLocation()
	if err != nil {
		return time.Time{}, err
	}
	timestamp, err := time.ParseInLocation(utils.DateTimeLayout, ts, location)
	if err != nil {
		return time.Time{}, err
	}

	return timestamp, nil
}
