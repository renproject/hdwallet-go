package hdwallet_test

import (
	"context"
	"fmt"
	"testing/quick"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hdwallet-go"
	"github.com/republicprotocol/libbtc-go"
)

var _ = Describe("HDWallet BTC", func() {
	getBTCTestnetAccount := func() libbtc.Account {
		client := libbtc.NewBlockchainInfoClient("testnet")
		key, err := loadKey(44, 1, 0, 0, 0) // "m/44'/1'/0'/0/0"
		Expect(err).Should(BeNil())
		return libbtc.NewAccount(client, key)
	}

	getBTCTestnetGenerator := func() Generator {
		key, err := loadKey(44, 1, 0, 0, 0) // "m/44'/1'/0'/0/0"
		Expect(err).Should(BeNil())
		pubKeyBytes, err := SerializedPublicKey((*btcec.PrivateKey)(key).PubKey(), &chaincfg.TestNet3Params)
		Expect(err).Should(BeNil())
		addrPubKey, err := btcutil.NewAddressPubKey(pubKeyBytes, &chaincfg.TestNet3Params)
		Expect(err).Should(BeNil())
		return NewBTCGenerator(addrPubKey.EncodeAddress(), "testnet")
	}

	getBTCTestnetCollector := func() Collector {
		key, err := loadKey(44, 1, 0, 0, 0) // "m/44'/1'/0'/0/0"
		Expect(err).Should(BeNil())
		wif, err := btcutil.NewWIF((*btcec.PrivateKey)(key), &chaincfg.TestNet3Params, false)
		Expect(err).Should(BeNil())
		return NewBTCCollector(wif.String(), "testnet")
	}

	Context("when interacting with bitcoin testnet", func() {
		It("should generate valid testnet addresses", func() {
			generator := getBTCTestnetGenerator()
			test := func(uuid [16]byte) bool {
				address, err := generator.GenerateAddress(uuid)
				if err != nil {
					return false
				}
				_, err = btcutil.DecodeAddress(address, &chaincfg.TestNet3Params)
				return err == nil
			}
			Expect(quick.Check(test, quickCheckConfig())).ShouldNot(HaveOccurred())
		})

		It("should deposit testnet BTC to generated addresses and collect them back", func() {
			account := getBTCTestnetAccount()
			generator := getBTCTestnetGenerator()
			collector := getBTCTestnetCollector()

			uuids := []uuid.UUID{}
			test := func(uuid [16]byte) bool {
				uuids = append(uuids, uuid)
				address, err := generator.GenerateAddress(uuid)
				if err != nil {
					return false
				}
				txHash, err := account.Transfer(context.Background(), address, 25000)
				if err != nil {
					return false
				}
				fmt.Println(account.FormatTransactionView(fmt.Sprintf("deposit successful for %s", address), txHash))
				return true
			}
			Expect(quick.Check(test, &quick.Config{
				MaxCount: 4,
			})).ShouldNot(HaveOccurred())

			mainAddress, err := account.Address()
			Expect(err).Should(BeNil())

			err = collector.Collect(context.Background(), mainAddress.EncodeAddress(), uuids)
			Expect(err).Should(BeNil())
		})
	})
})
