// +build darwin linux

package utils

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	GracefulEnvironKey    = "IS_GRACEFUL"
	GracefulEnvironString = GracefulEnvironKey + "=1"
)

// OnRestartSuccess 子进程启动成功的回调
// 同时, 当前进程停止的时候也会调用. 此函数应用于资源的释放
type OnRestartSuccess func()

var restartManager *RestartManager

type RestartManager struct {
	listenerMap     map[int]*net.TCPListener //必须在重启之前取fd，故这里把listener传进来
	restartHandlers []OnRestartSuccess
	isStop          bool
}

func init() {
	restartManager = &RestartManager{
		listenerMap:     make(map[int]*net.TCPListener),
		restartHandlers: make([]OnRestartSuccess, 0),
	}
	//初始化的时候就开始监听
	go restartManager.handleSignals()
}

//获取重启管理器
func GetRestartManager() *RestartManager {
	return restartManager
}

//注册回调
func (manager *RestartManager) RegisterHandler(restartSuccess OnRestartSuccess) {
	manager.restartHandlers = append(manager.restartHandlers, restartSuccess)
}

//记录文件描述符，返回false说明index已存在，记录失败
//0 1 2已被占用，从3开始使用，为避免浪费所有key必须连续
func (manager *RestartManager) MarkFd(key int, listener *net.TCPListener) {
	if key < 3 {
		panic("can't use 0 1 2 as key")
	}
	if _, ok := manager.listenerMap[key]; ok {
		panic("listener key exists!")
	}
	manager.listenerMap[key] = listener
}

// IsStop 当前是否接受到停止信号
func (manager *RestartManager) IsStop() bool {
	return manager.isStop
}

//处理信号
func (manager *RestartManager) handleSignals() {
	//当前进程id
	pid := os.Getpid()
	//注册需要接收的信号
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGUSR2, syscall.SIGUSR1)
	//开始接收
	for {
		sig, ok := <-signalChan
		//如果通道被关闭，跳出循环
		if !ok {
			break
		}
		switch sig {
		case syscall.SIGUSR1:
			timeStr := time.Now().Format("2006-01-02 15:04:05")
			fmt.Println(timeStr + ": receive stop signal")
			manager.isStop = true
			fmt.Println("check work to finish")
			// 准备停止当前进程时等待进程退出
			GetRestartWaitManager().WaitToRestart()
			// 当子进程启动成功后的回调, 一般用于当前进程停止处理
			for _, restartSuccess := range manager.restartHandlers {
				restartSuccess()
			}
			return
		case syscall.SIGUSR2:
			fmt.Printf("Process %d received SIGUSR2\n", pid)
			//校验是否有需要等待结束的工作
			fmt.Println("check work to finish")
			GetRestartWaitManager().WaitToRestart()
			//所有准备工作已经结束，重新启动
			fmt.Println("Restart gracefully...")
			err := manager.startNewProcess()
			//如果子进程启动失败，主进程继续服务
			if err != nil {
				fmt.Printf("Start new process failed: %v, process %d continue to serve.\n", err, pid)
			} else {
				//不再接收信号，避免重复启动
				signal.Stop(signalChan)
				//回调子进程启动成功
				for _, restartSuccess := range manager.restartHandlers {
					restartSuccess()
				}
				break
			}
		}
	}
}

//通过fork的方式启动子进程
func (manager *RestartManager) startNewProcess() error {
	//找出最大的key
	maxKey := 0
	for key := range manager.listenerMap {
		if key > maxKey {
			maxKey = key
		}
	}
	//按照key的顺序传入数组
	files := []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}
	//已有012，故从3开始
	for i := 3; i <= maxKey; i++ {
		if _, ok := manager.listenerMap[i]; ok {
			listener := manager.listenerMap[i]
			file, err := listener.File()
			if err != nil {
				panic(err)
			}
			files = append(files, file.Fd())
		} else {
			//为避免浪费，如果不连续则报错
			panic("key not in sequence")
		}
	}
	//将优雅重启的环境变量追加到系统环境变量中
	environList := make([]string, 0)
	for _, value := range os.Environ() {
		if value != GracefulEnvironString {
			environList = append(environList, value)
		}
	}
	environList = append(environList, GracefulEnvironString)
	//子进程信息
	execSpec := &syscall.ProcAttr{
		Env:   environList,
		Files: files,
	}
	//通过fork启动子进程
	path := os.Args[0]
	childPid, err := syscall.ForkExec(path, os.Args, execSpec)
	if err != nil {
		return fmt.Errorf("failed to forkexec: %v", err)
	}
	fmt.Printf("Start new process successfully, pid %d\n", childPid)
	return nil
}
