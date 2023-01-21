# P2P-Blockchain
A basic implementation of a peer-to-peer blockchain network in Go, with functions to add transactions, mine blocks, and sync the chain among nodes.

## Features
- P2P network connection
- Blockchain data distribution
- Block mining
- Node syncing
- Register new nodes
- Validate the chain

## Getting Started
- Clone the repository
```
git clone https://github.com/yagcioglutoprak/p2p-blockchain.git
```
- Execute blockchain.go
```
go run blockchain.go
```

## Testing
You can test the network by running multiple instances of the program on different terminals.

- You can add new transactions to the network by calling the NewTransaction function with sender, recipient and amount as arguments
```
blockchain.NewTransaction("Alice", "Bob", 5)
```
- You can mine a new block by calling the mineBlock function
```
blockchain.mineBlock()
```
- You can retrieve the current state of the blockchain by calling the getChain function
```
blockchain.getChain()
```
- You can add a new node to the network by calling the RegisterNode function with the IP and Port of the new node as arguments
```
blockchain.RegisterNode("192.168.0.2", "8000")
```

## Python Testing
- You can add new nodes and get the blockchain using python
```
add_node("192.168.0.1", "8000")
blockchain = get_chain("192.168.0.1", "8000")
print(blockchain)
```
- You can mine a block using python by calling the mine_block function
```
mine_block("192.168.0.1", "8000")
```
Please note that in order to test the network, you will need to run multiple instances of the program on different terminals and add nodes to the network.

## Built With
- Go
- ChatGPT :)

## Acknowledgments
- This project was designed as a basic implementation of blockchain and p2p networks, so it may not be suitable for production use.
- If you want to continue building on this project, it would be best to research more on P2P networks and blockchain technology for better implementation.






