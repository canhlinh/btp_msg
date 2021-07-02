package main

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/icon-project/btp/cmd/btpsimple/module"
	"github.com/icon-project/btp/cmd/btpsimple/module/icon"
	"github.com/icon-project/btp/common"
	"github.com/icon-project/btp/common/codec"
	"github.com/icon-project/btp/common/crypto"
	"github.com/icon-project/btp/common/log"
	"github.com/icon-project/btp/common/mpt"
)

type Client struct {
	*icon.Client
}

func (c *Client) getBlockHeader(height icon.HexInt) (*icon.BlockHeader, []byte, error) {
	b, err := c.GetBlockHeaderByHeight(&icon.BlockHeightParam{
		Height: height,
	})
	if err != nil {
		return nil, nil, err
	}

	bh := &icon.BlockHeader{}
	if _, err = codec.RLP.UnmarshalFromBytes(b, bh); err != nil {
		return nil, nil, err
	}

	return bh, b, nil
}

func (c *Client) newBlockUpdate(v *icon.BlockNotification) (*module.BlockUpdate, error) {
	bh, b, err := c.getBlockHeader(v.Height)
	if err != nil {
		return nil, err
	}

	blkHash, _ := v.Hash.Value()
	if !bytes.Equal(blkHash, crypto.SHA3Sum256(b)) {
		return nil, fmt.Errorf("mismatch block hash with BlockNotification")
	}

	var update icon.BlockUpdate
	update.BlockHeader = b
	vb, vbErr := c.GetVotesByHeight(&icon.BlockHeightParam{Height: v.Height})
	if vbErr != nil {
		return nil, err
	}
	update.Votes = vb

	nvb, err := c.GetDataByHash(&icon.DataHashParam{
		Hash: icon.NewHexBytes(bh.NextValidatorsHash),
	})
	if err != nil {
		return nil, err
	}

	update.Validators = nvb

	bu := &module.BlockUpdate{
		BlockHash: blkHash,
		Height:    bh.Height,
		Header:    b,
	}
	bu.Proof, err = codec.RLP.MarshalToBytes(&update)
	if err != nil {
		return nil, err
	}

	return bu, nil
}

func (c *Client) newReceiptProofs(v *icon.BlockNotification) ([]*module.ReceiptProof, error) {
	nextEp := 0
	rps := make([]*module.ReceiptProof, 0)
	if len(v.Indexes) > 0 {
		l := v.Indexes[0]

		for i, index := range l {
			p := &icon.ProofEventsParam{BlockHash: v.Hash, Index: index, Events: v.Events[0][i]}

			proofs, err := c.GetProofForEvents(p)
			if err != nil {
				return nil, err
			}

			idx, _ := index.Value()
			rp := &module.ReceiptProof{
				Index:       int(idx),
				EventProofs: make([]*module.EventProof, 0),
			}
			if rp.Proof, err = codec.RLP.MarshalToBytes(proofs[0]); err != nil {
				return nil, err
			}
			for k := nextEp; k < len(p.Events); k++ {
				idxv, _ := p.Events[k].Value()

				ep := &module.EventProof{
					Index: int(idxv),
				}
				if ep.Proof, err = codec.RLP.MarshalToBytes(proofs[k+1]); err != nil {
					return nil, err
				}
				var evt *module.Event
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

func (c *Client) toEvent(proof [][]byte) (*module.Event, error) {
	el, err := toEventLog(proof)
	if err != nil {
		return nil, err
	}

	var i common.HexInt
	i.SetBytes(el.Indexed[EventIndexSequence])
	return &module.Event{
		Next:     module.BtpAddress(el.Indexed[EventIndexNext]),
		Sequence: i.Int64(),
		Message:  el.Data[0],
	}, nil
}

func toEventLog(proof [][]byte) (*icon.EventLog, error) {
	mp, err := mpt.NewMptProof(proof)
	if err != nil {
		return nil, err
	}
	el := &icon.EventLog{}
	if _, err := codec.RLP.UnmarshalFromBytes(mp.Leaf().Data, el); err != nil {
		return nil, fmt.Errorf("fail to parse EventLog on leaf err:%+v", err)
	}
	return el, nil
}

const (
	EventSignature      = "Message(str,int,bytes)"
	EventIndexSignature = 0
	EventIndexNext      = 1
	EventIndexSequence  = 2
)

var (
	GoloopURL         = "http://goloop.linhnc.info/api/v3/icondao"
	StartBlock  int64 = 10
	ContractSrc       = "cx7a0c2dd9751e592ac4fbd6c70bd5ec574ebf198a"
	BtpDst            = "btp://0x8.pra/0x5CC307268a1393AB9A764A20DACE848AB8275c46"
)

func main() {
	c := Client{Client: icon.NewClient(GoloopURL, log.New())}

	r := &icon.BlockRequest{
		Height: icon.NewHexInt(StartBlock),
		EventFilters: []*icon.EventFilter{
			{
				Addr:      icon.Address(ContractSrc),
				Signature: EventSignature,
				Indexed:   []*string{&BtpDst},
			},
		},
	}

	rm := icon.RelayMessage{
		BlockUpdates:  make([][]byte, 0),
		BlockProof:    make([]byte, 0),
		ReceiptProofs: make([][]byte, 0),
	}

	if err := c.MonitorBlock(r,
		func(conn *websocket.Conn, v *icon.BlockNotification) error {
			bu, err := c.newBlockUpdate(v)
			if err != nil {
				panic(err)
			}

			rps, err := c.newReceiptProofs(v)
			if err != nil {
				panic(err)
			}
			if len(rps) > 0 {
				rm.BlockUpdates = make([][]byte, 0)
			}

			rm.BlockUpdates = append(rm.BlockUpdates, bu.Proof)
			if len(rm.BlockUpdates) == 2 {
				c.CloseMonitor(conn)
			}

			log.Printf("bu: %v rps:%v \n", bu.Height, len(rps))
			return nil
		},
		func(conn *websocket.Conn) {
			log.Printf("ReceiveLoop connected %s \n", conn.LocalAddr().String())
		},
		func(conn *websocket.Conn, err error) {
			log.Printf("onError %s err:%+v n", conn.LocalAddr().String(), err)
		}); err != nil {
		panic(err)
	}

	b, err := codec.RLP.MarshalToBytes(rm)
	if err != nil {
		panic(err)
	}

	msg := base64.URLEncoding.EncodeToString(b)
	log.Println(msg)
}
