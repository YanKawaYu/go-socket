package gosocket

import (
	"fmt"
	"sync"
	"time"
)

// 需要等待结束的程序, 当当前的所有程序全部结束之后才能进行重启
type IRestartWaitWork interface {
	GetName() string
	ReceiveWaitChan() bool
	SendWaitChan()
	StartToWait()                  //开始等待
	GetWorkStatus() WaitWorkStatus //修改等待工作的状态，开始等待所有的工作结束
	SetWorkStatus(WaitWorkStatus)  // 结束当前工作
}

type WaitWorkStatus int8

const (
	WaitWorkStatusNone   = WaitWorkStatus(0) //处于闲置状态中
	WaitWorkStatusWait   = WaitWorkStatus(1) //处于等待结束的进程中
	WaitWorkStatusFinish = WaitWorkStatus(2) //已经结束
)

type RestartWaitWorkBase struct {
	finishChan chan bool
	name       string         // 用于日志打印
	status     WaitWorkStatus //当前等待的状态
}

// 生成一个新的等待工作的基础类
func GenerateNewWaitWorkBase(name string) RestartWaitWorkBase {
	return RestartWaitWorkBase{
		finishChan: make(chan bool),
		name:       name,
		status:     WaitWorkStatusNone,
	}
}

func (work *RestartWaitWorkBase) GetName() string {
	return work.name
}

func (work *RestartWaitWorkBase) ReceiveWaitChan() bool {
	return <-work.finishChan
}

func (work *RestartWaitWorkBase) SendWaitChan() {
	work.finishChan <- true
}

func (work *RestartWaitWorkBase) GetWorkStatus() WaitWorkStatus {
	return work.status
}

func (work *RestartWaitWorkBase) SetWorkStatus(status WaitWorkStatus) {
	work.status = status
}

type RestartWaitManager struct {
	WaitWorkMap map[int]IRestartWaitWork
	CountChan   chan int //用于接收其他进程结束的时候的信息
	MaxWaitTime int      //最长等待时间
	SyncLock    *sync.Mutex
}

var restartWaitManager *RestartWaitManager

func init() {
	restartWaitManager = &RestartWaitManager{
		CountChan:   make(chan int),
		WaitWorkMap: make(map[int]IRestartWaitWork),
		SyncLock:    &sync.Mutex{},
	}
}

func GetRestartWaitManager() *RestartWaitManager {
	return restartWaitManager
}

func (manager *RestartWaitManager) SetMaxWaitTime(maxWaitTime int) {
	manager.MaxWaitTime = maxWaitTime
}

// 防止注册等待进程的时候并发, 因此加锁
func (manager *RestartWaitManager) RegisterWaitWork(work IRestartWaitWork) {
	manager.SyncLock.Lock()
	num := len(manager.WaitWorkMap) + 1
	manager.WaitWorkMap[num] = work
	manager.SyncLock.Unlock()
}

func (manager *RestartWaitManager) WaitToRestart() {
	//如果当前没有注册过需要等待的工作, 直接返回，进行下一步
	if manager.WaitWorkMap == nil || len(manager.WaitWorkMap) <= 0 {
		fmt.Println("no work to finish")
		return
	}
	//循环改变每一个需要等待的工作状态，并等待其结束
	for workNum, waitWork := range manager.WaitWorkMap {
		go func() {
			fmt.Printf("wait to finish work: %s. \n", waitWork.GetName())
			//设置状态为等待结束
			waitWork.SetWorkStatus(WaitWorkStatusWait)
			//开始等待
			go waitWork.StartToWait()
			//阻塞直到所有都结束
			waitWork.ReceiveWaitChan()
			manager.CountChan <- workNum //将当前work的序号返回回去
		}()
	}
	count := 0
	timer := time.NewTicker(time.Second * 60 * 5) //如果5分钟所有工作还没有结束，直接自动重启
	//阻塞在当前环节
Loop:
	for {
		select {
		case workNum := <-manager.CountChan:
			currentWaitWork := manager.WaitWorkMap[workNum]
			//如果当前工作已经结束了，直接break
			if currentWaitWork.GetWorkStatus() == WaitWorkStatusFinish {
				break
			}
			//记录数量
			count++
			//修改状态
			currentWaitWork.SetWorkStatus(WaitWorkStatusFinish)
			//判断是否退出循环
			if count >= len(manager.WaitWorkMap) {
				break Loop
			}
		case <-timer.C:
			break Loop
		}
	}
	//打印所有工作是否结束么，如果没有结束也强制结束，返回重启
	failFinishWork := make([]IRestartWaitWork, 0)
	for _, waitWork := range manager.WaitWorkMap {
		if waitWork.GetWorkStatus() != WaitWorkStatusFinish {
			failFinishWork = append(failFinishWork, waitWork)
		}
	}
	//判断是否所有工作已经结束
	if len(failFinishWork) > 0 {
		for _, failWork := range failFinishWork {
			fmt.Printf("%s finish failed, force restart \n", failWork.GetName())
		}
	} else {
		fmt.Printf("all work finished. \n")
	}
}
