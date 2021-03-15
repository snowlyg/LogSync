package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	// _ "net/http/pprof"

	_ "github.com/go-sql-driver/mysql"
	"github.com/patrickmn/go-cache"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
	"github.com/snowlyg/LogSync/utils/logging"
)

// Version 版本
var Version = "<UNDEFINED>"

func syncDevice() {
	t := utils.Config.Data.Timeduration
	v := utils.Config.Data.Timetype
	duration := getDuration(t, v)
	logger := logging.GetMyLogger("sync")
	go func() {
		for {
			sync.SyncDevice(logger)
			fmt.Println(fmt.Sprintf("设备数据同步"))
			time.Sleep(duration)
		}
	}()
}

func syncDeviceLog() {
	go func() {
		t := utils.Config.Device.Timeduration
		v := utils.Config.Device.Timetype
		duration := getDuration(t, v)
		loggerD := logging.GetMyLogger("device")
		var logMsgs []*sync.LogMsg
		var logCodes []string
		ca := cache.New(20*time.Minute, 24*time.Hour)
		for {
			// TODO：运维后台增加运维时间段，跳过微信报警
			// 进入当天目录,跳过 23点45 当天凌晨 0点59 分钟，给设备创建目录的时间
			// 当天 3 点会重启 apache，mysql，emqx，rabbitmq，音视频，数据服务，接口服务，暂停 30 分钟
			// if time.Now().Hour() == 0 {
			// 	time.Sleep(time.Minute * 40)
			// }
			// if time.Now().Hour() == 23 && time.Now().Minute() >= 45 {
			// 	time.Sleep(time.Minute * 15)
			// }
			// if time.Now().Hour() == 3 {
			// 	time.Sleep(time.Minute * 30)
			// }
			sync.SyncDeviceLog(logMsgs, logCodes, loggerD, ca)
			fmt.Println(fmt.Sprintf("设备日志监控同步"))
			time.Sleep(duration)
		}
	}()
}

func syncService() {
	t := utils.Config.Device.Timeduration
	v := utils.Config.Device.Timetype
	duration := getDuration(t, v)
	logger := logging.GetMyLogger("service")
	go func() {
		for {
			sync.CheckService(logger)
			fmt.Println(fmt.Sprintf("服务数据同步"))
			time.Sleep(duration)
		}
	}()
}

func syncRestful() {
	t := utils.Config.Restful.Timeduration
	v := utils.Config.Restful.Timetype
	duration := getDuration(t, v)
	logger := logging.GetMyLogger("restful")
	var restfulMsgs []*sync.RestfulMsg
	var restfulURL []string
	go func() {
		for {
			sync.CheckRestful(restfulMsgs, restfulURL, logger)
			fmt.Println(fmt.Sprintf("接口监控同步"))
			time.Sleep(duration)
		}
	}()
}

func getDuration(t int64, v string) time.Duration {
	switch v {
	case "h":
		return time.Hour * time.Duration(t)
	case "m":
		return time.Minute * time.Duration(t)
	case "s":
		return time.Second * time.Duration(t)
	default:
		return time.Minute * time.Duration(t)
	}
}

// Action 程序操作指令 install remove start stop version restart
var Action = flag.String("action", "", "程序操作指令")

func main() {
	// go func() {
	// 	http.ListenAndServe("localhost:6061", nil)
	// }()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] [command]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  -action <install remove start stop restart version>\n")
		fmt.Fprintf(os.Stderr, "    程序操作指令\n")
		fmt.Fprintf(os.Stderr, "\n")
	}
	flag.Parse()
	utils.InitConfig()

	if *Action == "version" {
		fmt.Println(Version)
		return
	}

	defer println("********** START **********")
	err := utils.GetToken()
	if err != nil {
		fmt.Println(fmt.Sprintf("get token err %v", err))
	}
	go func() {
		syncDeviceLog()
	}()
	go func() {
		syncRestful()
	}()
	go func() {
		syncService()
	}()
	go func() {
		syncDevice()
	}()

	// 定时推送内存，cpu 使用率
	go func() {
		t := utils.Config.System.Timeduration
		v := utils.Config.System.Timetype
		duration := getDuration(t, v)
		for {
			var memData [][]interface{}
			var cpuData [][]interface{}
			var diskData [][]interface{}
			m, _ := mem.VirtualMemory()
			c, _ := cpu.Percent(0, false)
			v, _ := disk.Usage("C:")
			now := time.Now().Format(utils.TimeLayout)
			memData = append(memData, []interface{}{now, m.UsedPercent})
			cpuData = append(cpuData, []interface{}{now, c[0]})
			diskData = append(diskData, []interface{}{now, v.UsedPercent})

			memJSON, err := json.Marshal(memData)
			if err != nil {
				fmt.Println(err)
			}
			cpuJSON, err := json.Marshal(cpuData)
			if err != nil {
				fmt.Println(err)
			}
			diskJSON, err := json.Marshal(diskData)
			if err != nil {
				fmt.Println(err)
			}
			data := fmt.Sprintf("mem=%s&cpu=%s&disk=%s", string(memJSON), string(cpuJSON), string(diskJSON))
			_, err = utils.SyncServices("platform/report/push", data)
			if err != nil {
				fmt.Println(err)
			}
			time.Sleep(duration)
		}
	}()

	select {}
}
