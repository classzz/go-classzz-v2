// Copyright 2015 The go-classzz-v2 Authors
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
	"math/big"
	"sync"
	"time"

	"github.com/classzz/go-classzz-v2/czz/protocols/czz"
	"github.com/classzz/go-classzz-v2/czz/protocols/snap"
)

// czzPeerInfo represents a short summary of the `czz` sub-protocol metadata known
// about a connected peer.
type czzPeerInfo struct {
	Version    uint     `json:"version"`    // Classzz protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // Hex hash of the peer's best owned block
}

// czzPeer is a wrapper around czz.Peer to maintain a few extra metadata.
type czzPeer struct {
	*czz.Peer
	snapExt *snapPeer // Satellite `snap` connection

	syncDrop *time.Timer   // Connection dropper if `czz` sync progress isn't validated in time
	snapWait chan struct{} // Notification channel for snap connections
	lock     sync.RWMutex  // Mutex protecting the internal fields
}

// info gathers and returns some `czz` protocol metadata known about a peer.
func (p *czzPeer) info() *czzPeerInfo {
	hash, td := p.Head()

	return &czzPeerInfo{
		Version:    p.Version(),
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

// snapPeerInfo represents a short summary of the `snap` sub-protocol metadata known
// about a connected peer.
type snapPeerInfo struct {
	Version uint `json:"version"` // Snapshot protocol version negotiated
}

// snapPeer is a wrapper around snap.Peer to maintain a few extra metadata.
type snapPeer struct {
	*snap.Peer
}

// info gathers and returns some `snap` protocol metadata known about a peer.
func (p *snapPeer) info() *snapPeerInfo {
	return &snapPeerInfo{
		Version: p.Version(),
	}
}
