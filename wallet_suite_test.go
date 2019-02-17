package hdwallet_test

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"testing/quick"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tyler-smith/go-bip39"
)

func TestHdwalletGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HdwalletGo Suite")
}

func loadMasterKey(network uint32) (*hdkeychain.ExtendedKey, error) {
	switch network {
	case 1:
		seed := bip39.NewSeed(os.Getenv("TESTNET_MNEMONIC"), os.Getenv("TESTNET_PASSPHRASE"))
		return hdkeychain.NewMaster(seed, &chaincfg.TestNet3Params)
	case 0:
		seed := bip39.NewSeed(os.Getenv("MNEMONIC"), os.Getenv("PASSPHRASE"))
		return hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	default:
		return nil, fmt.Errorf("unsupported network id: %d", network)
	}
}

func loadKey(path ...uint32) (*ecdsa.PrivateKey, error) {
	key, err := loadMasterKey(path[1])
	if err != nil {
		return nil, err
	}
	for _, val := range path {
		key, err = key.Child(val)
		if err != nil {
			return nil, err
		}
	}
	privKey, err := key.ECPrivKey()
	if err != nil {
		return nil, err
	}
	return privKey.ToECDSA(), nil
}

func quickCheckConfig() *quick.Config {
	return &quick.Config{
		Rand:     rand.New(rand.NewSource(time.Now().Unix())),
		MaxCount: 512,
	}
}
