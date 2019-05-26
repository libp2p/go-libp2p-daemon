package p2pd

import (
	"io/ioutil"

	"github.com/libp2p/go-libp2p-core/crypto"
)

func ReadIdentity(path string) (crypto.PrivKey, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return crypto.UnmarshalPrivateKey(bytes)
}

func WriteIdentity(k crypto.PrivKey, path string) error {
	bytes, err := crypto.MarshalPrivateKey(k)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, bytes, 0400)
}
