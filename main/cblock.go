package main

import (
	"strconv"
	"bytes"
	"crypto/sha256"
	"time"
	"encoding/gob"
	"github.com/boltdb/bolt"
	"log"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"

type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.Data, timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), []byte(data),
		prevBlockHash, []byte{}, 0}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Hash = hash[:]
	block.Nonce = nonce
	//block.SetHash()
	return block
}

type Blockchain struct {
	tip []byte
	db  *bolt.DB
}

func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))
		return nil
	})
	if err!=nil{
		log.Panic(err)
	}
	//prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := NewBlock(data, lastHash)

	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err!=nil{
			log.Panic(err)
		}
		err = b.Put([]byte("l"), newBlock.Hash)
		if err!=nil{
			log.Panic(err)
		}
		bc.tip = newBlock.Hash
		return nil
	})
	//bc.blocks = append(bc.blocks, newBlock)
}

func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

/**
1. Open a DB file.
2. Check if there’s a blockchain stored in it.
3. If there’s a blockchain:
   3.1 Create a new Blockchain instance.
   3.2 Set the tip of the Blockchain instance to the last block hash stored in the DB.
4. If there’s no existing blockchain:
   4.1 Create the genesis block.
   4.2 Store in the DB.
   4.3 Save the genesis block’s hash as the last block hash.
   4.4 Create a new Blockchain instance with its tip pointing at the genesis block.
*/
func NewBlockChain() *Blockchain {
	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err!=nil{
		log.Panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			genesis := NewGenesisBlock()
			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err!=nil{
				log.Panic(err)
			}
			err = b.Put(genesis.Hash, genesis.Serialize())
			if err!= nil {
				log.Panic(err)
			}
			err = b.Put([]byte("l"), genesis.Hash)
			if err!=nil{
				log.Panic(err)
			}
			tip = genesis.Hash
		} else {
			tip = b.Get([]byte("l"))
		}
		return nil
	})

	if err!=nil{
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	encoder.Encode(b)
	return result.Bytes()
}

func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	decoder.Decode(&block)
	return &block
}

type BlockchainInterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (bc *Blockchain)Iterator() *BlockchainInterator{
	bci := &BlockchainInterator{bc.tip, bc.db}
	return bci
}

func (i *BlockchainInterator) Next() *Block {
	var block *Block
	err := i.db.View(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)
		return nil
	})
	if err!=nil{
		log.Panic(err)
	}
	i.currentHash = block.PrevBlockHash
	return block
}