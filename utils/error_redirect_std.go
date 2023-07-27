// +build darwin linux

package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//将标准输出和错误输出重定向到文件，方便追查异常退出（仅适用于linux）
func RedirectStdoutAndStderr() error {
	// 若是运行在容器中, 则不进行重定向, 直接输出到标准输出流
	virtualEnv := os.Getenv("virtual_env")
	if virtualEnv != "" {
		return nil
	}
	strArr := strings.Split(os.Args[0], "/")
	length := len(strArr)
	//默认日志文件名
	crashName := "app"
	if length > 0 {
		crashName = strArr[length-1]
	}
	crashFile, err := os.OpenFile("runtime/"+crashName+".crash", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0644)
	startTime := time.Now().Format("2006-01-02 15:04:05")
	pid := strconv.Itoa(os.Getpid())
	crashFile.WriteString("Start from " + startTime + " pid:" + pid + "\n")
	if err != nil {
		return err
	}
	err = syscall.Dup2(int(crashFile.Fd()), 1)
	if err != nil {
		return fmt.Errorf("redirect stdout failed:%v", err)
	}
	err = syscall.Dup2(int(crashFile.Fd()), 2)
	if err != nil {
		return fmt.Errorf("redirect stderr failed:%v", err)
	}
	return nil
}
