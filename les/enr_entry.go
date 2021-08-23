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

package les

import (
	"github.com/classzz/go-classzz-v2/core/forkid"
	"github.com/classzz/go-classzz-v2/p2p/dnsdisc"
	"github.com/classzz/go-classzz-v2/p2p/enode"
	"github.com/classzz/go-classzz-v2/rlp"
)

// lesEntry is the "les" ENR entry. This is set for LES servers only.
type lesEntry struct {
	// Ignore additional fields (for forward compatibility).
	VfxVersion uint
	Rest       []rlp.RawValue `rlp:"tail"`
}

func (lesEntry) ENRKey() string { return "les" }

// ethEntry is the "czz" ENR entry. This is redeclared here to avoid depending on package czz.
type ethEntry struct {
	ForkID forkid.ID
	Tail   []rlp.RawValue `rlp:"tail"`
}

func (ethEntry) ENRKey() string { return "czz" }

// setupDiscovery creates the node discovery source for the czz protocol.
func (czz *lightClasszz) setupDiscovery() (enode.Iterator, error) {
	it := enode.NewFairMix(0)

	// Enable DNS discovery.
	if len(czz.config.EthDiscoveryURLs) != 0 {
		client := dnsdisc.NewClient(dnsdisc.Config{})
		dns, err := client.NewIterator(czz.config.EthDiscoveryURLs...)
		if err != nil {
			return nil, err
		}
		it.AddSource(dns)
	}

	// Enable DHT.
	if czz.udpEnabled {
		it.AddSource(czz.p2pServer.DiscV5.RandomNodes())
	}

	forkFilter := forkid.NewFilter(czz.blockchain)
	iterator := enode.Filter(it, func(n *enode.Node) bool { return nodeIsServer(forkFilter, n) })
	return iterator, nil
}

// nodeIsServer checks whether n is an LES server node.
func nodeIsServer(forkFilter forkid.Filter, n *enode.Node) bool {
	var les lesEntry
	var czz ethEntry
	return n.Load(&les) == nil && n.Load(&czz) == nil && forkFilter(czz.ForkID) == nil
}
