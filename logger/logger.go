package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"time"
)

var logFile *os.File

const logFileName = "logs/filesync.log"

func createLogDir() {
	const dir = "logs"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 创建目录
		err := os.Mkdir(dir, 0755) // 0755表示具有读写执行权限的目录
		if err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		} else {
			fmt.Println("Directory created successfully.")
		}
	}
}

func init() {
	createLogDir()
	checkLogFile()
}

func createLogFile() {
	if logFile != nil {
		logFile.Close()
	}
	lF, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	logFile = lF
}

func checkLogFile() {
	if fileInfo, err := os.Stat(logFileName); os.IsNotExist(err) {
		createLogFile()
	} else if fileInfo.Size() >= 1*1024*1024 || logFile == nil {
		// logFile.Close()
		os.Rename(logFileName, "logs/filesync_"+time.Now().Format("20060102150405")+".log")
		createLogFile()
	} else {
		return
	}
	log.SetOutput(io.MultiWriter(logFile, os.Stdout))
	log.SetFlags(0)
}

func getCallerInfo(skip int) (info string) {

	_, file, lineNo, ok := runtime.Caller(skip)
	if !ok {

		info = "runtime.Caller() failed"
		return
	}
	fileName := path.Base(file) // Base函数返回路径的最后一个元素
	now := time.Now()
	// 使用自定义格式字符串格式化时间
	formattedTime := now.Format("2006-01-02 15:04:05")
	return fmt.Sprintf("%s file:%s, line:%d ", formattedTime, fileName, lineNo)
}

func Info(msg string, args ...any) {
	checkLogFile()
	log.SetPrefix("[INFO] " + getCallerInfo(2) + " ")
	log.Printf(msg+"\n", args...)
}

func Error(msg string, args ...any) {
	checkLogFile()
	log.SetPrefix("[ERROR] " + getCallerInfo(2) + " ")
	log.Printf(msg+"\n", args...)
}

func Writer() io.Writer {
	return log.Writer()
}
