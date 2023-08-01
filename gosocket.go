package gosocket

func Run(config *AppConfig, user IUser, log ILogger, fastLog IFastLogger) {
	if user == nil {
		setAuthUser(&AuthUser{})
	} else {
		setAuthUser(user)
	}
	TcpApp.Run(config, log, fastLog)
}
