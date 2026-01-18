package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
)

// 地址处理器，P2P网络中实现节点发现和网络扩散的核心机制
func handleAddr(request []byte) {
	var buff bytes.Buffer // 创建一个字节缓冲区，用于临时存储解码的数据
	var payload addr      // 用于存储解码后的地址数据

	buff.Write(request[commandLength:]) // 跳过命令头（前12字节），获取消息体,这里只处理数据部分，因为命令部分已经在handleConnection中处理过了
	dec := gob.NewDecoder(&buff)        // 创建一个gob解码器，连接到缓冲区
	err := dec.Decode(&payload)         // 将缓冲区中的二进制数据解码到payload变量中
	if err != nil {                     // 如果解码失败（如数据格式不对），程序会panic并终止
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...) // 将解码得到的地址列表添加到全局的knownNodes列表中
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	requestBlocks() // 向所有已知节点请求区块数据
}

// 这是在获得新节点地址后，主动开始同步区块链
func requestBlocks() {
	for _, address := range knownNodes {
		payload := gobEncode(getBlocks{nodeAddress})
		request := append(commandToBytes("getblocks"), payload...)

		sendData(address, request) // 函数将消息发送到指定地址
	}
}

// 这个结构体转换为二进制格式，便于网络传输
func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}
