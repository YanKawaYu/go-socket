package gotcp

import "github.com/yankawayu/go-socket/utils"

func Run(config *AppConfig, log utils.ILogger, fastLog utils.IFastLogger) {
	TcpApp.Run(config, log, fastLog)
}