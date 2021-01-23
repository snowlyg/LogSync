package main

import (
	"flag"
	"fmt"
	"github.com/snowlyg/LogSync/sync"
	"github.com/snowlyg/LogSync/utils"
	"os"
	"time"
)

var Action = flag.String("action", "", "程序操作指令")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] [command]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  -action <del plugin device real>\n")
		fmt.Fprintf(os.Stderr, "    程序操作指令\n")
		fmt.Fprintf(os.Stderr, "\n")
	}
	flag.Parse()

	if *Action == "del" {
		for _, device := range GetDevices() {
			err := os.RemoveAll(GetPath(device))
			if err != nil {
				fmt.Println(err)
			}
		}
		fmt.Println(fmt.Sprintf("删除日志文件"))
		return
	}

	location, err := utils.GetLocation()
	if err != nil {
		fmt.Println(err)
	}

	if *Action == "device" {
		for _, device := range GetDevices() {
			err = os.MkdirAll(GetPath(device), 0777)
			if err != nil {
				fmt.Println(err)
			}
			plugin := sync.Plugin{Code: "1", Reason: "已就绪"}
			interf := sync.Plugin{Code: "1", Reason: "OK"}
			timestamp := time.Now().Add(30 * time.Minute).In(location).Format(utils.DateTimeLayout)
			err = CreateFaultFile(device, plugin, interf, timestamp, "true", "已就绪")
			if err != nil {
				fmt.Println(err)
			}
		}
		fmt.Println(fmt.Sprintf("生成时间异常文件"))
		return
	}

	if *Action == "plugin" {
		for _, device := range GetDevices() {
			err := os.MkdirAll(GetPath(device), 0777)
			if err != nil {
				fmt.Println(err)
			}
			plugin := sync.Plugin{Code: "4", Reason: "连接失败"}
			interf := sync.Plugin{Code: "3", Reason: "连接失败"}
			timestamp := time.Now().In(location).Format(utils.DateTimeLayout)
			err = CreateFaultFile(device, plugin, interf, timestamp, "false", "连接失败")
			if err != nil {
				fmt.Println(err)
			}
		}
		fmt.Println(fmt.Sprintf("生成插件异常文件"))
		return
	}

	if *Action == "real" {
		for _, device := range GetDevices() {
			err = os.MkdirAll(GetPath(device), 0777)
			if err != nil {
				fmt.Println(err)
			}
			plugin := sync.Plugin{Code: "1", Reason: "已就绪"}
			interf := sync.Plugin{Code: "1", Reason: "OK"}
			timestamp := time.Now().In(location).Format(utils.DateTimeLayout)
			err = CreateFaultFile(device, plugin, interf, timestamp, "true", "已就绪")
			if err != nil {
				fmt.Println(err)
			}
		}
		fmt.Println(fmt.Sprintf("生成正常文件"))
		return
	}
}
