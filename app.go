package gosocket

import (
	"crypto/tls"
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

// App is the entry class to start the server
type App struct {
	Server  *Server     //Responsible for listening and serving requests
	Config  *AppConfig  //Same as appConfig in the Run function
	Log     ILogger     //Same as log in the Run function
	FastLog IFastLogger //Same as fastLog in the Run function
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
		}
	}()
	if log == nil || fastLog == nil {
		panic("log or fastLog can't be nil")
	}
	app.Config = appConfig
	app.Log = log
	app.FastLog = fastLog
	//Initialize graceful restart
	InitGracefulRestart()
	//创建一个server
	app.Server = NewServer(app.Config.TcpAddr)
	//Whether to enable tls
	if app.Config.TlsEnable {
		//tls certificate
		config := &tls.Config{}
		certificate, err := tls.LoadX509KeyPair(app.Config.TlsCert, app.Config.TlsKey)
		if err != nil {
			panic(err)
		}
		config.Certificates = []tls.Certificate{certificate}
		app.Server.ListenAndServe(config)
	} else {
		app.Server.ListenAndServe(nil)
	}
}
