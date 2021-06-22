package main

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"

	"github.com/icon-project/goloop/server/jsonrpc"
)

func HexBytesValue(hs jsonrpc.HexBytes) ([]byte, error) {
	if hs == "" {
		return nil, nil
	}
	raw := string(hs)
	raw = strings.TrimPrefix(raw, "0x")
	return hex.DecodeString(raw)
}

func dump(v interface{}) {
	json.NewEncoder(os.Stdout).Encode(v)
}
