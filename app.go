package gosocket

import (
	"crypto/tls"
	"fmt"
	"os"
)

var (
	TcpApp *App
)

func init() {
	TcpApp = NewApp()
}

// AppConfig Server configuration
// 服务器配置
type AppConfig struct {
	TcpAddr   string //The address which the app is listening to 地址
	TcpPort   int    //The port which the app is listening to 端口号
	TlsEnable bool   //Whether to enable tls 是否开启TLS
	TlsCert   string //The certification used by tls TLS证书
	TlsKey    string //The key used by tls TLS密钥
}

type App struct {
	Server  *Server
	Log     ILogger
	FastLog IFastLogger
	Config  *AppConfig
}

func NewApp() *App {
	app := &App{}
	return app
}

// Run start the app server
// 启动服务器
// The appConfig is used to configure the server
// The log is used to log any errors happens during the process
// The fastLog is used to log any requests received by the server
// The requests can be very frequent therefore using the fast logger will make sure the performance is high
func (app *App) Run(appConfig *AppConfig, log ILogger, fastLog IFastLogger) {
	defer func() {
		if e := recover(); e != nil {
			TcpApp.Log.Error(e)
			//同时直接输出到控制台，方便查错
			fmt.Println(e)
		}
	}()
	if log == nil || fastLog == nil {
		panic("log or fastLog can't be nil")
	}
	//配置文件
	app.Config = appConfig
	//日志
	app.Log = log
	app.FastLog = fastLog
	//从环境变量中判断是否为优雅重启
	isGraceful := false
	if os.Getenv(GracefulEnvironKey) != "" {
		isGraceful = true
		//initialize graceful restart
		InitGracefulRestart()
	}
	//创建一个server
	app.Server = NewServer(app.Config.TcpAddr, isGraceful)
	//如果开启了Tls
	if app.Config.TlsEnable {
		//tls证书配置
		config := &tls.Config{}
		certificate, err := tls.LoadX509KeyPair(app.Config.TlsCert, app.Config.TlsKey)
		if err != nil {
			panic(err)
		}
		config.Certificates = []tls.Certificate{certificate}
		//TSL启动服务器
		app.Server.ListenAndServe(config)
	} else {
		//非TSL启动服务器
		app.Server.ListenAndServe(nil)
	}
}
