package main

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/icon-project/btp/common"
	"github.com/icon-project/btp/common/mpt"
	"github.com/icon-project/goloop/client"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/server/jsonrpc"
	v3 "github.com/icon-project/goloop/server/v3"
)

const (
	EventSignature      = "Message(str,int,bytes)"
	EventIndexSignature = 0
	EventIndexNext      = 1
	EventIndexSequence  = 2
)

var (
	GoloopURL         = "http://goloop.linhnc.info/api/v3/icondao"
	StartBlock  int64 = 9085
	BlockHash         = jsonrpc.HexBytes("0x22b733d9bb461247c3589d858ccd804ae5261b802d09cbf888fba97f05a8b9ca")
	ContractSrc       = "cx7a0c2dd9751e592ac4fbd6c70bd5ec574ebf198a"
	BtpDst            = "btp://0x8.pra/0x5CC307268a1393AB9A764A20DACE848AB8275c46"
)

type Client struct {
	*client.ClientV3
}

func (c *Client) getBlockHeader(height jsonrpc.HexInt) (*BlockHeader, error) {
	b, err := c.GetBlockHeaderByHeight(&v3.BlockHeightParam{
		Height: jsonrpc.HexInt(height),
	})
	if err != nil {
		return nil, err
	}

	bh := &BlockHeader{}
	if _, err = codec.RLP.UnmarshalFromBytes(b, bh); err != nil {
		return nil, err
	}
	bh.serialized = b

	return bh, nil
}

func (c *Client) newBlockUpdate(v BlockEvent) (*BlockUpdate, error) {
	bh, err := c.getBlockHeader(v.Height)
	if err != nil {
		return nil, err
	}

	blkHash, _ := HexBytesValue(v.Hash)
	if !bytes.Equal(blkHash, crypto.SHA3Sum256(bh.serialized)) {
		return nil, fmt.Errorf("mismatch block hash with BlockNotification")
	}

	var update BlockUpdate2
	update.BlockHeader = bh.serialized
	vb, vbErr := c.GetVotesByHeight(&v3.BlockHeightParam{Height: jsonrpc.HexInt(v.Height)})
	if vbErr != nil {
		return nil, err
	}
	update.Votes = vb

	nvb, err := c.GetDataByHash(&v3.DataHashParam{
		Hash: NewHexBytes(bh.NextValidatorsHash),
	})
	if err != nil {
		return nil, err
	}

	update.Validators = nvb

	bu := &BlockUpdate{
		BlockHash: blkHash,
		Height:    bh.Height,
		Header:    bh.serialized,
	}
	bu.Proof, err = codec.RLP.MarshalToBytes(&update)
	if err != nil {
		return nil, err
	}

	return bu, nil
}

func (c *Client) newReceiptProofs(v BlockEvent) ([]*ReceiptProof, error) {
	nextEp := 0
	rps := make([]*ReceiptProof, 0)
	if len(v.Indexes) > 0 {
		l := v.Indexes[0]

		for i, index := range l {
			p := &v3.ProofEventsParam{BlockHash: v.Hash, Index: index, Events: v.Events[0][i]}

			proofs, err := c.GetProofForEvents(p)
			if err != nil {
				return nil, err
			}

			idx := index.Value()
			rp := &ReceiptProof{
				Index:       int(idx),
				EventProofs: make([]*EventProof, 0),
			}
			if rp.Proof, err = codec.RLP.MarshalToBytes(proofs[0]); err != nil {
				return nil, err
			}
			for k := nextEp; k < len(p.Events); k++ {
				ep := &EventProof{
					Index: int(p.Events[k].Value()),
				}
				if ep.Proof, err = codec.RLP.MarshalToBytes(proofs[k+1]); err != nil {
					return nil, err
				}
				var evt *Event
				if evt, err = c.toEvent(proofs[k+1]); err != nil {
					return nil, err
				}
				rp.Events = append(rp.Events, evt)
				rp.EventProofs = append(rp.EventProofs, ep)
			}
			rps = append(rps, rp)
			nextEp = 0
		}
	}
	return rps, nil
}

func (c *Client) toEvent(proof [][]byte) (*Event, error) {
	el, err := toEventLog(proof)
	if err != nil {
		return nil, err
	}

	var i common.HexInt
	i.SetBytes(el.Indexed[EventIndexSequence])
	return &Event{
		Next:     string(el.Indexed[EventIndexNext]),
		Sequence: i.Int64(),
		Message:  el.Data[0],
	}, nil
}

func toEventLog(proof [][]byte) (*EventLog, error) {
	mp, err := mpt.NewMptProof(proof)
	if err != nil {
		return nil, err
	}
	el := &EventLog{}
	if _, err := codec.RLP.UnmarshalFromBytes(mp.Leaf().Data, el); err != nil {
		return nil, fmt.Errorf("fail to parse EventLog on leaf err:%+v", err)
	}
	return el, nil
}

func main() {
	c := Client{ClientV3: client.NewClientV3(GoloopURL)}

	b, err := c.GetBlockByHash(&v3.BlockHashParam{Hash: BlockHash})
	if err != nil {
		panic(err)
	}

	v := BlockEvent{
		Hash:    BlockHash,
		Height:  NewHexInt(b.Height),
		Indexes: [][]jsonrpc.HexInt{{NewHexInt(0)}},
		Events:  [][][]jsonrpc.HexInt{{[]jsonrpc.HexInt{NewHexInt(0)}}},
	}

	bu, err := c.newBlockUpdate(v)
	if err != nil {
		panic(err)
	}

	rps, err := c.newReceiptProofs(v)
	if err != nil {
		panic(err)
	}

	if len(rps) == 0 {
		panic("empty RPS")
	}

	bp, err := codec.RLP.MarshalToBytes(&BlockProof{
		Header: bu.Header,
		BlockWitness: &BlockWitness{
			Height: 0,
		},
	})
	if err != nil {
		panic(err)
	}

	rm := &RelayMessage{
		BlockUpdates:        [][]byte{bu.Proof},
		ReceiptProofs:       [][]byte{},
		BlockProof:          bp,
		height:              b.Height,
		numberOfBlockUpdate: 1,
		eventSequence:       0,
	}

	for _, rp := range rps {
		trp := &ReceiptProof{
			Index:       rp.Index,
			Proof:       rp.Proof,
			EventProofs: rp.EventProofs,
		}

		if b, err := codec.RLP.MarshalToBytes(trp); err != nil {
			panic(err)
		} else {
			rm.ReceiptProofs = append(rm.ReceiptProofs, b)
		}
	}

	data, err := codec.RLP.MarshalToBytes(rm)
	if err != nil {
		panic(err)
	}

	msg := base64.URLEncoding.EncodeToString(data)
	fmt.Println(msg)
}
