package main

import (
	"fmt"
	"log"
	"strconv"
)

// 查询余额
func (cli *CLI) getbalance(address, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc, err := GetBlockchain(nodeID) // 获取节点的区块链
	if err != nil {
		log.Panic(err)
	}
	defer bc.db.Close()

	UTXOSet := UTXOSet{bc} // new一个UTXO

	pubKeyHash, err := Base58Decode([]byte(address)) // 根据地址获取公钥
	if err != nil {
		return
	}
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4] // [0x00] + [公钥哈希（20字节）] + [校验和（4字节）]
	UTXOs, err := UTXOSet.GetUTXO(pubKeyHash)      // 根据公钥获取UTXO得到余额
	if err != nil {
		log.Panic(err)
	}
	var balance uint64
	for _, out := range UTXOs {
		balance += out.Value
	}
	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

// 创建区块链
func (cli *CLI) createBlockchaincmd(address, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := CreateBlockchain(address, nodeID) // 创建区块链
	defer bc.db.Close()

	UTXOSet := UTXOSet{bc} // 区块链写入UTXO
	UTXOSet.Reindex()      // 重建UTXO（保留未花费的输出）

	fmt.Println("Done!")
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
	bc := NewBlockchain(nodeID) // 获取节点上最后一个区块的hash
	defer bc.db.Close()

	bci := bc.Iterator() // 使用程序遍历区块的内容

	for {
		block := bci.Next() // 得到区块

		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Height: %d\n", block.Height)
		fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)
		pow := NewProofOfWork(block)                                  // 组装工作量证明结构体
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate())) // 验证区块的工作量证明是否篡改
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

// 重新构建UTXO集合，并打印交易数量
func (cli *CLI) reindexUTXO(nodeID string) {
	bc := NewBlockchain(nodeID) // 获取当前节点的区块链
	UTXOSet := UTXOSet{bc}      // 组装utxoset结构体
	UTXOSet.Reindex()           // 重建UTXO（保留未花费的输出）
	if err := UTXOSet.Reindex(); err != nil {
		log.Panic(err)
	}
	count, err := UTXOSet.CountTransactions() // 交易数量
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

// 转账
func (cli *CLI) send(from, to string, amount uint64, nodeID string, mineNow bool) {
	if !ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}

	bc, err := GetBlockchain(nodeID) // 获取当前节点的区块链
	if err != nil {
		log.Panic(err)
	}
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()

	wallets, err := NewWallets(nodeID) // 获取钱包集合
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from) // 获取发送者钱包

	tx := NewUTXOTransaction(&wallet, to, amount, &UTXOSet) // 新建转账交易，并签名

	// 设置mine节点将立即挖矿，否则就是一笔普通转账交易
	if mineNow {
		cbTx := NewCoinbaseTX(from, "")    // 挖出新区块，奖励10块，挖矿成功的交易
		txs := []*Transaction{cbTx, tx}    // 挖出区块的交易和转账交易放一个数组
		newBlock, err := bc.MineBlock(txs) // 交易放入新区块，并存入数据库
		if err != nil {
			log.Panic(err)
		}
		if err := UTXOSet.Update(newBlock); err != nil { // 修改区块状态
			log.Panic(err)
		}
	} else {
		sendTx(knownNodes[0], tx) // 让其它节点一起挖矿，发送交易，发送给其它矿工去执行交易
	}

	fmt.Println("Success!")
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
