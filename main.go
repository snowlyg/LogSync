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
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
	"github.com/snowlyg/LogSync/utils/logging"
)

var Version string

type program struct {
	httpServer *http.Server
}

//func (p *program) StartHTTP() {
//	port := utils.Conf().Section("http").Key("port").MustInt64(8001)
//	p.httpServer = &http.Server{
//		Addr:              fmt.Sprintf(":%d", port),
//		Handler:           routers.Router,
//		ReadHeaderTimeout: 5 * time.Second,
//	}
//	link := fmt.Sprintf("http://%s:%d", utils.LocalIP(), port)
//	log.Println("http server start -->", link)
//	go func() {
//		if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
//			log.Println("start http server error", err)
//		}
//		log.Println("http server start")
//	}()
//}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	defer println("********** START **********")
	//err = routers.Init()
	//if err != nil {
	//	panic(err)
	//}
	//
	//p.StartHTTP()

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
	t := utils.Conf().Section("time").Key("sync_data_time").MustInt64(1)
	v := utils.Conf().Section("time").Key("sync_data").MustString("h")
	var ch chan int
	tickerSync := getTicker(t, v)
	go func() {
		for range tickerSync.C {
			err := utils.GetToken()
			if err != nil {
				logging.GetCommonLogger().Infof("get token err %v", err)
				return
			}
			sync.SyncDevice()
			logging.GetCommonLogger().Infof("设备数据同步", time.Now())
		}
		ch <- 1
	}()
	<-ch
}

func syncDeviceLog() {
	var ch chan int
	var t int64
	t = utils.Conf().Section("time").Key("sync_log_time").MustInt64(4)
	v := utils.Conf().Section("time").Key("sync_log").MustString("m")
	ticker := getTicker(t, v)
	go func() {
		for range ticker.C {
			err := utils.GetToken()
			if err != nil {
				logging.GetCommonLogger().Infof("get token err %v", err)
				return
			}

			// 进入当天目录,跳过 23点45 当天凌晨 0点59 分钟，给设备创建目录的时间
			if !((time.Now().Hour() == 0 && time.Now().Minute() < 59) || (time.Now().Hour() == 23 && time.Now().Minute() > 45)) {
				sync.SyncDeviceLog()
			}
			logging.GetCommonLogger().Infof("设备日志监控同步", time.Now())
		}
		ch <- 1
	}()
	<-ch
}

func syncService() {
	var ch chan int
	var t int64
	t = utils.Conf().Section("time").Key("sync_log_time").MustInt64(4)
	v := utils.Conf().Section("time").Key("sync_log").MustString("m")
	ticker := getTicker(t, v)
	go func() {
		for range ticker.C {
			err := utils.GetToken()
			if err != nil {
				logging.GetCommonLogger().Infof("get token err %v", err)
				return
			}
			sync.CheckService()
			logging.GetCommonLogger().Infof("服务数据同步", time.Now())
		}
		ch <- 1
	}()
	<-ch
}

func syncRestful() {
	var ch chan int
	var t int64
	t = utils.Conf().Section("time").Key("sync_log_time").MustInt64(4)
	v := utils.Conf().Section("time").Key("sync_log").MustString("m")
	ticker := getTicker(t, v)
	go func() {
		for range ticker.C {
			err := utils.GetToken()
			if err != nil {
				logging.GetCommonLogger().Infof("get token err %v", err)
				return
			}
			sync.CheckRestful()
			logging.GetCommonLogger().Infof("接口监控同步", time.Now())
		}
		ch <- 1
	}()
	<-ch
}

//func (p *program) StopHTTP() (err error) {
//	if p.httpServer == nil {
//		err = fmt.Errorf("HTTP Server Not Found")
//		return
//	}
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	if err = p.httpServer.Shutdown(ctx); err != nil {
//		return
//	}
//	return
//}

func (p *program) Stop(s service.Service) error {
	defer log.Println("********** STOP **********")
	//defer utils.CloseLogWriter()
	//_ = p.StopHTTP()
	models.Close()
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
		logging.GetCommonLogger().Error(err)
	}

	if err != nil {
		logging.GetCommonLogger().Error(err)
	}

	err = models.Init()
	if err != nil {
		panic(err)
	}

	if *Action == "install" {
		err = s.Install()
		if err != nil {
			panic(err)
		}
		logging.GetCommonLogger().Info("服务安装成功")
		return
	}

	if *Action == "remove" {
		err := s.Uninstall()
		if err != nil {
			panic(err)
		}
		logging.GetCommonLogger().Info("服务卸载成功")
		return
	}

	if *Action == "start" {
		err = s.Start()
		if err != nil {
			panic(err)
		}
		logging.GetCommonLogger().Info("服务启动成功")
		return
	}

	if *Action == "stop" {
		err = s.Stop()
		if err != nil {
			panic(err)
		}
		logging.GetCommonLogger().Info("服务停止成功")
		return
	}

	if *Action == "sync_device" {
		err = utils.GetToken()
		if err != nil {
			logging.GetCommonLogger().Infof("get token err %v", err)
			return
		}
		sync.SyncDevice()
		return
	}

	if *Action == "check_service" {
		err := utils.GetToken()
		if err != nil {
			logging.GetCommonLogger().Infof("get token err %v", err)
			return
		}
		sync.CheckService()
		return
	}

	if *Action == "check_device" {
		err := utils.GetToken()
		if err != nil {
			logging.GetCommonLogger().Infof("get token err %v", err)
			return
		}
		sync.SyncDeviceLog()
		return
	}
	if *Action == "check_restful" {
		err := utils.GetToken()
		if err != nil {
			logging.GetCommonLogger().Infof("get token err %v", err)
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

		logging.GetCommonLogger().Info("服务重启成功")
		return
	}

	if *Action == "version" {
		logging.GetCommonLogger().Info(fmt.Sprintf("版本号：%s", Version))
		return
	}

	s.Run()

}
