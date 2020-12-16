package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jander/golog/logger"
	"github.com/kardianos/service"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/routers"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
)

var Version string

func init() {
	rotatingHandler := logger.NewRotatingHandler(utils.LogDir(), "logsync.log", 4, 4*1024*1024)
	logger.SetHandlers(logger.Console, rotatingHandler)
}

type program struct {
	httpServer *http.Server
}

func (p *program) StartHTTP() {
	port := utils.Conf().Section("http").Key("port").MustInt64(8001)
	p.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           routers.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	link := fmt.Sprintf("http://%s:%d", utils.LocalIP(), port)
	log.Println("http server start -->", link)
	go func() {
		if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("start http server error", err)
		}
		log.Println("http server start")
	}()
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	err := models.Init()
	if err != nil {
		panic(err)
	}
	err = routers.Init()
	if err != nil {
		panic(err)
	}

	p.StartHTTP()

	go syncDeviceLog()
	go syncDevice()

}

func syncDevice() {
	t := utils.Conf().Section("time").Key("sync_data_time").MustInt64(1)
	v := utils.Conf().Section("time").Key("sync_data").MustString("h")
	var chSy chan int
	var tickerSync *time.Ticker
	switch v {
	case "h":
		tickerSync = time.NewTicker(time.Hour * time.Duration(t))
	case "m":
		tickerSync = time.NewTicker(time.Minute * time.Duration(t))
	case "s":
		tickerSync = time.NewTicker(time.Second * time.Duration(t))
	default:
		tickerSync = time.NewTicker(time.Hour * time.Duration(t))
	}
	go func() {
		for range tickerSync.C {
			utils.GetToken()
			sync.SyncDevice()
		}
		chSy <- 1
	}()
	<-chSy
}

func syncDeviceLog() {
	var ch chan int
	var t int64
	t = utils.Conf().Section("time").Key("sync_log_time").MustInt64(4)
	v := utils.Conf().Section("time").Key("sync_log").MustString("m")
	var ticker *time.Ticker

	ticker = time.NewTicker(time.Hour * time.Duration(t))
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
	sync.NotFirst = false
	go func() {
		for range ticker.C {
			utils.GetToken()
			sync.CheckRestful()
			sync.CheckService()
			// 进入当天目录,跳过 23点45 当天凌晨 0点15 分钟，给设备创建目录的时间
			if !((time.Now().Hour() == 0 && time.Now().Minute() < 15) || (time.Now().Hour() == 23 && time.Now().Minute() > 45)) {
				sync.SyncDeviceLog()
			}
			sync.NotFirst = true
		}
		ch <- 1
	}()
	<-ch
}

func (p *program) StopHTTP() (err error) {
	if p.httpServer == nil {
		err = fmt.Errorf("HTTP Server Not Found")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = p.httpServer.Shutdown(ctx); err != nil {
		return
	}
	return
}

func (p *program) Stop(s service.Service) error {
	defer log.Println("********** STOP **********")
	defer utils.CloseLogWriter()
	_ = p.StopHTTP()
	models.Close()
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "LogSync",  //服务显示名称
		DisplayName: "LogSync",  //服务名称
		Description: "同步错误日志信息", //服务描述
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Println(err)
	}

	if err != nil {
		logger.Println(err)
	}

	if len(os.Args) == 2 {
		if os.Args[1] == "install" {
			err := s.Install()
			if err != nil {
				panic(err)
			}
			logger.Println("服务安装成功")
			return
		}

		if os.Args[1] == "remove" {
			err := s.Uninstall()
			if err != nil {
				panic(err)
			}
			logger.Println("服务卸载成功")
			return
		}

		if os.Args[1] == "start" {
			err := s.Start()
			if err != nil {
				panic(err)
			}
			logger.Println("服务启动成功")
			return
		}

		if os.Args[1] == "stop" {
			err := s.Stop()
			if err != nil {
				panic(err)
			}
			logger.Println("服务停止成功")
			return
		}

		if os.Args[1] == "restart" {
			err := s.Restart()
			if err != nil {
				panic(err)
			}

			logger.Println("服务重启成功")
			return
		}

		if os.Args[1] == "version" {
			fmt.Println(fmt.Sprintf("版本号：%s", Version))
			return
		}
	}

	err = s.Run()
	if err != nil {
		logger.Println(err)
	}
}
