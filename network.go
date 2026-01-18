package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
)

const protocol = "tcp"
const nodeVersion = 1                       //当前节点的版本号
const commandLength = 12                    //命令名称长度 - 模仿比特币协议消息类型标识符的固定长度
var nodeAddress string                      // 当前节点ip地址和端口
var miningAddress string                    //挖矿奖励地址 - 挖出新块的奖励发送到此地址
var knownNodes = []string{"localhost:3000"} //已知节点列表 - 启动时已知的其他节点地址
var blocksInTransit = [][]byte{}            //传输中的区块 - 正在从其他节点下载的区块哈希列表
var memPool = make(map[string]Trainscation) //内存池 - 存储尚未被打包进区块的交易

// 地址消息 - 用于交换节点节点间互相告知已知的其他节点地址
type addr struct {
	AddrList []string // 包含多个节点地址的列表
}

// 区块消息 - 用于传输完整的区块数据
type block struct {
	AddrFrom string // 发送此区块的节点地址
	Block    []byte // 序列化的区块数据
}

// 获取区块列表请求 - 请求区块哈希列表
type getBlocks struct {
	AddrFrom string // 请求方的节点地址
}

// 获取数据请求 - 请求具体的数据（区块或交易）
type getData struct {
	AddrFrom string // 请求方地址
	Type     string // 数据类型："block"或"tx"
	ID       []byte // 数据的哈希值
}

// 库存消息 - 通告本地拥有的数据
type inv struct {
	AddrFrom string   // 发送方地址
	Type     string   // 数据类型："block"或"tx"
	Items    [][]byte // 哈希值列表
}

// 交易消息 - 用于传播交易
type tx struct {
	AddFrom     string // 发送方地址（注意：字段名拼写错误，应为AddrFrom）
	Transaction []byte // 序列化的交易数据
}

// 版本消息 - 建立连接时的握手消息
type verzion struct { // 注意：类型名拼写错误，应为version
	Version    int    // 协议版本
	BestHeight uint64 // 本节点区块链的最新高度
	AddrFrom   string // 本节点地址
}

// 字符串转字节编码
func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

// 字节编码转字符串
func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

/*
	1、设置节点地址
	2、初始化区块链数据库
	3、创建监听器
	4、Accept启动监听，等待其它节点连接
	5、其它节点来了，连接已知节点同步区块链状态
	5、成为网络的种子节点，等待其他节点加入
*/
// 开启服务器（节点连接服务器）
func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID) // 设置节点地址localshot:3000
	miningAddress = minerAddress                      // 奖励矿工地址

	bc := GetBlockchain(nodeID) // 初始化区块链数据库

	ln, err := net.Listen(protocol, nodeAddress) // 监听传入连接localshot:3000
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close() // 执行完毕记得关闭监听

	if nodeAddress != knownNodes[0] { // 检查自己是不是那个种子节点
		sendVersion(knownNodes[0], bc) // 连接已知节点同步区块链状态
	}

	// 成为网络的种子节点，等待其他节点加入
	for {
		conn, err := ln.Accept() // 等待其它节点来连接
		if err != nil {
			log.Panic(err)
		}
		go handleConnection(conn, bc)
	}
}

// 新区块链连接已知节点（已知地址，新区块链）同步区块链状态
func sendVersion(addr string, bc *BlockChain) {
	bestHeight := bc.GetBestHeight()                                    // 新区块链的高度
	payload := gobEncode(verzion{nodeVersion, bestHeight, nodeAddress}) // 新编码（新版本，新区块，新节点地址 ）
	request := append(commandToBytes("version"), payload...)            // 带版本的新编码
	sendData(addr, request)                                             // 加入到连接（已知地址，带版本的新编码）
}

// 区块链 P2P 网络的消息处理器
func handleConnection(conn net.Conn, bc *BlockChain) {
	// 接受连接 → 读取数据 → 验证消息 → 解析命令 → 分发处理 → 响应/转发 → 清理资源
	request, err := io.ReadAll(conn) // 读取连接中的所有数据
	if err != nil {
		log.Panic(err)
	}
	command := bytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

// 发送数据，加入到连接（已知地址，带版本的新编码）
func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr) // 创建已知地址的网络连接
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string
		// 若报错无法连接，检查已知节点
		for _, node := range knownNodes {
			if node != addr {
				// 很可能其它人提前挖到矿了，我们要更新节点
				updatedNodes = append(updatedNodes, node)
			}
		}
		// 很可能其它人提前挖到矿了，我们要更新节点，新节点赋值给老节点
		knownNodes = updatedNodes

		return
	}
	defer conn.Close()
	// 加入连接，让监听器接收到
	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}
