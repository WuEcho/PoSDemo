package main

import (
	"sync"
	"crypto/sha256"
	"encoding/hex"
	"github.com/joho/godotenv"
	"log"
	"time"
	"github.com/davecgh/go-spew/spew"
	"os"
	"net"
	"io"
	"encoding/json"
	"bufio"
	"strconv"
	"fmt"
	"math/rand"
)

//创建区块
type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
	Validator string
}


//声明区块链数组
var Blockchain []Block
var tempBlocks []Block


//创建通道，确保线程间传参,candidateBlocks保存block
var candidateBlocks = make(chan Block)
//存放网络中广播的内容
var announcements = make(chan string)

//确保线程互斥
var mutex = &sync.Mutex{}

//创建map,存放节点的地址和tokens
var validators = make(map[string]int)



//创建blockhash值
func calculateHash(s string) string {
	h:=sha256.New()
	h.Write([]byte(s))
	hashed:= h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func calculateBlockHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) +
		block.PrevHash
	return calculateHash(record)
}


//验证区块的合法性
func isBlockValid(newBlock ,oldBlock Block)  bool {
	if oldBlock.Index +1 != newBlock.Index {
		return  false
	}
	if oldBlock.Hash!=newBlock.PrevHash {
		return false
	}
	if calculateBlockHash(newBlock) != newBlock.Hash {
		return false
	}
	return  true

}

func main()  {
	//读取.env文件内容
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	// 创建创世区块
	t := time.Now()
	genesisBlock := Block{}
	genesisBlock = Block{0, t.String(),
	0, calculateBlockHash(genesisBlock), "", ""}
	//格式化区块，确保在终端的输出
	spew.Dump(genesisBlock)
	//将创世区块存放到区块链
	Blockchain = append(Blockchain, genesisBlock)

	//读取.env文件内容
	httpPort := os.Getenv("PORT")

	// 监听网络端口
	server, err := net.Listen("tcp", ":"+httpPort)
	//推迟关闭
	defer server.Close()


	// 启动了一个Go routine 从 candidateBlocks 通道中获取提议的区块，然后填充到临时缓冲区 tempBlocks 中
	go func() {
		for candidate := range candidateBlocks {
			//添加线程所，确保线程互斥
			mutex.Lock()
			//当candidateblocks通道有数据时，就会添加临时缓冲区
			tempBlocks = append(tempBlocks, candidate)
			mutex.Unlock()
		}
	}()


	// 启动了一个Go routine 完成 pickWinner 函数
	go func() {
		for {
			//查找谁去挖矿
			pickWinner()
		}
	}()


	// 接收验证者节点的连接
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}


}

func handleConn(conn net.Conn) {
	defer conn.Close()

	go func() {
		for {
			msg := <-announcements
			io.WriteString(conn, msg)
		}
	}()
	// 验证者地址
	var address string

	// 验证者输入他所拥有的 tokens，tokens 的值越大，越容易获得新区块的记账权
	io.WriteString(conn, "Enter token balance:")
	scanBalance := bufio.NewScanner(conn)
	for scanBalance.Scan() {
		// 获取输入的数据，并将输入的值转为 int
		balance, err := strconv.Atoi(scanBalance.Text())
		if err != nil {
			log.Printf("%v not a number: %v", scanBalance.Text(), err)
			return
		}
		t := time.Now()
		// 生成验证者的地址
		address = calculateHash(t.String())
		// 将验证者的地址和token 存储到 validators
		validators[address] = balance
		fmt.Println(validators)
		break
	}

	io.WriteString(conn, "\nEnter a new BPM:")

	scanBPM := bufio.NewScanner(conn)

	go func() {
		for {
			// take in BPM from stdin and add it to blockchain after conducting necessary validation
			for scanBPM.Scan() {
				bpm, err := strconv.Atoi(scanBPM.Text())
				// 如果验证者试图提议一个被污染（例如伪造）的block，例如包含一个不是整数的BPM，那么程序会抛出一个错误，我们会立即从我们的验证器列表validators中删除该验证者，他们将不再有资格参与到新块的铸造过程同时丢失相应的抵押令牌。
				if err != nil {
					log.Printf("%v not a number: %v", scanBPM.Text(), err)
					delete(validators, address)
					conn.Close()
				}

				mutex.Lock()
				oldLastIndex := Blockchain[len(Blockchain)-1]
				mutex.Unlock()

				// 创建新的区块，然后将其发送到 candidateBlocks 通道
				newBlock, err := generateBlock(oldLastIndex, bpm, address)
				if err != nil {
					log.Println(err)
					continue
				}
				if isBlockValid(newBlock, oldLastIndex) {
					candidateBlocks <- newBlock
				}
				io.WriteString(conn, "\nEnter a new BPM:")
			}
		}
	}()

	// 循环会周期性的打印出最新的区块链信息
	for {
		time.Sleep(time.Minute)
		mutex.Lock()
		output, err := json.Marshal(Blockchain)
		mutex.Unlock()
		if err != nil {
			log.Fatal(err)
		}
		io.WriteString(conn, string(output)+"\n")
	}

}
// pickWinner creates a lottery pool of validators and chooses the validator who gets to forge a block to the blockchain
// by random selecting from the pool, weighted by amount of tokens staked
func pickWinner() {
	time.Sleep(30 * time.Second)
	mutex.Lock()
	temp := tempBlocks
	mutex.Unlock()

	lotteryPool := []string{}
	if len(temp) > 0 {

		// slightly modified traditional proof of stake algorithm
		// from all validators who submitted a block, weight them by the number of staked tokens
		// in traditional proof of stake, validators can participate without submitting a block to be forged
	OUTER:
		for _, block := range temp {
			// if already in lottery pool, skip
			for _, node := range lotteryPool {
				if block.Validator == node {
					continue OUTER
				}
			}

			// lock list of validators to prevent data race
			mutex.Lock()
			setValidators := validators
			mutex.Unlock()

			// 获取验证者的tokens
			k, ok := setValidators[block.Validator]
			if ok {
				// 向 lotteryPool 追加 k 条数据，k 代表的是当前验证者的tokens
				for i := 0; i < k; i++ {
					lotteryPool = append(lotteryPool, block.Validator)
				}
			}
		}

		// 通过随机获得获胜节点的地址

		s := rand.NewSource(time.Now().Unix())
		r := rand.New(s)
		lotteryWinner := lotteryPool[r.Intn(len(lotteryPool))]

		// 把获胜者的区块添加到整条区块链上，然后通知所有节点关于胜利者的消息
		for _, block := range temp {
			if block.Validator == lotteryWinner {
				mutex.Lock()
				Blockchain = append(Blockchain, block)
				mutex.Unlock()
				for _ = range validators {
					announcements <- "\nwinning validator: " + lotteryWinner + "\n"
				}
				break
			}
		}
	}

	mutex.Lock()
	tempBlocks = []Block{}
	mutex.Unlock()
}




func generateBlock(oldBlock Block, BPM int, address string) (Block, error) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateBlockHash(newBlock)
	newBlock.Validator = address

	return newBlock, nil
}
