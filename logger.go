package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
)

type myLogLevel int

const (
	None myLogLevel = iota
	Error
	Warn
	Info
	Debug
	All
)

func (l myLogLevel) String() string {
	switch l {
	case None:
		return "NONE"
	case Error:
		return "ERROR"
	case Warn:
		return "WARN"
	case Info:
		return "INFO"
	case Debug:
		return "DEBUG"
	}
	return "Unknown"
}

type fLogger func(lvl myLogLevel, requestId, format string, a ...interface{})

type logger struct {
	filename string
	lvl      myLogLevel
	*log.Logger
}

func (l *logger) log(lvl myLogLevel, requestId, format string, a ...interface{}) {
	if lvl > myLogger.lvl {
		return
	}

	_, file, line, _ := runtime.Caller(2)
	fileParts := strings.Split(file, "/")

	myLogger.Println(fmt.Sprintf("%v:%v %v %v %v", fileParts[len(fileParts)-1], line, requestId, lvl, fmt.Sprintf(format, a...)))
}

var myLogger *logger
var once sync.Once

func NewLogger(fileName string, lvl myLogLevel) fLogger {
	createLogger := func(fname string) *logger {
		file, _ := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)

		return &logger{
			filename: fname,
			lvl:      lvl,
			Logger:   log.New(file, "DataProxy ", log.Ldate|log.Ltime|log.Lmicroseconds),
		}
	}

	once.Do(func() {
		myLogger = createLogger(fileName)
	})
	return func(lvl myLogLevel, requestID, format string, a ...interface{}) {
		myLogger.log(lvl, requestID, format, a...)
	}
}

func GetLogger() fLogger {
	return myLogger.log
}
