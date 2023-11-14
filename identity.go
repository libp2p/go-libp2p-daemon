package p2pd

import (
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func ReadIdentity(path string) (crypto.PrivKey, error) {
	bytes, err := os.ReadFile(path)
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

	return os.WriteFile(path, bytes, 0400)
}
