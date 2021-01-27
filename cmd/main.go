package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("tasklist")
	args := []string{"/C", "tasklist", "/s", "10.0.0.174", "/u", "administrator", "/p", "123456", "/fi", "IMAGENAME eq App.exe"}
	cmd := exec.Command("cmd.exe", args...)
	fmt.Println(fmt.Sprintf("%+v", cmd))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		fmt.Println(fmt.Sprintf("%+v cmd.Run() %+v", cmd, err))
	}

	fmt.Println(fmt.Sprintf("%+v", out.String()))
	fmt.Println(strings.Count(out.String(), "App.exe"))

	time.Sleep(5 * time.Second)
}
