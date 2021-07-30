package vm

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/classzz/go-classzz-v2/common"
	"github.com/classzz/go-classzz-v2/core/types"
	"github.com/classzz/go-classzz-v2/crypto"
	"github.com/classzz/go-classzz-v2/log"
	"github.com/classzz/go-classzz-v2/rlp"
	lru "github.com/hashicorp/golang-lru"
	"io"
	"math/big"
	"math/rand"
)

var IC *PledgeCache

func init() {
	IC = newPledgeCache()
}

type PledgeCache struct {
	Cache *lru.Cache
	size  int
}

func newPledgeCache() *PledgeCache {
	cc := &PledgeCache{
		size: 20,
	}
	cc.Cache, _ = lru.New(cc.size)
	return cc
}

type TeWakaImpl struct {
	PledgeInfos  []*types.Pledge
	ConvertItems []*types.ConvertItem
}

func NewTeWakaImpl() *TeWakaImpl {
	return &TeWakaImpl{
		PledgeInfos:  make([]*types.Pledge, 0, 0),
		ConvertItems: make([]*types.ConvertItem, 0, 0),
	}
}

func CloneTeWakaImpl(ori *TeWakaImpl) *TeWakaImpl {
	if ori == nil {
		return nil
	}
	tmp := &TeWakaImpl{}

	items := make([]*types.Pledge, 0, 0)
	for _, val := range ori.PledgeInfos {
		vv := val.Clone()
		items = append(items, vv)
	}
	tmp.PledgeInfos = items

	items1 := make([]*types.ConvertItem, 0, 0)
	for _, val := range ori.ConvertItems {
		vv := val.Clone()
		items1 = append(items1, vv)
	}
	tmp.ConvertItems = items1
	return tmp
}

type extTeWakaImpl struct {
	PledgeInfos  []*types.Pledge
	ConvertItems []*types.ConvertItem
}

func (twi *TeWakaImpl) DecodeRLP(s *rlp.Stream) error {
	var etwi extTeWakaImpl
	if err := s.Decode(&etwi); err != nil {
		return err
	}
	twi.PledgeInfos, twi.ConvertItems = etwi.PledgeInfos, etwi.ConvertItems
	return nil
}

func (twi *TeWakaImpl) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extTeWakaImpl{
		PledgeInfos:  twi.PledgeInfos,
		ConvertItems: twi.ConvertItems,
	})
}

func ValidPk(pk []byte) error {
	_, err := crypto.UnmarshalPubkey(pk)
	return err
}

func (twi *TeWakaImpl) Save(state StateDB, preAddress common.Address) error {
	key := common.BytesToHash(preAddress[:])
	data, err := rlp.EncodeToBytes(twi)

	if err != nil {
		log.Crit("Failed to RLP encode ImpawnImpl", "err", err)
	}
	hash := common.RlpHash(data)
	state.SetTeWakaState(preAddress, key, data)
	tmp := CloneTeWakaImpl(twi)
	if tmp != nil {
		IC.Cache.Add(hash, tmp)
	}
	return err
}

func (i *TeWakaImpl) Load(state StateDB, preAddress common.Address) error {
	key := common.BytesToHash(preAddress[:])
	data := state.GetTeWakaState(preAddress, key)
	lenght := len(data)
	if lenght == 0 {
		return errors.New("Load data = 0")
	}
	// cache := true
	hash := common.RlpHash(data)
	var temp TeWakaImpl
	if cc, ok := IC.Cache.Get(hash); ok {
		impawn := cc.(*TeWakaImpl)
		temp = *(CloneTeWakaImpl(impawn))
	} else {
		if err := rlp.DecodeBytes(data, &temp); err != nil {
			log.Error("Invalid ImpawnImpl entry RLP", "err", err)
			return errors.New(fmt.Sprintf("Invalid ImpawnImpl entry RLP %s", err.Error()))
		}
		tmp := CloneTeWakaImpl(&temp)
		if tmp != nil {
			IC.Cache.Add(hash, tmp)
		}
		// cache = false
	}
	i.PledgeInfos, i.ConvertItems = temp.PledgeInfos, temp.ConvertItems
	return nil
}

func (twi *TeWakaImpl) Mortgage(address common.Address, to common.Address, pubKey []byte, amount *big.Int, cba []common.Address) {

	info := &types.Pledge{
		Address:         address,
		PubKey:          pubKey,
		ToAddress:       to,
		StakingAmount:   new(big.Int).Set(amount),
		CoinBaseAddress: cba,
	}

	twi.PledgeInfos = append(twi.PledgeInfos, info)
}

func (twi *TeWakaImpl) Update(address common.Address, cba []common.Address) {

	for _, v := range twi.PledgeInfos {
		if bytes.Equal(v.Address[:], address[:]) {
			v.CoinBaseAddress = cba
			return
		}
	}
}

func (twi *TeWakaImpl) Convert(item *types.ConvertItem) {
	twi.ConvertItems = append(twi.ConvertItems, item)
}

func (twi *TeWakaImpl) Confirm(item *types.ConvertItem) {
	for i, v := range twi.ConvertItems {
		if v.ID.Cmp(item.ID) == 0 {
			twi.ConvertItems = append(twi.ConvertItems[:i], twi.ConvertItems[i+1:]...)
			return
		}
	}
}

func (twi *TeWakaImpl) GetStakingByUser(address common.Address) *big.Int {

	sumAmount := big.NewInt(0)
	for _, v := range twi.PledgeInfos {
		for _, v1 := range v.CoinBaseAddress {
			if bytes.Equal(address.Bytes(), v1.Bytes()) {
				sumAmount = new(big.Int).Add(sumAmount, v.StakingAmount)
				break
			}
		}
	}

	return sumAmount
}

func (twi *TeWakaImpl) GetCommittee() common.Address {
	return twi.PledgeInfos[rand.Intn(len(twi.PledgeInfos))].Address
}
