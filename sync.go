package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stacktrace.top/filesync/config"
	"stacktrace.top/filesync/logger"
)

type SyncFileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	IsDir   bool
}

var srcSyncFileMap = make(map[string]*SyncFileInfo)
var dstSyncFileMap = make(map[string]*SyncFileInfo)
var excludeMap = make(map[string]bool)

func init() {
	if config.ServerConfig.Sync.Excludefrom != "" {
		scanner := bufio.NewScanner(strings.NewReader(config.ServerConfig.Sync.Excludefrom))
		for scanner.Scan() {
			path := filepath.Clean(scanner.Text())
			excludeMap[path] = true
		}
		if err := scanner.Err(); err != nil {
			logger.Error("read exclude-from file failed.err: %v", err)
		}
	}
}

func visit(path string, info os.FileInfo, err error) error {
	if err != nil {
		logger.Error("visit for path: %v failed.err: %v", path, err)
		return nil
	}
	relPath := strings.Replace(path, config.ServerConfig.Sync.Srcpath, "", 1)
	if strings.IndexRune(relPath, os.PathSeparator) == 0 {
		relPath = relPath[1:]
	}
	if path != config.ServerConfig.Sync.Srcpath && !excludeMap[info.Name()] && !excludeMap[relPath] && !excludeMap[path] && !excludeMap[filepath.Base(filepath.Dir(path))+string(os.PathSeparator)] {
		srcSyncFileMap[relPath] = &SyncFileInfo{
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			IsDir:   info.IsDir(),
		}
	}
	return nil
}

func fetchDir(rootDir string) {
	// 使用filepath.Walk来递归遍历目录
	err := filepath.Walk(rootDir, visit)
	if err != nil {
		logger.Error("fetchDir for path: %v failed.err: %v", rootDir, err)
	}
}

func saveCacheFile(fileMap map[string]*SyncFileInfo, filepath string) {
	jsonData, err := json.Marshal(fileMap)
	if err != nil {
		logger.Error("Marshal Json failed. Error: %v", err)
		return
	}

	// 将JSON数据写入文件
	err = os.WriteFile(filepath, jsonData, 0644)
	if err != nil {
		logger.Error("write cache file: %v failed. Error: %v", filepath, err)
		return
	}
}

func MakeSrcInfo() {
	fetchDir(config.ServerConfig.Sync.Srcpath)
	saveCacheFile(srcSyncFileMap, config.ServerConfig.Sync.Cachefile)
}

func loadSrcCache() {
	// 读取JSON文件
	jsonData, err := os.ReadFile(config.ServerConfig.Sync.Cachefile)
	if err != nil {
		logger.Error("read cache file: %v failed. Error: %v", config.ServerConfig.Sync.Cachefile, err)
		return
	}
	err = json.Unmarshal(jsonData, &srcSyncFileMap)
	if err != nil {
		logger.Error("Unmarshal Json failed. Error: %v", err)
		return
	}
}

func loadDstCache() {
	// 读取JSON文件
	jsonData, err := os.ReadFile(config.ServerConfig.Sync.Dstcachefile)
	if err != nil {
		logger.Error("read cache file: %v failed. Error: %v", config.ServerConfig.Sync.Dstcachefile, err)
		return
	}
	err = json.Unmarshal(jsonData, &dstSyncFileMap)
	if err != nil {
		logger.Error("Unmarshal Json failed. Error: %v", err)
		return
	}
}

func CompareDiffFiles() map[string]*SyncFileInfo {
	loadSrcCache()
	loadDstCache()
	diffFiles := make(map[string]*SyncFileInfo)
	// 比较差异文件
	for filePath, fileInfo := range srcSyncFileMap {
		if _, ok := dstSyncFileMap[filePath]; !ok {
			// 文件在源目录但不在目标目录，需要上传
			logger.Info("File %s is not exist in dst, need sync.", filePath)
			diffFiles[filePath] = fileInfo
		} else if !fileInfo.IsDir && (dstSyncFileMap[filePath].ModTime.Before(fileInfo.ModTime) || fileInfo.Size != dstSyncFileMap[filePath].Size) {
			logger.Info("File %s is modified in src, need sync.", filePath)
			diffFiles[filePath] = fileInfo
		}
	}
	return diffFiles
}

func makeSyncOper() SyncOper {
	return &OsSyncOper{}
}

func syncFiles(diffFiles map[string]*SyncFileInfo) {
	syncOper := makeSyncOper()
	for filePath, fileInfo := range diffFiles {
		logger.Info("sync file: %v", filePath)
		srcFilePath := filepath.Join(config.ServerConfig.Sync.Srcpath, filePath)
		dstFilePath := filepath.Join(config.ServerConfig.Sync.Dstpath, filePath)
		err := syncOper.SyncFile(srcFilePath, dstFilePath, fileInfo)
		if err == nil {
			dstSyncFileMap[filePath] = fileInfo
		}
	}
}

func DoSync() {
	diffFiles := CompareDiffFiles()
	syncFiles(diffFiles)
	saveCacheFile(dstSyncFileMap, config.ServerConfig.Sync.Dstcachefile)
}
