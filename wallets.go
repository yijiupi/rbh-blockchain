package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
)

const walletFile = "wallet_%s.dat" // 该节点下的钱包数据库

// 钱包集合（地址为key，存储钱包的指针）
type Wallets struct {
	Wallets map[string]*Wallet
}

// 获取/新建，钱包集合（有则返回无则返回空集合）
func NewWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}                       // 创建空钱包集合对象
	wallets.Wallets = make(map[string]*Wallet) // 钱包集合对象的参数初始化为空
	err := wallets.LoadFromFile(nodeID)        // 加载钱包数据

	return &wallets, err
}

// LoadFromFile 从磁盘文件加载钱包集合
func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID) // 例如 "wallet_3000.dat"
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err // 文件不存在，返回错误
	}

	fileContent, err := os.ReadFile(walletFile)
	if err != nil {
		return fmt.Errorf("read wallet file: %w", err)
	}

	// 反序列化为 map[string][]byte
	var serializedMap map[string][]byte
	// gob.Register(elliptic.P256()) // 已弃用：Go 1.23+ 无法序列化曲线类型，改用 x509 序列化
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	if err := decoder.Decode(&serializedMap); err != nil {
		return fmt.Errorf("decode wallet data: %w", err)
	}

	// 初始化钱包映射
	if ws.Wallets == nil {
		ws.Wallets = make(map[string]*Wallet)
	}

	// 遍历每个钱包的序列化数据，逐一恢复
	for address, walletData := range serializedMap {
		wallet, err := DeserializeWallet(walletData) // 使用 x509 反序列化函数
		if err != nil {
			return fmt.Errorf("deserialize wallet for address %s: %w", address, err)
		}
		ws.Wallets[address] = wallet
	}

	return nil
}

// 保存钱包到文件
func (ws Wallets) SaveToFile(nodeID string) error {
	serializedMap := make(map[string][]byte)
	for addr, wallet := range ws.Wallets {
		data, err := wallet.Serialize()
		if err != nil {
			return fmt.Errorf("serialize wallet %s: %w", addr, err)
		}
		serializedMap[addr] = data
	}
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeID)
	// gob.Register(elliptic.P256()) // 已弃用：Go 1.23+ 无法序列化曲线类型，改用 x509 序列化
	encoder := gob.NewEncoder(&content)
	if err := encoder.Encode(serializedMap); err != nil {
		return fmt.Errorf("encode wallets: %w", err)
	}
	if err := os.WriteFile(walletFile, content.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// 获取钱包集合中左右钱包地址
func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// 获取钱包集合中具体的钱包
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}
