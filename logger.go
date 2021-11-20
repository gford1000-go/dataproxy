package main

import (
    "log"
    "os"
    "sync"
)

type logger struct {
    filename string
    *log.Logger
}

var myLogger *logger
var once sync.Once

func NewLogger(fileName string) *logger {
	createLogger := func(fname string) *logger {
		file, _ := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	
		return &logger{
			filename: fname,
			Logger:   log.New(file, "DataProxy ", log.Lshortfile | log.Ldate | log.Ltime | log.Lmicroseconds ),
		}
	}

	once.Do(func() {
        myLogger = createLogger(fileName)
    })
	return GetLogger()
}

func GetLogger() *logger {
    return myLogger
}

