package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"log"
	"os"
)

const walletFile = "wallet_%s.dat" // 该节点下的钱包数据库

// 钱包集合（地址为key，存储钱包的指针）
type Wallets struct {
	Wallets map[string]*Wallet
}

// 创建一个新钱包集合（每个人可有多个钱包）
func NewWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}                       // 创建钱包集合对象
	wallets.Wallets = make(map[string]*Wallet) // 钱包集合对象的参数初始化
	err := wallets.LoadFromFile(nodeID)        // 加载钱包数据

	return &wallets, err
}

// 解码钱包内容
func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)          // 钱包文件
	if _, err := os.Stat(walletFile); os.IsNotExist(err) { // 检查钱包文件是否存在
		return err
	}

	fileContent, err := os.ReadFile(walletFile) // 获取钱包文件内容
	if err != nil {                             // 钱包不存在
		log.Panic(err)
	}

	var wallets Wallets                                     // 用于接收钱包内容
	gob.Register(elliptic.P256())                           // 注册椭圆曲线
	decoder := gob.NewDecoder(bytes.NewReader(fileContent)) // 创建gob解码器，解码文件内容
	err = decoder.Decode(&wallets)                          // 解码器指针wallets
	if err != nil {
		log.Panic(err)
	}
	ws.Wallets = wallets.Wallets // 将解码的钱包数据赋给当前对象
	return nil
}

// 保存钱包到文件
func (ws Wallets) SaveToFile(nodeID string) {
	var content bytes.Buffer                      // 缓冲区
	walletFile := fmt.Sprintf(walletFile, nodeID) // 钱包文件名称
	gob.Register(elliptic.P256())                 // 注册椭圆曲线gob中
	encoder := gob.NewEncoder(&content)           // 创建gob编码器，编码器指针content
	err := encoder.Encode(ws)                     // 编码器给ws加编码
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(walletFile, content.Bytes(), 0644) // 钱包集合写入磁盘，用户可读写其它人只读
	if err != nil {
		log.Panic(err)
	}
}
