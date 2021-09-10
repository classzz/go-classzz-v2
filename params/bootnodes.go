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

package params

import "github.com/classzz/go-classzz-v2/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Classzz network.
var MainnetBootnodes = []string{
	// Classzz Foundation Go Bootnodes
	"enode://b8161aca1037cd18a051bd5bda61ec9c6c41b623f22f514ccda378b5778c56f5e1e37850f5d9e33c19ef90a502e1d4d7a72010a60723cfdca8c2e38d3cf598a5@8.147.87.155:32668",
	"enode://0fa2bff1b807381129fc2c4fdac1d79844b9f98c7e166262354aff61c16a4524160ef8f813aa3a5f15087557828b3413f474115d6bcc64cb90631c521aa54425@39.103.173.83:32668",
	"enode://2e682227fad19542d0754cda52b6cd8d840e953e1f33e69c632c1f03455a05a2f2b65eccb7624fac383243e310a7155b32ebcab13611a66eaa92bfd3053fe70e@185.239.69.240:32668",
	"enode://f2dc8b5f11fef4deba40ddc3f5ccfc04a2210acb9ebf0d748f4460cde6b83c8f57e97f6303dd375857ace7936c44edc47e0fa459f1faab7cd278f0b3f148ac5e@47.243.41.104:32668",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
var TestnetBootnodes = []string{
	"",
}

var V5Bootnodes = []string{}

const dnsPrefix = "enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@"

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/classzz/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	var net string
	switch genesis {
	case MainnetGenesisHash:
		net = "mainnet"
	case TestnetGenesisHash:
		net = "testnet"
	default:
		return ""
	}
	return dnsPrefix + protocol + "." + net + ".czzdisco.net"
}
