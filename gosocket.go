package gosocket

import "github.com/yankawayu/go-socket/utils"

func Run(config *AppConfig, user IUser, log utils.ILogger, fastLog utils.IFastLogger) {
	if user == nil {
		setAuthUser(&AuthUser{})
	} else {
		setAuthUser(user)
	}
	TcpApp.Run(config, log, fastLog)
}
