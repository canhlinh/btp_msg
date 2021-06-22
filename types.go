package main

import (
	"encoding/hex"
	"encoding/json"
	"strconv"

	"github.com/icon-project/goloop/server/jsonrpc"
)

type RelayMessage struct {
	BlockUpdates  [][]byte
	BlockProof    []byte
	ReceiptProofs [][]byte
	//
	height              int64
	numberOfBlockUpdate int
	eventSequence       int64
	numberOfEvent       int
}

type BlockWitness struct {
	Height  int64
	Witness [][]byte
}

type EventFilter struct {
	Addr      string    `json:"addr,omitempty"`
	Signature string    `json:"event"`
	Indexed   []*string `json:"indexed,omitempty"`
	Data      []*string `json:"data,omitempty"`
}

type BlockRequest struct {
	Height       jsonrpc.HexInt `json:"height"`
	EventFilters []*EventFilter `json:"eventFilters,omitempty"`
}

type WSResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

func NewHexBytes(b []byte) jsonrpc.HexBytes {
	return jsonrpc.HexBytes("0x" + hex.EncodeToString(b))
}

func NewHexInt(v int64) jsonrpc.HexInt {
	return jsonrpc.HexInt("0x" + strconv.FormatInt(v, 16))
}

type BlockEvent struct {
	Hash    jsonrpc.HexBytes     `json:"hash"`
	Height  jsonrpc.HexInt       `json:"height"`
	Indexes [][]jsonrpc.HexInt   `json:"indexes,omitempty"`
	Events  [][][]jsonrpc.HexInt `json:"events,omitempty"`
}

type ProofResultParam struct {
	BlockHash jsonrpc.HexBytes `json:"hash" validate:"required,t_hash"`
	Index     jsonrpc.HexInt   `json:"index" validate:"required,t_int"`
}

type Request struct {
	Version string          `json:"jsonrpc" validate:"required,version"`
	Method  string          `json:"method" validate:"required"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

type Response struct {
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type BlockUpdate struct {
	Height    int64
	BlockHash []byte
	Header    []byte
	Proof     []byte
}

type BlockUpdate2 struct {
	BlockHeader []byte
	Votes       []byte
	Validators  []byte
}

type BlockHeader struct {
	Version                int
	Height                 int64
	Timestamp              int64
	Proposer               []byte
	PrevID                 []byte
	VotesHash              []byte
	NextValidatorsHash     []byte
	PatchTransactionsHash  []byte
	NormalTransactionsHash []byte
	LogsBloom              []byte
	Result                 []byte
	serialized             []byte
}

type EventProof struct {
	Index int
	Proof []byte
}

type Event struct {
	Next     string
	Sequence int64
	Message  []byte
}

type ReceiptProof struct {
	Index       int
	Proof       []byte
	EventProofs []*EventProof
	Events      []*Event
}

type EventLog struct {
	Addr    []byte
	Indexed [][]byte
	Data    [][]byte
}

type BlockProof struct {
	Header       []byte
	BlockWitness *BlockWitness
}
