// +build windows

package utils

type RestartManager struct {}
//注册信号，windows系统不支持平滑重启，故留空
func GetRestartManager() *RestartManager {
	return &RestartManager{}
}

type OnRestartSuccess func()
func (manager *RestartManager) RegisterHandler(restartSuccess OnRestartSuccess) {}
func (manager *RestartManager) MarkFd(key int, listener *net.TCPListener) {}
