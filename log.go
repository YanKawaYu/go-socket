//go:build darwin || linux
// +build darwin linux

package gosocket

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"
	"sync"
	"time"
)

// ILogger is an interface for logger class in the framework
// The purpose of defining this interface is to make sure the framework won't heavily rely on go.uber.org/zap
// All these functions come from zap.SugaredLogger
//
// ILogger has many more functions than IFastLogger, since ILogger values function more than performance
// 用于打印普通日志，包括信息、错误等
type ILogger interface {
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Error(err interface{})
	Errorf(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
}

// IFastLogger is an interface for logger class in the framework
// The purpose of defining this interface is to make sure the framework won't heavily rely on go.uber.org/zap
// All these functions come from zap.Logger
// 用于打印访问日志
type IFastLogger interface {
	Info(msg string, fields ...zapcore.Field)
	Debug(msg string, fields ...zapcore.Field)
}

// Get the configuration of log
// 获取日志配置
func getLogConfig(logName string, isDebug bool) *zap.Logger {
	var config zap.Config
	//Whether to use debug config
	//根据debug与否选择不同的log配置
	if isDebug {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	// The caller is useless, disable it to reduce the size of the log
	// 不记录log的调用者, 对此内容没有使用. 可减少日志文件的大小
	config.DisableCaller = true
	//WARNING! This line is essential. By default, the zap.Logger will try to sample the log to boost performance
	//But this will cause the log to be incomplete, therefore we need to disable the sampling
	//大坑！！！不加会导致线上日志不全
	config.Sampling = nil
	config.OutputPaths = []string{"runtime/" + logName + ".log"}
	//Change the time format for the log
	//修改时间格式
	config.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

func GetLog(isDebug bool) *Log {
	//Get the executable name from the arguments
	//日志文件名
	strArr := strings.Split(os.Args[0], string(os.PathSeparator))
	length := len(strArr)
	//Set the executable name as the default log name
	//默认日志文件名
	logName := ""
	if length > 0 {
		logName = strArr[length-1]
	}
	if logName == "" {
		logName = "app"
	}
	return &Log{
		sugarLogger: getLogConfig(logName, isDebug).Sugar(),
	}
}

// Log is a wrapper for zap.SugaredLogger
// This class is to make sure the framework won't have too much dependency on go.uber.org/zap
type Log struct {
	sugarLogger *zap.SugaredLogger
}

// Fatal is equivalent to l.Critical(fmt.Sprint()) followed by a call to os.Exit(1).
func (log *Log) Fatal(args ...interface{}) {
	log.sugarLogger.Fatal(args)
}

// Fatalf is equivalent to l.Critical followed by a call to os.Exit(1).
func (log *Log) Fatalf(format string, args ...interface{}) {
	log.sugarLogger.Fatalf(format, args)
}

// Panic is equivalent to l.Critical(fmt.Sprint()) followed by a call to panic().
func (log *Log) Panic(args ...interface{}) {
	log.sugarLogger.Panic(args)
}

// Panicf is equivalent to l.Critical followed by a call to panic().
func (log *Log) Panicf(format string, args ...interface{}) {
	log.sugarLogger.Panicf(format, args)
}

// Error logs a message using ERROR as log level.
//func (log *Log) Error(args ...interface{}) {
//	log.sugarLogger.Error(args)
//}

// Error logs a message with the stack trace, it's more powerful than the original Error
// 可同时打印原始错误发生时的堆栈，同时兼容上面Error函数的功能，故替换
func (log *Log) Error(err interface{}) {
	log.sugarLogger.Errorf("%+v\n", err)
}

// Errorf logs a message using ERROR as log level.
func (log *Log) Errorf(format string, args ...interface{}) {
	log.sugarLogger.Errorf(format, args)
}

// Warning logs a message using WARNING as log level.
func (log *Log) Warning(args ...interface{}) {
	log.sugarLogger.Warn(args)
}

// Warningf logs a message using WARNING as log level.
func (log *Log) Warningf(format string, args ...interface{}) {
	log.sugarLogger.Warnf(format, args)
}

// Info logs a message using INFO as log level.
func (log *Log) Info(args ...interface{}) {
	log.sugarLogger.Info(args)
}

// Infof logs a message using INFO as log level.
func (log *Log) Infof(format string, args ...interface{}) {
	log.sugarLogger.Infof(format, args)
}

// Debug logs a message using DEBUG as log level.
func (log *Log) Debug(args ...interface{}) {
	log.sugarLogger.Debug(args)
}

// Debugf logs a message using DEBUG as log level.
func (log *Log) Debugf(format string, args ...interface{}) {
	log.sugarLogger.Debugf(format, args)
}

// FastLog is a wrapper for zap.Logger
// This class is to make sure the framework won't have too much dependency on go.uber.org/zap
// The FastLog has a much higher performance than the Log, make sure you choose the right one according to your situation
type FastLog struct {
	logger *zap.Logger
}

var fastLogMap = map[string]*FastLog{}
var fastLogLock sync.RWMutex

func GetFastLog(logName string, isDebug bool) *FastLog {
	fastLog, ok := fastLogMap[logName]
	if !ok {
		//加读写锁，防止map多线程访问
		fastLogLock.RLock()
		fastLog, ok = fastLogMap[logName]
		fastLogLock.RUnlock()
		if !ok {
			fastLog = &FastLog{
				logger: getLogConfig(logName, isDebug),
			}
			fastLogLock.Lock()
			fastLogMap[logName] = fastLog
			fastLogLock.Unlock()
		}
	}
	return fastLog
}

func (fastLog *FastLog) Info(msg string, fields ...zapcore.Field) {
	fastLog.logger.Info(msg, fields...)
}

func (fastLog *FastLog) Debug(msg string, fields ...zapcore.Field) {
	fastLog.logger.Debug(msg, fields...)
}
