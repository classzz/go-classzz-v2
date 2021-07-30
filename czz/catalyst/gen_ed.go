// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package catalyst

import (
	"encoding/json"
	"errors"

	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/common/hexutil"
)

var _ = (*executableDataMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (e executableData) MarshalJSON() ([]byte, error) {
	type executableData struct {
		BlockHash    common.Hash     `json:"blockHash"     gencodec:"required"`
		ParentHash   common.Hash     `json:"parentHash"    gencodec:"required"`
		Miner        common.Address  `json:"miner"         gencodec:"required"`
		StateRoot    common.Hash     `json:"stateRoot"     gencodec:"required"`
		Number       hexutil.Uint64  `json:"number"        gencodec:"required"`
		GasLimit     hexutil.Uint64  `json:"gasLimit"      gencodec:"required"`
		GasUsed      hexutil.Uint64  `json:"gasUsed"       gencodec:"required"`
		Timestamp    hexutil.Uint64  `json:"timestamp"     gencodec:"required"`
		ReceiptRoot  common.Hash     `json:"receiptsRoot"  gencodec:"required"`
		LogsBloom    hexutil.Bytes   `json:"logsBloom"     gencodec:"required"`
		Transactions []hexutil.Bytes `json:"transactions"  gencodec:"required"`
	}
	var enc executableData
	enc.BlockHash = e.BlockHash
	enc.ParentHash = e.ParentHash
	enc.Miner = e.Miner
	enc.StateRoot = e.StateRoot
	enc.Number = hexutil.Uint64(e.Number)
	enc.GasLimit = hexutil.Uint64(e.GasLimit)
	enc.GasUsed = hexutil.Uint64(e.GasUsed)
	enc.Timestamp = hexutil.Uint64(e.Timestamp)
	enc.ReceiptRoot = e.ReceiptRoot
	enc.LogsBloom = e.LogsBloom
	if e.Transactions != nil {
		enc.Transactions = make([]hexutil.Bytes, len(e.Transactions))
		for k, v := range e.Transactions {
			enc.Transactions[k] = v
		}
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (e *executableData) UnmarshalJSON(input []byte) error {
	type executableData struct {
		BlockHash    *common.Hash    `json:"blockHash"     gencodec:"required"`
		ParentHash   *common.Hash    `json:"parentHash"    gencodec:"required"`
		Miner        *common.Address `json:"miner"         gencodec:"required"`
		StateRoot    *common.Hash    `json:"stateRoot"     gencodec:"required"`
		Number       *hexutil.Uint64 `json:"number"        gencodec:"required"`
		GasLimit     *hexutil.Uint64 `json:"gasLimit"      gencodec:"required"`
		GasUsed      *hexutil.Uint64 `json:"gasUsed"       gencodec:"required"`
		Timestamp    *hexutil.Uint64 `json:"timestamp"     gencodec:"required"`
		ReceiptRoot  *common.Hash    `json:"receiptsRoot"  gencodec:"required"`
		LogsBloom    *hexutil.Bytes  `json:"logsBloom"     gencodec:"required"`
		Transactions []hexutil.Bytes `json:"transactions"  gencodec:"required"`
	}
	var dec executableData
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.BlockHash == nil {
		return errors.New("missing required field 'blockHash' for executableData")
	}
	e.BlockHash = *dec.BlockHash
	if dec.ParentHash == nil {
		return errors.New("missing required field 'parentHash' for executableData")
	}
	e.ParentHash = *dec.ParentHash
	if dec.Miner == nil {
		return errors.New("missing required field 'miner' for executableData")
	}
	e.Miner = *dec.Miner
	if dec.StateRoot == nil {
		return errors.New("missing required field 'stateRoot' for executableData")
	}
	e.StateRoot = *dec.StateRoot
	if dec.Number == nil {
		return errors.New("missing required field 'number' for executableData")
	}
	e.Number = uint64(*dec.Number)
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for executableData")
	}
	e.GasLimit = uint64(*dec.GasLimit)
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for executableData")
	}
	e.GasUsed = uint64(*dec.GasUsed)
	if dec.Timestamp == nil {
		return errors.New("missing required field 'timestamp' for executableData")
	}
	e.Timestamp = uint64(*dec.Timestamp)
	if dec.ReceiptRoot == nil {
		return errors.New("missing required field 'receiptsRoot' for executableData")
	}
	e.ReceiptRoot = *dec.ReceiptRoot
	if dec.LogsBloom == nil {
		return errors.New("missing required field 'logsBloom' for executableData")
	}
	e.LogsBloom = *dec.LogsBloom
	if dec.Transactions == nil {
		return errors.New("missing required field 'transactions' for executableData")
	}
	e.Transactions = make([][]byte, len(dec.Transactions))
	for k, v := range dec.Transactions {
		e.Transactions[k] = v
	}
	return nil
}
