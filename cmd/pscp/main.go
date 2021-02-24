package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/snowlyg/LogSync/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/text/encoding/simplifiedchinese"
	"gopkg.in/natefinch/lumberjack.v2"
)

// commandTimeout 可设定超时的 cmd 命令
func commandTimeout(args []string, timeout int, loggerD *zap.Logger, stdin *bytes.Buffer) (stdout, stderr string) {
	cmd := exec.Command("cmd.exe", args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}
	loggerD.Info(fmt.Sprintf("%+v", cmd))
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	cmd.Start()
	// 启动routine等待结束
	done := make(chan error)
	go func() { done <- cmd.Wait() }()
	// 设定超时时间，并select它
	after := time.After(time.Duration(timeout) * time.Second)
	select {
	case <-after:
		cmd.Process.Signal(syscall.SIGINT)
		time.Sleep(time.Second)
		cmd.Process.Kill()
		loggerD.Info(fmt.Sprintf("运行命令（%s %s）超时，超时设定：%v 秒。", cmd, strings.Join(args, " "), timeout))
	case <-done:
		if done != nil {
			loggerD.Info(fmt.Sprintf("%s get error %+v", args[1], done))
		}
	}

	tOut := trimOutput(out)
	tOutErr := trimOutput(outErr)
	if len(tOut) > 0 {
		loggerD.Info(fmt.Sprintf("%s get %s", args[1], tOut))
	}
	if len(tOutErr) > 0 {
		if strings.Contains(tOutErr, "The server's host key is not cached in the registry") {
			return commandTimeout(args, timeout, loggerD, bytes.NewBufferString("y"))
		}
		loggerD.Info(fmt.Sprintf("执行 %s 失败 %s", args[1], tOutErr))
	}
	return tOut, tOutErr
}

// trimOutput 处理输入数据
func trimOutput(buffer bytes.Buffer) string {
	b := bytes.TrimRight(buffer.Bytes(), "\x00")
	if utils.IsGBK(b) {
		b, _ = simplifiedchinese.GBK.NewDecoder().Bytes(b)
	}
	if strings.Contains(string(b), "警告: ") {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(string(b), "\n", ""))
}

func main() {
	hook := lumberjack.Logger{
		Filename:   "D:/go/src/github.com/snowlyg/LogSync/cmd/pscp/logs/info.log", // 日志文件路径
		MaxSize:    128,                                                           // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: 30,                                                            // 日志文件最多保存多少个备份
		MaxAge:     7,                                                             // 文件最多保存多少天
		Compress:   true,                                                          // 是否压缩
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
			if runtime.GOOS == "windows" {
				iDir := fmt.Sprintf("D:/App/data/log/nis/4CEDFB5F7187/%s", "2021-02-23")
				oDir := fmt.Sprintf("D:/go/src/github.com/snowlyg/LogSync/cmd/pscp/logs")
				if !utils.Exist(oDir) {
					_ = utils.CreateDir(oDir)
				}
				args := []string{"/C", "pscp.exe", "-scp", "-r", "-pw", "123456", "-P", "22", fmt.Sprintf("%s@%s:%s", "Administrator", "10.0.0.146", iDir), oDir}
				stdout, stderr := commandTimeout(args, 3, logger, nil)
				logger.Info(stdout)
				logger.Info(stderr)
			}
		}
	}
}
