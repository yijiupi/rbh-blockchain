package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

// 版本处理器-区块链节点间版本同步握手协议的核心实现
func handleVersion(request []byte, bc *BlockChain) {
	var buff bytes.Buffer // 缓冲区
	var payload verzion   // 版本消息结构体

	buff.Write(request[commandLength:]) // 请求消息写入缓冲区
	dec := gob.NewDecoder(&buff)        // gob 解码器（Go的二进制序列化格式）
	err := dec.Decode(&payload)         // 将缓冲区数据解码到 payload 结构体中
	if err != nil {
		log.Panic(err)
	}

	myBestHeight := bc.GetBestHeight()        // 获取本地区块链的最新高度
	foreignerBestHeight := payload.BestHeight // 获取对方区块链的高度

	if myBestHeight < foreignerBestHeight { // 比较高度对方最长合法链
		sendGetBlocks(payload.AddrFrom) // 向对方请求区块数据
	} else if myBestHeight > foreignerBestHeight { // 我方最长合法链
		sendVersion(payload.AddrFrom, bc) // 向对方发送我的版本（触发对方请求）
	}

	if !nodeIsKnown(payload.AddrFrom) { // 检查发送者是否已在已知节点列表中
		knownNodes = append(knownNodes, payload.AddrFrom) // 不在则添加
	}
}

// 组装版本信息，并在已知列表中连接种子节点
func sendVersion(addr string, bc *BlockChain) {
	bestHeight := bc.GetBestHeight()                                    // 新区块链的高度
	payload := gobEncode(verzion{nodeVersion, bestHeight, nodeAddress}) // 新编码（新版本，新区块，新节点地址 ）
	request := append(commandToBytes("version"), payload...)            // 带版本的新编码
	sendData(addr, request)                                             // 连接种子节点
}

func sendGetBlocks(address string) {
	payload := gobEncode(getBlocks{nodeAddress})               // 编码请求数据
	request := append(commandToBytes("getBlocks"), payload...) // 拼接命令+数据
	sendData(address, request)                                 // 发送请求
}
func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes { // 遍历已知节点列表
		if node == addr { // 找到匹配
			return true
		}
	}
	return false // 未找到匹配
}
