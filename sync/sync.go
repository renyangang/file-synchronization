package sync

import (
	"bufio"
	"encoding/json"
	"fmt"
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
var DstSyncFileMap = make(map[string]*SyncFileInfo)
var excludeMap = make(map[string]bool)

func init() {
	if config.InstanceConfig.Sync.Excludefrom != "" {
		file, err := os.Open(config.InstanceConfig.Sync.Excludefrom)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			path := filepath.Clean(scanner.Text())
			fmt.Println("exclude path:", path)
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
	relPath := strings.Replace(path, config.InstanceConfig.Sync.Srcpath, "", 1)
	if strings.IndexRune(relPath, os.PathSeparator) == 0 {
		relPath = relPath[1:]
	}

	filepaths := strings.Split(relPath, string(os.PathSeparator))
	for i := 0; i < len(filepaths); i++ {
		if excludeMap[filepaths[i]] {
			return nil
		}
	}

	if path != config.InstanceConfig.Sync.Srcpath && !excludeMap[info.Name()] && !excludeMap[relPath] && !excludeMap[path] {
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
	fetchDir(config.InstanceConfig.Sync.Srcpath)
	saveCacheFile(srcSyncFileMap, config.InstanceConfig.Sync.Cachefile)
}

func MakeDirInfo(path string) map[string]*SyncFileInfo {
	fetchDir(path)
	return srcSyncFileMap
}

func loadCacheFile(path string) map[string]*SyncFileInfo {
	tempMap := make(map[string]*SyncFileInfo)
	// 读取JSON文件
	jsonData, err := os.ReadFile(path)
	if err != nil {
		logger.Error("read cache file: %v failed. Error: %v", path, err)
		return nil
	}
	err = json.Unmarshal(jsonData, &tempMap)
	if err != nil {
		logger.Error("Unmarshal Json failed. Error: %v", err)
		return nil
	}
	return tempMap
}

func LoadSrcCache() {
	// 读取JSON文件
	tempMap := loadCacheFile(config.InstanceConfig.Sync.Cachefile)
	if tempMap != nil {
		for k, v := range tempMap {
			srcSyncFileMap[filepath.FromSlash(k)] = v
		}
	} else {
		logger.Error("load cache file: %v failed.", config.InstanceConfig.Sync.Cachefile)
	}
}

func loadDstCache() {
	// 读取JSON文件
	tempMap := loadCacheFile(config.InstanceConfig.Sync.Dstcachefile)
	if tempMap != nil {
		for k, v := range tempMap {
			DstSyncFileMap[filepath.FromSlash(k)] = v
		}
	} else {
		logger.Error("load cache file: %v failed.", config.InstanceConfig.Sync.Dstcachefile)
	}
}

func Compare() map[string]*SyncFileInfo {
	diffFiles := make(map[string]*SyncFileInfo)
	// 比较差异文件
	for filePath, fileInfo := range srcSyncFileMap {
		if _, ok := DstSyncFileMap[filePath]; !ok {
			// 文件在源目录但不在目标目录，需要上传
			logger.Info("File %s is not exist in dst, need sync.", filePath)
			diffFiles[filePath] = fileInfo
		} else if !fileInfo.IsDir && (DstSyncFileMap[filePath].ModTime.Before(fileInfo.ModTime) || fileInfo.Size != DstSyncFileMap[filePath].Size) {
			logger.Info("File %s is modified in src, need sync.", filePath)
			diffFiles[filePath] = fileInfo
		}
	}
	return diffFiles
}

func CompareDiffFiles() map[string]*SyncFileInfo {
	syncOper := makeSyncOper()
	diffFiles, err := syncOper.CompareDiffFiles()
	if err != nil {
		logger.Error("CompareDiffFiles failed. Error: %v", err)
		return nil
	}
	return diffFiles
}

func makeSyncOper() SyncOper {
	return &OsSyncOper{}
}

func syncFiles(diffFiles map[string]*SyncFileInfo) {
	syncOper := makeSyncOper()
	syncOper.SyncFiles(diffFiles)
}

func DoSync() {
	diffFiles := CompareDiffFiles()
	syncFiles(diffFiles)
	saveCacheFile(DstSyncFileMap, config.InstanceConfig.Sync.Dstcachefile)
}
