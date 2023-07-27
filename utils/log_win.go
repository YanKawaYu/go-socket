// +build windows

package utils

import "go.uber.org/zap/zapcore"

func GenerateLog(isDebug bool) *Log {
	return &Log{}
}

type Log struct {}

// Fatal is equivalent to l.Critical(fmt.Sprint()) followed by a call to os.Exit(1).
func (log *Log) Fatal(args ...interface{}) {}

// Fatalf is equivalent to l.Critical followed by a call to os.Exit(1).
func (log *Log) Fatalf(format string, args ...interface{}) {}

// Panic is equivalent to l.Critical(fmt.Sprint()) followed by a call to panic().
func (log *Log) Panic(args ...interface{}) {}

// Panicf is equivalent to l.Critical followed by a call to panic().
func (log *Log) Panicf(format string, args ...interface{}) {}

// Error logs a message using ERROR as log level.
//func (log *Log) Error(args ...interface{}) {
//	log.sugarLogger.Error(args)
//}

//可同时打印原始错误发生时的堆栈，同时兼容上面Error函数的功能，故替换
func (log *Log) Error(err interface{}) {}

// Errorf logs a message using ERROR as log level.
func (log *Log) Errorf(format string, args ...interface{}) {}

// Warning logs a message using WARNING as log level.
func (log *Log) Warning(args ...interface{}) {}

// Warningf logs a message using WARNING as log level.
func (log *Log) Warningf(format string, args ...interface{}) {}

// Info logs a message using INFO as log level.
func (log *Log) Info(args ...interface{}) {}

// Infof logs a message using INFO as log level.
func (log *Log) Infof(format string, args ...interface{}) {}

// Debug logs a message using DEBUG as log level.
func (log *Log) Debug(args ...interface{}) {}

// Debugf logs a message using DEBUG as log level.
func (log *Log) Debugf(format string, args ...interface{}) {}

type FastLog struct {}

func GetFastLog(logName string) *FastLog {
	fastLog := &FastLog{}
	return fastLog
}

func (fastLog *FastLog) Info(msg string, fields ...zapcore.Field) {}