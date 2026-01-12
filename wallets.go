package main

// 钱包集合（地址为key，存储钱包的指针）
type Wallets struct {
	Wallets map[string]*Wallet
}
