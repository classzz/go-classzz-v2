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

// enrEntry is the ENR entry which advertises `czz` protocol on the discovery.
type enrEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e enrEntry) ENRKey() string {
	return "czz"
}

// StartENRUpdater starts the `czz` ENR updater loop, which listens for chain
// head events and updates the requested node record whenever a fork is passed.
func StartENRUpdater(chain *core.BlockChain, ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := chain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(currentENREntry(chain))
			case <-sub.Err():
				// Would be nice to sync with Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

// currentENREntry constructs an `czz` ENR entry based on the current state of the chain.
func currentENREntry(chain *core.BlockChain) *enrEntry {
	return &enrEntry{
		ForkID: forkid.NewID(chain.Config(), chain.Genesis().Hash(), chain.CurrentHeader().Number.Uint64()),
	}
}
