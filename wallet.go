package main

import "crypto/ecdsa"

// 个人钱包
type Wallet struct {
	ProvateKey ecdsa.PrivateKey // 保密私钥
	PublicKey  []byte           // 钱包公钥
}
