package main

import (
	"fmt"
	"log"
)

// 查询余额
func (cli *CLI) getbalance(address, nodeID string) {

}

// 创建区块链
func (cli *CLI) createblockchaincmd(address, nodeID string) {

}

// 创建钱包
func (cli *CLI) createWallet(nodeID string) {
	wallets, _ := NewWallets(nodeID)  // 创建/获取，钱包集合
	address := wallets.CreateWallet() // 创建新钱包
	wallets.SaveToFile(nodeID)        // 钱包保存到文件

	fmt.Printf("Your new address: %s\n", address)
}

// 获取所有钱包地址
func (cli *CLI) listAddresses(nodeID string) {
	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

// 打印输出区块链上所有块
func (cli *CLI) printChain(nodeID string) {

}

// 重构UTXO集合
func (cli *CLI) reindexUTXO(nodeID string) {

}

// 转账
func (cli *CLI) send(from, to string, amount int, nodeID string, mineNow bool) {

}

// 启动节点（节点，矿工地址）
func (cli *CLI) startNode(nodeID, minerAddress string) {
	fmt.Printf("Starting node %s\n", nodeID)
	if len(minerAddress) > 0 {
		if ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	// minerAddress为空普通节，否则为挖矿节点
	StartServer(nodeID, minerAddress)
}
