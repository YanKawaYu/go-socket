package main

import (
	"github.com/yankawayu/go-socket"
)

func main() {
	appConfig := &gosocket.AppConfig{
		TcpAddr:   "0.0.0.0",
		TcpPort:   8080,
		TlsEnable: false,
	}
	fastLog := gosocket.GetFastLog("app.access", false)
	//Add ChatController to the router
	gosocket.Router("chat", &ChatController{})
	gosocket.Run(appConfig, &TestUser{}, gosocket.GetLog(false), fastLog)
}
