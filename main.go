package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kardianos/service"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
	_ "net/http/pprof"
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
			fmt.Println(fmt.Sprintf("设备数据同步 %v", time.Now()))
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
		for range ticker.C {
			// 进入当天目录,跳过 23点45 当天凌晨 0点59 分钟，给设备创建目录的时间
			if !((time.Now().Hour() == 0 && time.Now().Minute() < 59) || (time.Now().Hour() == 23 && time.Now().Minute() > 45)) {
				sync.SyncDeviceLog()
			}
			fmt.Println(fmt.Sprintf("设备日志监控同步 %v", time.Now()))
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
		for range ticker.C {
			sync.CheckService()
			fmt.Println(fmt.Sprintf("服务数据同步 %v", time.Now()))
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
		for range ticker.C {
			sync.CheckRestful()
			fmt.Println(fmt.Sprintf("接口监控同步 %v", time.Now()))
		}
		ch <- 1
	}()
	<-ch
}

func (p *program) Stop(s service.Service) error {
	defer log.Println("********** STOP **********")
	return nil
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
	go func() {
		http.ListenAndServe("localhost:6061", nil)
	}()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] [command]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  -action <install remove start stop restart version>\n")
		fmt.Fprintf(os.Stderr, "    程序操作指令\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.Parse()

	// 初始化日志目录
	svcConfig := &service.Config{
		Name:        "LogSync",  //服务显示名称
		DisplayName: "LogSync",  //服务名称
		Description: "同步错误日志信息", //服务描述
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

	if *Action == "sync_device" {
		err = utils.GetToken()
		if err != nil {
			fmt.Println(fmt.Sprintf("get token err %v", err))
			return
		}
		sync.SyncDevice()
		return
	}

	if *Action == "check_service" {
		err = utils.GetToken()
		if err != nil {
			fmt.Println(fmt.Sprintf("get token err %v", err))
			return
		}
		sync.CheckService()
		return
	}

	if *Action == "check_device" {
		err = utils.GetToken()
		if err != nil {
			fmt.Println(fmt.Sprintf("get token err %v", err))
			return
		}
		sync.SyncDeviceLog()
		return
	}

	if *Action == "check_restful" {
		err = utils.GetToken()
		if err != nil {
			fmt.Println(fmt.Sprintf("get token err %v", err))
			return
		}
		sync.CheckRestful()
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
