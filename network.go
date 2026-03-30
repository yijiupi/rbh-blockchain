package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const protocol = "tcp"
const nodeVersion = 1                       //当前节点的版本号
const commandLength = 12                    //命令名称长度 - 模仿比特币协议消息类型标识符的固定长度
var nodeAddress string                      // 当前节点ip地址和端口
var miningAddress string                    //挖矿奖励地址 - 挖出新块的奖励发送到此地址
var knownNodes = []string{"localhost:3000"} //已知节点列表 - 启动时已知的其他节点地址
var blocksInTransit = [][]byte{}            //传输中的区块 - 正在从其他节点下载的区块哈希列表
var memPool = make(map[string]Transaction)  //内存池 - 存储尚未被打包进区块的交易
var (
	// 并发安全锁（互斥锁）
	knownNodesMutex      sync.RWMutex
	memPoolMutex         sync.RWMutex
	blocksInTransitMutex sync.RWMutex
	// 已广播交易的去重集合
	broadcastedTxs      = make(map[string]bool)
	broadcastedTxsMutex sync.RWMutex
)
var (
	// 待下载区块队列（避免重复）
	blocksToDownload  = make(map[string]bool)      // key = 区块哈希字符串
	downloadingBlocks = make(map[string]time.Time) // 正在下载的区块及请求时间
	downloadRetries   = make(map[string]int)       // 重试次数
	downloadMutex     sync.RWMutex
	downloaderStarted bool
)

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
	1、当前节点地址
	2、创建监听器
	3、判定当前节点是否为种子节点，若不是种子节点，创建当前节点数据库，并在已知列表中连接种子节点
	4、Accept启动监听，等待其它节点连接
	5、其它节点来了，连接已知节点同步区块链状态
	5、成为网络的种子节点，等待其他节点加入
*/
// 开启服务器（节点连接服务器）
func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID) // 当前服务器节点
	miningAddress = minerAddress                      // 奖励矿工地址

	// 1. 初始化区块链（若不存在则创建空链）
	bc, err := initBlockchain(nodeID)
	if err != nil {
		log.Fatalf("Failed to initialize blockchain: %v", err)
	}
	defer func() {
		if bc.db != nil {
			bc.db.Close()
		}
	}()

	// 2. 启动下载管理器（仅一次，且仅在数据库有效时）// 使用 atomic bool 或普通 bool + 互斥锁确保只启动一次
	if !downloaderStarted && bc.db != nil {
		go downloadManager() // 启动后台 goroutine，负责管理区块的并发下载、超时重试等
		downloaderStarted = true
		log.Println("Block download manager started")
	}

	// 3. 启动网络监听
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	// 4. 如果不是种子节点，主动向种子节点发送版本信息
	if nodeAddress != knownNodes[0] {
		knownNodesMutex.RLock()
		seed := knownNodes[0] // 并发安全锁，安全读取种子节点
		knownNodesMutex.RUnlock()
		sendVersion(seed, bc) // 自己不是种子节点，连接已知种子节点同步区块链状态
	}

	// 5. 接受连接
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleConnection(conn, bc) // 监测到其它节点连接，处理连接同步数据到db
	}
}

// 发送数据，连接种子节点（已知地址，带版本的新编码）
func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr) // 客户端连接已知列表中的种子节点
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string
		// 并非安全锁
		knownNodesMutex.Lock()
		// 若报错说明addr是个失败节点，需要删除
		for _, node := range knownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node) // 排除掉失败节点的新数组
			}
		}
		knownNodes = updatedNodes // 排除掉失败节点的新数组，赋值给已知节点列表
		// 并非安全锁
		knownNodesMutex.Unlock()
		return
	}
	defer conn.Close()
	// data 是要发送的字节切片（比如序列化的结构体、文件内容等）
	// conn 是网络连接（实现了 io.Writer 接口）
	_, err = io.Copy(conn, bytes.NewReader(data)) // 它将内存中的数据通过 TCP 连接发送出去，让服务端接收到
	if err != nil {
		log.Printf("Failed to send data to %s: %v", addr, err)
		// 不 panic，仅记录日志，发送失败由调用方决定是否重试（当前忽略）
		return
	}
}

// 执行交易
func sendTx(addr string, tnx *Transaction) {
	data := tx{nodeAddress, tnx.Serialize()} // 准备我的数据，发送给其它节点执行
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	sendData(addr, request)
}

// sendGetData 发送 getdata 消息，请求特定数据（区块或交易）
func sendGetData(addr, kind string, id []byte) {
	payload := getData{
		AddrFrom: nodeAddress,
		Type:     kind,
		ID:       id,
	}
	data := gobEncode(payload)
	request := append(commandToBytes("getdata"), data...)
	sendData(addr, request)
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
		handleVersion(request, bc) // 版本处理，用于比较区块高度最长合法链
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}
