package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kardianos/service"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
	//_ "net/http/pprof"
)

var Version string

type program struct {
	httpServer *http.Server
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
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
		ticker := getTicker(t, v)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
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

				memJson, err := json.Marshal(memData)
				if err != nil {
					fmt.Println(err)
				}
				cpuJson, err := json.Marshal(cpuData)
				if err != nil {
					fmt.Println(err)
				}
				diskJson, err := json.Marshal(diskData)
				if err != nil {
					fmt.Println(err)
				}
				data := fmt.Sprintf("mem=%s&cpu=%s&disk=%s", string(memJson), string(cpuJson), string(diskJson))
				_, err = utils.SyncServices("platform/report/push", data)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}()
}

func (p *program) Stop(s service.Service) error {
	defer log.Println("********** STOP **********")
	return nil
}

func syncDevice() {
	t := utils.Config.Data.Timeduration
	v := utils.Config.Data.Timetype
	var ch chan int
	tickerSync := getTicker(t, v)
	defer tickerSync.Stop()
	go func() {
		for range tickerSync.C {
			sync.SyncDevice()
			fmt.Println(fmt.Sprintf("设备数据同步"))
		}
		ch <- 1
	}()
	<-ch
}

func syncDeviceLog() {
	var ch chan int
	go func() {
		t := utils.Config.Device.Timeduration
		v := utils.Config.Device.Timetype
		ticker := getTicker(t, v)
		defer ticker.Stop()

		sync.SyncDeviceLog()
		for range ticker.C {
			// 进入当天目录,跳过 23点45 当天凌晨 0点59 分钟，给设备创建目录的时间
			if !((time.Now().Hour() == 0 && time.Now().Minute() < 59) || (time.Now().Hour() == 23 && time.Now().Minute() > 45)) {
				sync.SyncDeviceLog()
			}
			fmt.Println(fmt.Sprintf("设备日志监控同步"))
		}
		ch <- 1
	}()
	<-ch
}

func syncService() {
	var ch chan int
	t := utils.Config.Device.Timeduration
	v := utils.Config.Device.Timetype
	ticker := getTicker(t, v)
	defer ticker.Stop()
	go func() {
		sync.CheckService()
		for range ticker.C {
			sync.CheckService()
			fmt.Println(fmt.Sprintf("服务数据同步"))
		}
		ch <- 1
	}()
	<-ch
}

func syncRestful() {
	var ch chan int
	t := utils.Config.Restful.Timeduration
	v := utils.Config.Restful.Timetype
	ticker := getTicker(t, v)
	defer ticker.Stop()
	go func() {
		sync.CheckRestful()
		for range ticker.C {
			sync.CheckRestful()
			fmt.Println(fmt.Sprintf("接口监控同步"))
		}
		ch <- 1
	}()
	<-ch
}

func getTicker(t int64, v string) *time.Ticker {
	var ticker *time.Ticker
	switch v {
	case "h":
		ticker = time.NewTicker(time.Hour * time.Duration(t))
	case "m":
		ticker = time.NewTicker(time.Minute * time.Duration(t))
	case "s":
		ticker = time.NewTicker(time.Second * time.Duration(t))
	default:
		ticker = time.NewTicker(time.Minute * time.Duration(t))
	}
	return ticker
}

var Action = flag.String("action", "", "程序操作指令")

func main() {

	//go func() {
	//	http.ListenAndServe("localhost:6061", nil)
	//}()
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
	// 初始化日志目录
	svcConfig := &service.Config{
		Name:             "LogSync",  //服务显示名称
		DisplayName:      "LogSync",  //服务名称
		Description:      "同步错误日志信息", //服务描述
		WorkingDirectory: utils.Config.Outdir,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		fmt.Println(err)
	}

	if err != nil {
		fmt.Println(err)
	}

	if *Action == "install" {
		err = s.Install()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务安装成功"))
		return
	}

	if *Action == "remove" {
		err = s.Uninstall()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务卸载成功"))
		return
	}

	if *Action == "start" {
		err = s.Start()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务启动成功"))
		return
	}

	if *Action == "stop" {
		err = s.Stop()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务停止成功"))
		return
	}

	if *Action == "restart" {
		err = s.Restart()
		if err != nil {
			panic(err)
		}

		fmt.Println(fmt.Sprintf("服务重启成功"))
		return
	}

	if *Action == "version" {
		fmt.Println(fmt.Sprintf(fmt.Sprintf("版本号：%s", Version)))
		return
	}

	s.Run()
}
