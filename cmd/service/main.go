package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kardianos/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/text/encoding/simplifiedchinese"
	"gopkg.in/natefinch/lumberjack.v2"
)

type program struct {
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	defer log.Println("********** STOP **********")
	return nil
}

func (p *program) run() {
	hook := lumberjack.Logger{
		Filename:   "D:\\go\\src\\github.com\\snowlyg\\LogSync\\cmd\\service\\logs\\info.log", // 日志文件路径
		MaxSize:    128,                                                                       // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: 30,                                                                        // 日志文件最多保存多少个备份
		MaxAge:     7,                                                                         // 文件最多保存多少天
		Compress:   true,                                                                      // 是否压缩
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "linenum",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.FullCallerEncoder,      // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	// 设置日志级别
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.InfoLevel)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),                                           // 编码器配置
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&hook)), // 打印到控制台和文件
		atomicLevel, // 日志级别
	)

	// 开启开发模式，堆栈跟踪
	caller := zap.AddCaller()
	// 开启文件及行号
	development := zap.Development()
	// 设置初始化字段
	filed := zap.Fields(zap.String("serviceName", "serviceName"))
	// 构造日志
	logger := zap.New(core, caller, development, filed)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Info("tasklist")
			args := []string{"/C", "tasklist", "/s", "10.0.0.174", "/u", "Administrator", "/p", "123456", "/fi", "IMAGENAME eq App.exe"}
			cmd := exec.Command("cmd.exe", args...)
			logger.Info(fmt.Sprintf("%+v", cmd))
			var out bytes.Buffer
			var outErr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &outErr
			if err := cmd.Run(); err != nil {
				utf8Data := outErr.Bytes()
				if isGBK(outErr.Bytes()) {
					utf8Data, _ = simplifiedchinese.GBK.NewDecoder().Bytes(outErr.Bytes())
					logger.Info(fmt.Sprintf("outErr isUtf8 %v", isUtf8(utf8Data)))
				}
				logger.Info(fmt.Sprintf("outErr isGBK %v", isGBK(outErr.Bytes())))
				logger.Info(fmt.Sprintf("outErr isUtf8 %v", isUtf8(outErr.Bytes())))
				logger.Info(fmt.Sprintf("tasklist %v", string(utf8Data)))
			}
			logger.Info(fmt.Sprintf("out isGBK %v", isGBK(out.Bytes())))
			logger.Info(fmt.Sprintf("out isUtf8 %v", isUtf8(out.Bytes())))
			logger.Info(fmt.Sprintf("%v", out.String()))
			logger.Info("info app count", zap.Int("count:", strings.Count(out.String(), "App.exe")))

		}
	}

}

func preNUm(data byte) int {
	var mask byte = 0x80
	var num int = 0
	//8bit中首个0bit前有多少个1bits
	for i := 0; i < 8; i++ {
		if (data & mask) == mask {
			num++
			mask = mask >> 1
		} else {
			break
		}
	}
	return num
}

func isUtf8(data []byte) bool {
	i := 0
	for i < len(data) {
		if (data[i] & 0x80) == 0x00 {
			// 0XXX_XXXX
			i++
			continue
		} else if num := preNUm(data[i]); num > 2 {
			// 110X_XXXX 10XX_XXXX
			// 1110_XXXX 10XX_XXXX 10XX_XXXX
			// 1111_0XXX 10XX_XXXX 10XX_XXXX 10XX_XXXX
			// 1111_10XX 10XX_XXXX 10XX_XXXX 10XX_XXXX 10XX_XXXX
			// 1111_110X 10XX_XXXX 10XX_XXXX 10XX_XXXX 10XX_XXXX 10XX_XXXX
			// preNUm() 返回首个字节的8个bits中首个0bit前面1bit的个数，该数量也是该字符所使用的字节数
			i++
			for j := 0; j < num-1; j++ {
				//判断后面的 num - 1 个字节是不是都是10开头
				if (data[i] & 0xc0) != 0x80 {
					return false
				}
				i++
			}
		} else {
			//其他情况说明不是utf-8
			return false
		}
	}
	return true
}

func isGBK(data []byte) bool {
	length := len(data)
	var i int = 0
	for i < length {
		if data[i] <= 0x7f {
			//编码0~127,只有一个字节的编码，兼容ASCII码
			i++
			continue
		} else {
			//大于127的使用双字节编码，落在gbk编码范围内的字符
			if data[i] >= 0x81 &&
				data[i] <= 0xfe &&
				data[i+1] >= 0x40 &&
				data[i+1] <= 0xfe &&
				data[i+1] != 0xf7 {
				i += 2
				continue
			} else {
				return false
			}
		}
	}
	return true
}

func main() {
	// action 程序操作指令 install remove start stop version restart
	var action = flag.String("action", "", "程序操作指令")
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
		Name:             "TTL",            //服务显示名称//服务显示名称
		DisplayName:      "TestTaskList",   //服务名称
		Description:      "test task list", //服务描述
		WorkingDirectory: "D:\\go\\src\\github.com\\snowlyg\\LogSync\\cmd\\service",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		fmt.Println(err)
	}

	if err != nil {
		fmt.Println(err)
	}

	if *action == "install" {
		err = s.Install()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务安装成功"))
		return
	}

	if *action == "remove" {
		err = s.Uninstall()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务卸载成功"))
		return
	}

	if *action == "start" {
		err = s.Start()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务启动成功"))
		return
	}

	if *action == "stop" {
		err = s.Stop()
		if err != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("服务停止成功"))
		return
	}

	if *action == "restart" {
		err = s.Restart()
		if err != nil {
			panic(err)
		}

		fmt.Println(fmt.Sprintf("服务重启成功"))
		return
	}

	s.Run()
}
