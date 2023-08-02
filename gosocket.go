package gosocket

// Run start the server
// The appConfig is used to configure the server
// The user is used to customize the user identification process
// The log is used to log any errors happens during the process
// The fastLog is used to log any requests received by the server
// The requests can be very frequent therefore using the fast logger will make sure the performance is high
func Run(config *AppConfig, user IUser, log ILogger, fastLog IFastLogger) {
	if user == nil {
		setAuthUser(&AuthUser{})
	} else {
		setAuthUser(user)
	}
	TcpApp.Run(config, log, fastLog)
}
