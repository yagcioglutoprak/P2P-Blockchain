package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)
var syncLock = false
type Block struct {
    Index         int64
    Timestamp     int64
    Transactions  []Transaction
    Proof         int64
    Hash          string
    PreviousHash  string
    Miner         string
}



type Transaction struct {
    SenderPublicKey string
    Recipient       string
    Amount          int64
    Signature       []byte
}


var Accounts map[string]*ecdsa.PrivateKey

type Blockchain struct {
    Chain      []Block
    Nodes      map[string]bool
    TmpTxs     []Transaction
    NodeIP     string
    NodePort   string
    NodeCount  int
    Accounts   map[string]*ecdsa.PrivateKey
}




type Account struct {
    PublicKey  string
    PrivateKey string
}

func (blockchain *Blockchain) NewTransaction(senderPrivateKey *ecdsa.PrivateKey, recipient string, amount int64) int {
    // Get the public key from the private key
    pubKey := senderPrivateKey.Public().(*ecdsa.PublicKey)
    // Convert the public key to a string
    senderPublicKey := hex.EncodeToString(elliptic.Marshal(elliptic.P256(), pubKey.X, pubKey.Y))
    // Create the transaction
    transaction := Transaction{SenderPublicKey: senderPublicKey, Recipient: recipient, Amount: amount}
    // Sign the transaction with the private key
    hashedTransaction, _ := json.Marshal(transaction)
    r, s, _ := ecdsa.Sign(rand.Reader, senderPrivateKey, hashedTransaction)
    transaction.Signature = append(r.Bytes(), s.Bytes()...)
    // Add the transaction to the temporary transaction pool
    blockchain.TmpTxs = append(blockchain.TmpTxs, transaction)
    // Return the index of the next block that this transaction will be included in
    return int(blockchain.LastBlock().Index + 1)
}



func (blockchain *Blockchain) CreateAccount(ip string) {
    curve := elliptic.P256()
    priv, _, _, _ := elliptic.GenerateKey(curve, rand.Reader)
    privkey := new(ecdsa.PrivateKey)
privkey.Curve = elliptic.P256()
privkey.D = new(big.Int).SetBytes(priv)
privkey.PublicKey.X, privkey.PublicKey.Y = privkey.Curve.ScalarBaseMult(priv)
blockchain.Accounts[ip] = privkey

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

func generateKeys() *Account {
    curve := elliptic.P256()

    priv, _ := ecdsa.GenerateKey(curve, rand.Reader)
    pub := priv.PublicKey
    publicKey := hex.EncodeToString(elliptic.Marshal(elliptic.P256(), pub.X, pub.Y))
    privateKey := hex.EncodeToString(priv.D.Bytes())
    return &Account{PublicKey: publicKey, PrivateKey: privateKey}
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
        if syncLock {
            return
        }
        blockchain.Sync()
        blockchain.mineBlock()
        if syncLock {
            return
        }
        blockchain.Sync()
        conn.Write([]byte("Block mined"))
    case data == "getchain":
        if syncLock {
            return
        }
        blockchain.Sync()
	
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
    if syncLock {
        return
    }
    syncLock = true
    defer func() { syncLock = false }()

    for node := range blockchain.Nodes {
        fmt.Print("retrieving data from: "+node)
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
        buf := make([]byte, 2000000)
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
        //fmt.Print(receivedChain)
        if blockchain.validateChain(receivedChain) && len(receivedChain) > len(blockchain.Chain) {
            blockchain.Chain = receivedChain
        }
    }
}




func (blockchain *Blockchain) mineBlock() {
    lastBlock := blockchain.LastBlock()
    proof := blockchain.PoW(lastBlock)
    for publicKey := range blockchain.Accounts {
        fmt.Print(publicKey)
        }
        // Validate transactions before creating the block
        for _, transaction := range blockchain.TmpTxs {
            pubKey := blockchain.Accounts[transaction.SenderPublicKey].Public().(*ecdsa.PublicKey)
        r := new(big.Int).SetBytes(transaction.Signature[:32])
        s := new(big.Int).SetBytes(transaction.Signature[32:])
        hashedTransaction, _ := json.Marshal(transaction)
        fmt.Print(hashedTransaction)
        fmt.Print(pubKey)
        if !ecdsa.Verify(pubKey, hashedTransaction, r, s) {
            fmt.Print("mismatch")
            return 
        }
        
        }
        blockchain.NewBlock(proof, lastBlock.Hash)
        }




func (blockchain *Blockchain) PoW(lastBlock Block) int64 {
    var proof int64
    var hash string
    for publicKey := range blockchain.Accounts{
        hash = blockchain.HashBlock(Block{
        Index: lastBlock.Index + 1,
        Timestamp: time.Now().UnixNano(),
        Transactions: blockchain.TmpTxs,
        Proof: proof,
        PreviousHash: lastBlock.Hash,
        Miner: publicKey,
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
	
	lis, err := net.Listen("tcp", ":2005")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

		
	
    blockchain := Blockchain{Chain: []Block{}, Nodes: map[string]bool{}, Accounts: map[string]*ecdsa.PrivateKey{}}
    blockchain.loadNodesFromFile()

    priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    pub := priv.PublicKey
    pubkey := hex.EncodeToString(elliptic.Marshal(elliptic.P256(), pub.X, pub.Y))
    blockchain.Accounts[pubkey] = priv
    blockchain.CreateAccount("127.0.0.1")
    fmt.Println("Private Key:", hex.EncodeToString(priv.D.Bytes()))
    fmt.Println("Public Key:", hex.EncodeToString(elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)))
    blockchain.NewTransaction(priv,"recipient",1)


blockchain.CreateAccount("127.0.0.1")
fmt.Println("Private Key:", hex.EncodeToString(priv.D.Bytes()))
curve := elliptic.P256()
fmt.Println("Public Key:", hex.EncodeToString(elliptic.Marshal(curve, pub.X, pub.Y)))
blockchain.NewTransaction(priv,"recipient",1)

	
	
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
