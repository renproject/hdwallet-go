package hdwallet_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing/quick"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hdwallet-go"
	"github.com/republicprotocol/beth-go"
)

var _ = Describe("HDWallet ETH", func() {
	getETHTestnetAccount := func() beth.Account {
		key, err := loadKey(44, 1, 0, 0, 0) // "m/44'/1'/0'/0/0"
		Expect(err).Should(BeNil())
		account, err := beth.NewAccount("https://kovan.infura.io", key)
		Expect(err).Should(BeNil())
		return account
	}

	getETHTestnetGenerator := func() Generator {
		key, err := loadKey(44, 1, 0, 0, 0) // "m/44'/1'/0'/0/0"
		Expect(err).Should(BeNil())
		return NewETHGenerator(hex.EncodeToString(crypto.FromECDSA(key)))
	}

	getETHTestnetCollector := func(erc20Tokens ...string) Collector {
		key, err := loadKey(44, 1, 0, 0, 0) // "m/44'/1'/0'/0/0"
		Expect(err).Should(BeNil())
		return NewETHCollector(hex.EncodeToString(crypto.FromECDSA(key)), "https://kovan.infura.io", erc20Tokens)
	}

	Context("when interacting with ethereum kovan testnet", func() {
		It("should generate valid testnet addresses", func() {
			generator := getETHTestnetGenerator()
			test := func(uuid [16]byte) bool {
				address, err := generator.GenerateAddress(uuid)
				if err != nil {
					return false
				}
				common.HexToAddress(address)
				return true
			}
			Expect(quick.Check(test, quickCheckConfig())).ShouldNot(HaveOccurred())
		})

		It("should deposit testnet ETH to generated addresses and collect them back", func() {
			account := getETHTestnetAccount()
			generator := getETHTestnetGenerator()
			collector := getETHTestnetCollector()

			uuids := []uuid.UUID{}
			test := func(uuid [16]byte) bool {
				uuids = append(uuids, uuid)
				address, err := generator.GenerateAddress(uuid)
				if err != nil {
					return false
				}
				txHash, err := account.Transfer(context.Background(), common.HexToAddress(address), big.NewInt(2500000000000000), 0)
				if err != nil {
					return false
				}
				fmt.Println(account.FormatTransactionView(fmt.Sprintf("deposit successful for %s", address), txHash))
				return true
			}
			Expect(quick.Check(test, &quick.Config{
				MaxCount: 4,
			})).ShouldNot(HaveOccurred())

			mainAddress := account.Address()
			err := collector.Collect(context.Background(), mainAddress.String(), uuids)
			Expect(err).Should(BeNil())
		})

		It("should deposit REN to generated addresses and collect them back", func() {
			account := getETHTestnetAccount()
			generator := getETHTestnetGenerator()
			collector := getETHTestnetCollector("REN")

			uuids := []uuid.UUID{}
			test := func(uuid [16]byte) bool {
				uuids = append(uuids, uuid)
				address, err := generator.GenerateAddress(uuid)
				if err != nil {
					return false
				}
				erc20, err := account.NewERC20("REN")
				if err != nil {
					return false
				}
				value, ok := new(big.Int).SetString("1000000000000000000", 10)
				if !ok {
					return false
				}
				txHash, err := erc20.Transfer(context.Background(), common.HexToAddress(address), value)
				if err != nil {
					return false
				}
				txLog, _ := account.FormatTransactionView(fmt.Sprintf("deposit successful for %s", address), txHash)
				fmt.Println(txLog)
				return true
			}
			Expect(quick.Check(test, &quick.Config{
				MaxCount: 4,
			})).ShouldNot(HaveOccurred())

			mainAddress := account.Address()
			err := collector.Collect(context.Background(), mainAddress.String(), uuids)
			Expect(err).Should(BeNil())
		})
	})
})
