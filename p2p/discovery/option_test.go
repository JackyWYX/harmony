package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	libp2p_peer "github.com/libp2p/go-libp2p-core/peer"

	p2putils "github.com/harmony-one/harmony/p2p/utils"
)

var (
	tmpDir    = filepath.Join(os.TempDir(), "harmony-one", "harmony", "p2p", "discovery")
	emptyFile = filepath.Join(tmpDir, "empty_file")
	validPath = filepath.Join(tmpDir, "dht-1.1.1.1")
)

var (
	testAddrStr = []string{
		"/ip4/52.40.84.2/tcp/9800/p2p/QmbPVwrqWsTYXq1RxGWcxx9SWaTUCfoo1wA6wmdbduWe29",
		"/ip4/54.86.126.90/tcp/9800/p2p/Qmdfjtk6hPoyrH1zVD9PEH4zfWLo38dP2mDvvKXfh3tnEv",
	}
	testMAs, _      = p2putils.StringsToMultiAddrs(testAddrStr)
	testAddrList, _ = libp2p_peer.AddrInfosFromP2pAddrs(testMAs...)
)

func init() {
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, os.ModePerm)

	f, _ := os.Create(emptyFile)
	f.Close()
}

func TestDHTOption_getLibp2pRawOptions(t *testing.T) {
	tests := []struct {
		opt    DHTOption
		expLen int
		expErr error
	}{
		{
			opt: DHTOption{
				BootNodes: testAddrList,
			},
			expLen: 1,
		},
		{
			opt: DHTOption{
				BootNodes:     testAddrList,
				DataStoreFile: &validPath,
			},
			expLen: 2,
		},
		{
			opt: DHTOption{
				BootNodes:     testAddrList,
				DataStoreFile: &emptyFile,
			},
			expErr: errors.New("not a directory"),
		},
	}
	for i, test := range tests {
		opts, err := test.opt.getLibp2pRawOptions()
		if assErr := assertError(err, test.expErr); assErr != nil {
			t.Errorf("Test %v: %v", i, assErr)
		}
		if err != nil || test.expErr != nil {
			continue
		}
		if len(opts) != test.expLen {
			t.Errorf("Test %v: unexpected option size %v / %v", i, len(opts), test.expLen)
		}
	}
}

func assertError(got, expect error) error {
	if (got == nil) != (expect == nil) {
		return fmt.Errorf("unexpected error [%v] / [%v]", got, expect)
	}
	if (got == nil) || (expect == nil) {
		return nil
	}
	if !strings.Contains(got.Error(), expect.Error()) {
		return fmt.Errorf("unexpected error [%v] / [%v]", got, expect)
	}
	return nil
}
