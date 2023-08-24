//go:build darwin || linux
// +build darwin linux

package gosocket

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

// OnRestartSuccess is called under the following situations
// syscall.SIGUSR1 the server is about to stop
// syscall.SIGUSR2 the child process has started successfully
// This callback is used to clear the remaining resources
// SIGUSR1时为子进程启动成功的回调，SIGUSR2时为当前进程停止的回调，此函数应用于资源的释放
type OnRestartSuccess func()

var restartManager *RestartManager

// RestartManager is the class to manage the server's restart process
// It will listen to two types of signals:
// syscall.SIGUSR1 the server will shut down gracefully
// syscall.SIGUSR2 the server will restart gracefully, this is often used when the server needs to be upgraded
type RestartManager struct {
	listenerMap     map[int]*net.TCPListener //必须在重启之前取fd，故这里把listener传进来
	restartHandlers []OnRestartSuccess
	isStop          bool
}

// InitGracefulRestart is used to initialize graceful restart
// If you don't call this function, the server won't be able to restart itself through signal
func InitGracefulRestart() {
	restartManager = &RestartManager{
		listenerMap:     make(map[int]*net.TCPListener),
		restartHandlers: make([]OnRestartSuccess, 0),
	}
	//Start listening to signals
	//初始化的时候就开始监听
	go restartManager.handleSignals()
}

// GetRestartManager get the single instance RestartManager
// 获取重启管理器
func GetRestartManager() *RestartManager {
	return restartManager
}

// RegisterHandler call this function to receive a notification once the server is about to restart
// That's the moment the child process has already started
// 注册回调
func (manager *RestartManager) RegisterHandler(restartSuccess OnRestartSuccess) {
	manager.restartHandlers = append(manager.restartHandlers, restartSuccess)
}

// MarkFd records the fd by saving the listener
// If the key is already occupied, it will throw an error
// 0,1,2 stands for stdin, stdout, stderr. So the key should start from 3.
// To reduce waste, it's recommended to use the key in sequence
// 记录文件描述符，如果index已存在，记录失败。0 1 2已被占用，从3开始使用，为避免浪费所有key必须连续
func (manager *RestartManager) MarkFd(key int, listener *net.TCPListener) {
	if key < 3 {
		panic("can't use 0 1 2 as key")
	}
	if _, ok := manager.listenerMap[key]; ok {
		panic("listener key exists!")
	}
	manager.listenerMap[key] = listener
}

// IsStop whether the server has received a restart or exit signal
// 当前是否接受到停止信号
func (manager *RestartManager) IsStop() bool {
	return manager.isStop
}

func (manager *RestartManager) handleSignals() {
	//当前进程id
	pid := os.Getpid()
	//Listen to both syscall.SIGUSR2 and syscall.SIGUSR1
	//注册需要接收的信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR2, syscall.SIGUSR1)
	//Keep receiving signals
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
			fmt.Println("Shutdown gracefully...")
			// notify all callbacks
			for _, restartSuccess := range manager.restartHandlers {
				restartSuccess()
			}
			return
		case syscall.SIGUSR2:
			fmt.Printf("Process %d received SIGUSR2\n", pid)
			fmt.Println("Restart gracefully...")
			err := manager.startNewProcess()
			//If the child process has failed to start, the main process will continue its service
			//如果子进程启动失败，主进程继续服务
			if err != nil {
				fmt.Printf("Start new process failed: %v, process %d continue to serve.\n", err, pid)
			} else {
				//Stop receiving signals to avoid duplicated restart
				//不再接收信号，避免重复启动
				signal.Stop(signalChan)
				// notify all callbacks
				for _, restartSuccess := range manager.restartHandlers {
					restartSuccess()
				}
				break
			}
		}
	}
}

// Start a child process with fork
// 通过fork的方式启动子进程
func (manager *RestartManager) startNewProcess() error {
	// Looking for the biggest key
	//找出最大的key
	maxKey := 0
	for key := range manager.listenerMap {
		if key > maxKey {
			maxKey = key
		}
	}
	//Default files
	files := []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}
	//Since 012 are occupied, start from 3
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
	//Add the graceful environment into the system environment variable list
	//将优雅重启的环境变量追加到系统环境变量中
	environList := make([]string, 0)
	for _, value := range os.Environ() {
		if value != GracefulEnvironString {
			environList = append(environList, value)
		}
	}
	environList = append(environList, GracefulEnvironString)
	execSpec := &syscall.ProcAttr{
		Env:   environList,
		Files: files,
	}
	//Start the new child process by fork
	//通过fork启动子进程
	path := os.Args[0]
	childPid, err := syscall.ForkExec(path, os.Args, execSpec)
	if err != nil {
		return fmt.Errorf("failed to forkexec: %v", err)
	}
	fmt.Printf("Start new process successfully, pid %d\n", childPid)
	return nil
}
