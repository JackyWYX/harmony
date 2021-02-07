package p2ptypes

import (
	"fmt"
	"strings"

	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// AddrList is a list of multi address
type AddrList []ma.Multiaddr

// String is a function to print a string representation of the AddrList
func (al *AddrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

// Set is a function to set the value of AddrList based on a string
func (al *AddrList) Set(value string) error {
	if len(*al) > 0 {
		return fmt.Errorf("AddrList is already set")
	}
	for _, a := range strings.Split(value, ",") {
		addr, err := ma.NewMultiaddr(a)
		if err != nil {
			return err
		}
		*al = append(*al, addr)
	}
	return nil
}

// StringsToMultiAddrs convert a list of strings to a list of multiaddresses
func StringsToMultiAddrs(addrStrings []string) (maddrs []ma.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := ma.NewMultiaddr(addrString)
		if err != nil {
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

// StringsToP2PAddrs convert a slice of strings to a slice of libp2p_peer.AddrInfo
func StringsToP2PAddrs(addrStrings []string) ([]libp2p_peer.AddrInfo, error) {
	ais := make([]libp2p_peer.AddrInfo, 0, len(addrStrings))
	for _, addrString := range addrStrings {
		ma, err := ma.NewMultiaddr(addrString)
		if err != nil {
			return nil, err
		}
		ai, err := libp2p_peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return nil, err
		}
		ais = append(ais, *ai)
	}
	return ais, nil
}
