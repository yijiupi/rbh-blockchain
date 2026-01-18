package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
)

const version = byte(0x00)
const addressChecksumLen = 4

// 个人钱包
type Wallet struct {
	ProvateKey ecdsa.PrivateKey // 保密私钥
	PublicKey  []byte           // 钱包公钥
}

// 验证钱包地址
func ValidateAddress(address string) bool {
	// 版本字节+pubKey+校验和
	pubKeyHash := Base58Decode([]byte(address))                       // 解密address得到pubKey
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:] // 校验和：获取切片最后四位
	version := pubKeyHash[0]                                          // 版本字节：获取切片第一位
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]   // 公钥：获取切片中间部分（上面的剩余部分）
	versionPubKey := append([]byte{version}, pubKeyHash...)           // 版本字节+公钥展开操作符
	targetChecksum := checksum(versionPubKey)                         // 版本+公钥计算得到新校验和
	return bytes.Equal(actualChecksum, targetChecksum)                // 比较原校验和和新校验和比较是否一致
}

// 版本和公钥加密计算取前四位得到新校验和
func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)      // 一次加密
	secondSHA := sha256.Sum256(firstSHA[:]) // 二次加密
	return secondSHA[:addressChecksumLen]   // 二次加密结果取前四位
}
