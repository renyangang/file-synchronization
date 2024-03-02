package net

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"

	"stacktrace.top/filesync/logger"
	"stacktrace.top/filesync/sync"
)

const (
	MSG_TOKEN     = 0
	MSG_MAKECACHE = 1
	MSG_SYNC      = 2
	MSG_FILEPART  = 3
)

const (
	RES_SUCCESS = 0
)

type SyncCmdMsg struct {
	MsgType  uint32
	Token    string
	DstDir   string
	SyncInfo *sync.SyncFileInfo
}

type SyncRespMsg struct {
	MsgType   uint32
	ResCode   int
	Err       error
	OffSet    int64
	PartSize  int64
	FileInfos map[string]*sync.SyncFileInfo
}

func ReadForSyncMsg(conn *net.TCPConn) (*SyncCmdMsg, error) {
	// 读取消息长度
	var msgLen uint32
	err := binary.Read(conn, binary.BigEndian, &msgLen)
	if err != nil {
		logger.Error("read msg len failed. conn: %v err: %v", conn, err)
		return nil, err
	}
	// 读取消息内容
	msgBytes := make([]byte, msgLen)
	readLen := 0
	for readLen < int(msgLen) {
		n, err := conn.Read(msgBytes)
		if err != nil {
			logger.Error("read msg failed. err: %v", err)
			return nil, err
		}
		readLen += n
	}

	// 解析消息
	msg := &SyncCmdMsg{}
	err = json.Unmarshal(msgBytes, msg)
	if err != nil {
		logger.Error("parse msg failed. err: %v", err)
		return nil, err
	}
	if msg.DstDir != "" {
		msg.DstDir = filepath.FromSlash(strings.ReplaceAll(msg.DstDir, "\\", "/"))
	}
	return msg, nil
}

func WriteForSyncMsg(conn *net.TCPConn, msg *SyncCmdMsg) error {
	// 序列化消息
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Error("marshal msg failed. err: %v", err)
		return err
	}
	// 写入消息长度
	err = binary.Write(conn, binary.BigEndian, uint32(len(msgBytes)))
	if err != nil {
		logger.Error("write msg len failed. err: %v", err)
		return err
	}
	// 写入消息内容
	_, err = conn.Write(msgBytes)
	if err != nil {
		logger.Error("write msg failed. err: %v", err)
		return err
	}
	return nil
}

func ReadForSyncRespMsg(conn *net.TCPConn) (*SyncRespMsg, error) {
	// 读取消息长度
	var msgLen uint32
	err := binary.Read(conn, binary.BigEndian, &msgLen)
	if err != nil {
		logger.Error("read msg len failed. err: %v", err)
		return nil, err
	}
	logger.Info("get msg len: %v", msgLen)
	// 读取消息内容
	msgBytes := make([]byte, msgLen)
	readLen := 0
	for readLen < int(msgLen) {
		n, err := conn.Read(msgBytes[readLen:])
		if err != nil {
			logger.Error("read msg failed. err: %v", err)
			return nil, err
		}
		// logger.Info("read msg len: %v", n)
		readLen += n
	}

	// 解析消息
	respMsg := &SyncRespMsg{}
	err = json.Unmarshal(msgBytes, respMsg)
	if err != nil {
		logger.Error("parse msg failed. err: %v", err)
		return nil, err
	}
	return respMsg, nil
}

func WriteForSyncRespMsg(conn *net.TCPConn, respMsg *SyncRespMsg) error {
	// 序列化消息
	msgBytes, err := json.Marshal(respMsg)
	if err != nil {
		logger.Error("marshal msg failed. err: %v", err)
		return err
	}
	logger.Info("write msg len: %v", len(msgBytes))
	// 写入消息长度
	err = binary.Write(conn, binary.BigEndian, uint32(len(msgBytes)))
	if err != nil {
		logger.Error("write msg len failed. err: %v", err)
		return err
	}
	// 写入消息内容
	n, err := conn.Write(msgBytes)
	if err != nil {
		logger.Error("write msg failed. err: %v", err)
		return err
	}
	logger.Info("writed msg len: %v", n)
	return nil
}
