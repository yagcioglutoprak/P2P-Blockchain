package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Block struct {
	Index        int64
	Timestamp    int64
	Transactions []Transaction
	Proof        int64
	Hash         string
	PreviousHash string
}

type Transaction struct {
	Sender    string
	Recipient string
	Amount    int64
}

type Blockchain struct {
    Chain      []Block
    Nodes      map[string]bool
    TmpTxs     []Transaction
    NodeIP     string
    NodePort   string
    NodeCount  int
}


func (blockchain *Blockchain) NewTransaction(sender string, recipient string, amount int64) int {
	blockchain.TmpTxs = append(blockchain.TmpTxs, Transaction{Sender: sender, Recipient: recipient, Amount: amount})
	return int(blockchain.LastBlock().Index + 1)

}

func (blockchain *Blockchain) LastBlock() Block {
    if len(blockchain.Chain) == 0 {
        return Block{}
    }
    return blockchain.Chain[len(blockchain.Chain)-1]
}


func (blockchain *Blockchain) NewBlock(proof int64, previousHash string) Block {
	block := Block{
		Index:        int64(len(blockchain.Chain) + 1),
		Timestamp:    time.Now().UnixNano(),
		Transactions: blockchain.TmpTxs,
		Proof:        proof,
		PreviousHash: previousHash,
	}
	block.Hash = blockchain.HashBlock(block)
	blockchain.TmpTxs = []Transaction{}
	blockchain.Chain = append(blockchain.Chain, block)
	return block
}

func (blockchain *Blockchain) RegisterNode(ip, port string) {
    blockchain.Nodes[ip+":"+port] = true
	println(ip,port)
    blockchain.NodeCount++
    // Broadcast the new node to all the existing nodes
    for node := range blockchain.Nodes {
        if node != ip+":"+port {
            conn, err := net.Dial("tcp", node)
            if err != nil {
                continue
            }
            defer conn.Close()
            // Send the new node information
            conn.Write([]byte("register," + ip + "," + port))
        }
    }
}


func handleConnection(conn net.Conn, blockchain *Blockchain) {
    defer conn.Close()
    // Create a buffer to store the incoming data
    buf := make([]byte, 1024)
    // Read the incoming data into the buffer
    n, err := conn.Read(buf)
    if err != nil {
        return
    }
    // Convert the buffer to a string
    data := string(buf[:n])
    // Use a switch statement to handle different types of requests
    switch {
    case data == "mine":
        blockchain.Sync()
        blockchain.mineBlock()
        blockchain.Sync()
        conn.Write([]byte("Block mined"))
    case data == "getchain":
        blockchain.Sync()
		println(blockchain.Nodes)
        sendData, _ := json.Marshal(blockchain.Chain)
        conn.Write([]byte(string(sendData)))
    case data == "getnodecount":
        conn.Write([]byte(strconv.Itoa(blockchain.NodeCount)))
    case strings.HasPrefix(data, "register"):
        parts := strings.Split(data, ",")
        if len(parts) != 3 {
            conn.Write([]byte("Invalid register request"))
            return
        }
        ip, port := parts[1], parts[2]
        // Validate the new node information and register the node
        if blockchain.validateNode(ip, port){
            blockchain.RegisterNode(ip, port)
        } else {
            conn.Write([]byte("Invalid node information"))
        }
    }
}



func (blockchain *Blockchain) validateNode(ip, port string) bool {
    // Validate IP address
    if net.ParseIP(ip) == nil {
        return false
    }
    // Validate port number
    _, err := strconv.Atoi(port)
    if err != nil {
        return false
    }
    return true
}



func (blockchain *Blockchain) HashBlock(block Block) string {
	data, _ := json.Marshal(block)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
func (blockchain *Blockchain) Broadcast(block Block) {
    for node := range blockchain.Nodes {
        conn, err := net.Dial("tcp", node)
        if err != nil {
            continue
        }
        defer conn.Close()
        sendData, _ := json.Marshal(block)
        conn.Write([]byte(string(sendData)))
    }
}

func (blockchain *Blockchain) validateChain(receivedChain []Block) bool {
    //check for valid chain
    if len(receivedChain) > len(blockchain.Chain) {
        return true
    }
    return false
}

func (blockchain *Blockchain) Sync() {
    for node := range blockchain.Nodes {
        _, err := net.DialTimeout("tcp", node, time.Second*5)
        if err != nil {
            log.Printf("Failed to connect to node %s: %v", node, err)
            delete(blockchain.Nodes, node)
            blockchain.NodeCount--
            continue
        }
        conn, err := net.Dial("tcp", node)
        if err != nil {
            log.Printf("Failed to connect to node %s: %v", node, err)
            delete(blockchain.Nodes, node)
            blockchain.NodeCount--
            continue
        }
        defer conn.Close()
        conn.Write([]byte("getchain"))
        buf := make([]byte, 4096)
        n, err := conn.Read(buf)
        if err != nil {
            log.Printf("Failed to get chain from node %s: %v", node, err)
            delete(blockchain.Nodes, node)
            blockchain.NodeCount--
            continue
        }
        receivedData := string(buf[:n])
        var receivedChain []Block
        json.Unmarshal([]byte(receivedData), &receivedChain)
        if blockchain.validateChain(receivedChain) && len(receivedChain) > len(blockchain.Chain) {
            blockchain.Chain = receivedChain
        }
    }
}




func (blockchain *Blockchain) mineBlock() {
    lastBlock := blockchain.LastBlock()
    proof := blockchain.PoW(lastBlock)
    blockchain.NewTransaction("network", blockchain.NodeIP, 1)
    blockchain.NewBlock(proof, lastBlock.Hash)

}


func (blockchain *Blockchain) PoW(lastBlock Block) int64 {
	var proof int64
	var hash string
	for {
		hash = blockchain.HashBlock(Block{
			Index:        lastBlock.Index + 1,
			Timestamp:    time.Now().UnixNano(),
			Transactions: blockchain.TmpTxs,
			Proof:        proof,
			PreviousHash: lastBlock.Hash,
		})
		if hash[:4] == "0000" {
			break
		}
		proof++
	}
	return proof
}
func (blockchain *Blockchain) loadNodesFromFile() {
    file, err := os.Open("nodes.txt")
    if err != nil {
        log.Fatalf("Failed to open nodes.txt: %v", err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        node := scanner.Text()
        parts := strings.Split(node, ":")
        if len(parts) != 2 {
            log.Printf("Invalid node information in nodes.txt: %s", node)
            continue
        }
        ip, port := parts[0], parts[1]
        if blockchain.validateNode(ip, port) {
            fmt.Println(ip)
            fmt.Println(port)
            blockchain.Nodes[ip+":"+port] = true
            blockchain.NodeCount++
        } else {
            log.Printf("Invalid node information in nodes.txt: %s", node)
        }
    }

    if err := scanner.Err(); err != nil {
        log.Fatalf("Failed to read nodes.txt: %v", err)
    }
}


func main() {
	
	lis, err := net.Listen("tcp", ":2006")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

		
	
	blockchain := Blockchain{Chain: []Block{}, Nodes: map[string]bool{}}
    blockchain.loadNodesFromFile()
	blockchain.NodeIP, blockchain.NodePort, _ = net.SplitHostPort(lis.Addr().String())

	
	
	//blockchain.RegisterNode("127.0.0.1","2007")
	//blockchain.NewTransaction("A", "B", 1)
	//lastBlock := blockchain.LastBlock()
	//proof := blockchain.PoW(lastBlock)
	//blockchain.NewBlock(proof, "")

	//println(blockchain.Chain[0].Hash)

	for {
		conn, err := lis.Accept()
		if err != nil {
		log.Fatalf("Failed to accept connection: %v", err)
		}
		go handleConnection(conn, &blockchain)
		}
}
