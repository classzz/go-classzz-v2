package types

import (
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/rlp"
	"io"
	"math/big"
)

type Pledge struct {
	Address         common.Address   `json:"address"`
	PubKey          []byte           `json:"pub_key"`
	ToAddress       common.Address   `json:"to_address"`
	Committee       bool             `json:"committee"`
	StakingAmount   *big.Int         `json:"staking_amount"`
	CoinBaseAddress []common.Address `json:"coinbase_address"`
}

type extPledge struct {
	Address         common.Address   `json:"address"`
	PubKey          []byte           `json:"pub_key"`
	ToAddress       common.Address   `json:"toAddress"`
	Committee       bool             `json:"committee"`
	StakingAmount   *big.Int         `json:"staking_amount"`
	CoinBaseAddress []common.Address `json:"coinbase_address"`
}

func (pi *Pledge) DecodeRLP(s *rlp.Stream) error {
	var epi extPledge
	if err := s.Decode(&epi); err != nil {
		return err
	}
	pi.Address, pi.PubKey, pi.ToAddress, pi.Committee, pi.StakingAmount, pi.CoinBaseAddress = epi.Address, epi.PubKey, epi.ToAddress, epi.Committee, epi.StakingAmount, epi.CoinBaseAddress
	return nil
}

func (pi *Pledge) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extPledge{
		Address:         pi.Address,
		PubKey:          pi.PubKey,
		ToAddress:       pi.ToAddress,
		Committee:       pi.Committee,
		StakingAmount:   pi.StakingAmount,
		CoinBaseAddress: pi.CoinBaseAddress,
	})
}

func (pi *Pledge) Clone() *Pledge {
	ss := &Pledge{
		Address:       pi.Address,
		PubKey:        CopyVotePk(pi.PubKey),
		ToAddress:     pi.ToAddress,
		Committee:     pi.Committee,
		StakingAmount: new(big.Int).Set(pi.StakingAmount),
	}
	for _, v := range pi.CoinBaseAddress {
		ss.CoinBaseAddress = append(ss.CoinBaseAddress, v)
	}
	return ss
}

type ConvertItem struct {
	ID          *big.Int         `json:"id"`
	AssetType   uint8            `json:"asset_type"`
	ConvertType uint8            `json:"convert_type"`
	TxHash      common.Hash      `json:"tx_hash"`
	PubKey      []byte           `json:"pub_key"`
	Amount      *big.Int         `json:"amount"` // czz asset amount
	FeeAmount   *big.Int         `json:"fee_amount"`
	Committee   common.Address   `json:"committee"`
	Path        []common.Address `json:"path"`
	RouterAddr  common.Address   `json:"router_addr"`
	Extra       []byte           `json:"extra"`
}

type extConvertItem struct {
	ID          *big.Int         `json:"id"`
	AssetType   uint8            `json:"asset_type"`
	ConvertType uint8            `json:"convert_type"`
	TxHash      common.Hash      `json:"tx_hash"`
	PubKey      []byte           `json:"pub_key"`
	Amount      *big.Int         `json:"amount"` // czz asset amount
	FeeAmount   *big.Int         `json:"fee_amount"`
	Committee   common.Address   `json:"committee"`
	Path        []common.Address `json:"path"`
	RouterAddr  common.Address   `json:"router_addr"`
	Extra       []byte           `json:"extra"`
}

func (ci *ConvertItem) DecodeRLP(s *rlp.Stream) error {
	var eci extConvertItem
	if err := s.Decode(&eci); err != nil {
		return err
	}
	ci.ID, ci.AssetType, ci.ConvertType, ci.TxHash, ci.PubKey, ci.Amount, ci.FeeAmount, ci.Committee, ci.Path, ci.RouterAddr, ci.Extra = eci.ID, eci.AssetType, eci.ConvertType, eci.TxHash, eci.PubKey, eci.Amount, eci.FeeAmount, eci.Committee, eci.Path, eci.RouterAddr, eci.Extra
	return nil
}

func (ci *ConvertItem) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extConvertItem{
		ID:          ci.ID,
		AssetType:   ci.AssetType,
		ConvertType: ci.ConvertType,
		TxHash:      ci.TxHash,
		PubKey:      ci.PubKey,
		Amount:      ci.Amount,
		FeeAmount:   ci.FeeAmount,
		Committee:   ci.Committee,
		Path:        ci.Path,
		RouterAddr:  ci.RouterAddr,
		Extra:       ci.Extra,
	})
}

func (ci *ConvertItem) Clone() *ConvertItem {
	ss := &ConvertItem{
		ID:          new(big.Int).Set(ci.ID),
		AssetType:   ci.AssetType,
		ConvertType: ci.ConvertType,
		TxHash:      ci.TxHash,
		PubKey:      CopyVotePk(ci.PubKey),
		Amount:      new(big.Int).Set(ci.Amount),
		FeeAmount:   new(big.Int).Set(ci.FeeAmount),
		Committee:   common.HexToAddress(ci.Committee.String()),
		RouterAddr:  common.HexToAddress(ci.RouterAddr.String()),
		Extra:       CopyVotePk(ci.Extra),
	}

	for _, v := range ci.Path {
		ss.Path = append(ss.Path, v)
	}
	return ss
}

func CopyVotePk(pk []byte) []byte {
	cc := make([]byte, len(pk))
	copy(cc, pk)
	return cc
}

type UsedItem struct {
	Atype  uint8
	TxHash common.Hash
}
