// Copyright 2019 The go-classzz-v2 Authors
// This file is part of the go-classzz-v2 library.
//
// The go-classzz-v2 library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-classzz-v2 library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-classzz-v2 library. If not, see <http://www.gnu.org/licenses/>.

package czz

import (
	"github.com/classzz/go-classzz-v2/core"
	"github.com/classzz/go-classzz-v2/core/forkid"
	"github.com/classzz/go-classzz-v2/p2p/enode"
	"github.com/classzz/go-classzz-v2/rlp"
)

// ethEntry is the "czz" ENR entry which advertises czz protocol
// on the discovery network.
type ethEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e ethEntry) ENRKey() string {
	return "czz"
}

// startEthEntryUpdate starts the ENR updater loop.
func (czz *Classzz) startEthEntryUpdate(ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := czz.blockchain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(czz.currentEthEntry())
			case <-sub.Err():
				// Would be nice to sync with czz.Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

func (czz *Classzz) currentEthEntry() *ethEntry {
	return &ethEntry{ForkID: forkid.NewID(czz.blockchain.Config(), czz.blockchain.Genesis().Hash(),
		czz.blockchain.CurrentHeader().Number.Uint64())}
}
