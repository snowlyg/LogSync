package main

import (
	"context"
	"fmt"
	"github.com/snowlyg/LogSync/routers"
	"github.com/snowlyg/LogSync/sync"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jander/golog/logger"
	"github.com/kardianos/service"
	"github.com/snowlyg/LogSync/models"
	"github.com/snowlyg/LogSync/utils"
)

func init() {
	rotatingHandler := logger.NewRotatingHandler(utils.LogDir(), "logsync.log", 4, 4*1024*1024)

	// logger set handlers: console, rotating
	logger.SetHandlers(logger.Console, rotatingHandler)
}

type program struct {
	httpServer *http.Server
}

// StartRTSP 启动 rtsp
func (p *program) StartTCP() {
	p.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", 8001),
		Handler:           routers.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	link := fmt.Sprintf("http://%s:%d", utils.LocalIP(), 8001)
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
	// 初始化路由
	err = routers.Init()
	if err != nil {
		return
	}

	//defer models.Close()
	p.StartTCP()
	go syncDeviceLog()
	go syncDevice()

}

func syncDevice() {
	t := utils.Conf().Section("time").Key("sync_data").MustString("h")
	var chSy chan int
	var tickerSync *time.Ticker
	switch t {
	case "h":
		tickerSync = time.NewTicker(time.Hour * 1)
	case "m":
		tickerSync = time.NewTicker(time.Minute * 1)
	case "s":
		tickerSync = time.NewTicker(time.Second * 1)
	default:
		tickerSync = time.NewTicker(time.Hour * 1)
	}
	go func() {
		for range tickerSync.C {
			sync.SyncDevice()
		}

		chSy <- 1
	}()
	<-chSy
}

func syncDeviceLog() {
	var ch chan int
	t := utils.Conf().Section("time").Key("sync_data").MustString("m")
	var ticker *time.Ticker
	switch t {
	case "h":
		ticker = time.NewTicker(time.Hour * 4)
	case "m":
		ticker = time.NewTicker(time.Minute * 4)
	case "s":
		ticker = time.NewTicker(time.Second * 4)
	default:
		ticker = time.NewTicker(time.Hour * 4)
	}
	go func() {
		for range ticker.C {
			sync.SyncDeviceLog()
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
		logger.Error(err)
	}

	if err != nil {
		logger.Error(err)
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "install" {
			_ = s.Install()
			logger.Println("服务安装成功")
			return
		}

		if os.Args[1] == "remove" {
			_ = s.Uninstall()
			logger.Println("服务卸载成功")
			return
		}
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
