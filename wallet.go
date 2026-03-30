package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"log"
)

const version = byte(0x00)
const addressChecksumLen = 4

// 个人钱包
type Wallet struct {
	PrivateKey ecdsa.PrivateKey // 保密私钥
	PublicKey  []byte           // 钱包公钥
}

// 验证钱包地址
func ValidateAddress(address string) bool {
	// 版本字节+pubKey+校验和
	pubKeyHash, err := Base58Decode([]byte(address)) // 解密address得到pubKey
	if err != nil {
		return false
	}
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

// 创建钱包
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()                             // 创建新钱包
	address := fmt.Sprintf("%s", wallet.GetAddress()) // 获取钱包地址
	ws.Wallets[address] = wallet                      // 公钥放入到钱包集合
	return address
}

// 钱包结构体参数初始化
func NewWallet() *Wallet {
	privateKey, pubKey := newKeyPair()
	wallet := Wallet{privateKey, pubKey}

	return &wallet
}

// 椭圆曲线生成公私钥
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	return *privateKey, pubKey
}

// 获取公钥的对外地址
func (w Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
}
func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := sha256.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

// Serialize 将钱包序列化为字节数组（用于存储）
func (w *Wallet) Serialize() ([]byte, error) {
	// 私钥序列化为 PKCS#8 格式
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(&w.PrivateKey)
	if err != nil {
		return nil, err
	}
	// 公钥序列化为 PKIX 格式
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&w.PrivateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	// 将两个字节数组编码为简单结构
	type serializedWallet struct {
		PrivKey []byte
		PubKey  []byte
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(serializedWallet{privKeyBytes, pubKeyBytes}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DeserializeWallet 从字节数组恢复钱包
func DeserializeWallet(data []byte) (*Wallet, error) {
	type serializedWallet struct {
		PrivKey []byte
		PubKey  []byte
	}
	var sw serializedWallet
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&sw); err != nil {
		return nil, err
	}
	// 解析私钥
	privKeyInterface, err := x509.ParsePKCS8PrivateKey(sw.PrivKey)
	if err != nil {
		return nil, err
	}
	privKey, ok := privKeyInterface.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA private key")
	}
	// 公钥可选（可从私钥推导，但为了完整性可解析）
	pubKeyInterface, err := x509.ParsePKIXPublicKey(sw.PubKey)
	if err != nil {
		// 如果公钥解析失败，从私钥中提取
		pubKeyInterface = &privKey.PublicKey
	}
	pubKey, ok := pubKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		pubKey = &privKey.PublicKey
	}
	// 将公钥转为字节数组（与原来 NewWallet 中的格式一致）
	pubKeyBytes := append(pubKey.X.Bytes(), pubKey.Y.Bytes()...)
	return &Wallet{
		PrivateKey: *privKey,
		PublicKey:  pubKeyBytes,
	}, nil
}
