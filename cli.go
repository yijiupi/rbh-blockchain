package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// 命令模式
type CLI struct{}

func (cli *CLI) printUsage() {
	fmt.Println("使用方法：")
	fmt.Println(" createwallet -- 创建一个钱包并得到钱包的地址xxxxx")
	fmt.Println(" getbalance -address xxxxxx -- 查询钱包xxxxx的余额")
	fmt.Println(" listaddresses -- 获得所有钱包的地址")
	fmt.Println(" createblockchain -address xxxxx -- 初始化创建一个区块链并获取奖励到地址xxxxx")
	fmt.Println(" printchain -- 输出区块链上面所有的块")
	fmt.Println(" reindexutxo -- 重构utxo集合")
	fmt.Println(" send -from xxxxx1 -to xxxxx2 -amount 金额 -mine -- 转账交易从地址1转给地址2一定的金额")
	fmt.Println(" startnode -miner xxxxx -- 当前节点下启动挖矿")

}
func (cli *CLI) Run() {
	// 如果命令小于2条，给出命令提示
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
	// 检查节点设置
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID 未设置")
		os.Exit(1)
	}

	// 注册命令，命令架构
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)
	// 初始化命令，组装参数和默认赋值
	getBalanceAddress := getBalanceCmd.String("address", "", "这个地址是您钱包的地址")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "这个地址是您的钱包地址")
	sendFrom := sendCmd.String("from", "", "转账的发起人也是账户拥有者")
	sendTo := sendCmd.String("to", "", "转账的接收者")
	sendAmount := sendCmd.Int("amount", 0, "转账的金额")
	sendMine := sendCmd.Bool("mine", false, "设置mine节点将立即挖矿，未设置mine交易将广播到网络")
	startNodeMiner := startNodeCmd.String("miner", "", "启动一个节点，具有挖矿的功能")
	// 解析命令，os.Args[2:]替换参数默认值
	switch os.Args[1] {
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	/** start --- 执行命令（Parsed判断命令是否解析，若已解析便可执行，用解析后的加*执行*/
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" { //
			getBalanceCmd.Usage() // 在NewFlagSet时候已经初始化了defaultUsage，打印错误信息
		}
		cli.getbalance(*getBalanceAddress, nodeID) // 执行命令，用*解址自动执行，并传递到执行函数
	}
	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
		}
		cli.createblockchaincmd(*createBlockchainAddress, nodeID)
	}
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}

		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}
	if startNodeCmd.Parsed() {
		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			os.Exit(1)
		}
		cli.startNode(nodeID, *startNodeMiner)
	}
	/** end --- 执行命令结束**/

	// 执行（无需判定解析，直接执行）
	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}

}
