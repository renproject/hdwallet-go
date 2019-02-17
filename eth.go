package hdwallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/republicprotocol/beth-go"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
)

type ethGenerator struct {
	privateKey string
}

func NewETHGenerator(privateKey string) Generator {
	return &ethGenerator{
		privateKey: privateKey,
	}
}

func (generator *ethGenerator) GenerateAddress(uuid uuid.UUID) (string, error) {
	privKeyBytes, err := hex.DecodeString(generator.privateKey)
	if err != nil {
		return "", fmt.Errorf("invalid privatekey: %v", err)
	}

	privKey, err := crypto.HexToECDSA(hex.EncodeToString(append(uuid[:], privKeyBytes[16:]...)))
	if err != nil {
		return "", fmt.Errorf("failed to generate a new private key: %v", err)
	}

	return crypto.PubkeyToAddress(privKey.PublicKey).String(), nil
}

type ethCollector struct {
	url                     string
	privateKey              string
	tokenAddressesOrAliases []string
}

func NewETHCollector(privateKey, url string, tokenAddressesOrAliases []string) Collector {
	return &ethCollector{
		privateKey:              privateKey,
		url:                     url,
		tokenAddressesOrAliases: tokenAddressesOrAliases,
	}
}

func (collector *ethCollector) Collect(ctx context.Context, address string, uuids []uuid.UUID) error {
	accounts := []beth.Account{}
	privKeyBytes, err := hex.DecodeString(collector.privateKey)
	if err != nil {
		return fmt.Errorf("invalid privatekey: %v", err)
	}

	for _, uuid := range uuids {
		account, err := collector.newAccount(hex.EncodeToString(append(uuid[:], privKeyBytes[16:]...)))
		if err != nil {
			return fmt.Errorf("failed to create beth account: %v", err)
		}
		accounts = append(accounts, account)
	}

	if err := collector.depositFees(ctx, accounts); err != nil {
		return err
	}

	for _, tokenAddressOrAlias := range collector.tokenAddressesOrAliases {
		if err := collectERC20(ctx, address, accounts, tokenAddressOrAlias); err != nil {
			return err
		}
	}

	if err := collectEther(ctx, address, accounts); err != nil {
		return err
	}
	return nil
}

func (collector *ethCollector) newAccount(privKey string) (beth.Account, error) {
	key, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}
	return beth.NewAccount(collector.url, key)
}

func (collector *ethCollector) depositFees(ctx context.Context, userAccounts []beth.Account) error {
	amount := new(big.Int).Mul(big.NewInt(int64(1200000*len(collector.tokenAddressesOrAliases))), big.NewInt(1000000000))
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil
	}

	account, err := collector.newAccount(collector.privateKey)
	if err != nil {
		return err
	}

	for _, userAccount := range userAccounts {
		tx, err := account.Transfer(ctx, userAccount.Address(), amount, 0)
		if err != nil {
			return err
		}
		txLog, _ := account.FormatTransactionView("depositing fees successful", tx)
		fmt.Println(txLog)
	}
	return nil
}

func collectEther(ctx context.Context, address string, accounts []beth.Account) error {
	for _, account := range accounts {
		if err := withdrawEther(ctx, address, account); err != nil {
			return err
		}
	}
	return nil
}

func collectERC20(ctx context.Context, address string, accounts []beth.Account, tokenAddressOrAlias string) error {
	for _, account := range accounts {
		if err := withdrawERC20(ctx, address, account, tokenAddressOrAlias); err != nil {
			return err
		}
	}
	return nil
}

func withdrawEther(ctx context.Context, address string, account beth.Account) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if bal, err := account.BalanceAt(ctx, nil); err != nil || bal.Cmp(big.NewInt(0)) == 0 {
		return err
	}
	tx, err := account.Transfer(ctx, common.HexToAddress(address), nil, 0)
	if err != nil {
		return err
	}
	txLog, _ := account.FormatTransactionView("ETH withdrawal successful", tx)
	fmt.Println(txLog)
	return nil
}

func withdrawERC20(ctx context.Context, address string, account beth.Account, tokenAddressOrAlias string) error {
	erc20, err := account.NewERC20(tokenAddressOrAlias)
	if err != nil {
		return err
	}
	if bal, err := erc20.BalanceOf(ctx, account.Address()); err != nil || bal.Cmp(big.NewInt(0)) == 0 {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	tx, err := erc20.Transfer(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return err
	}
	txLog, _ := account.FormatTransactionView(fmt.Sprintf("ERC20 (%s) withdrawal successful", tokenAddressOrAlias), tx)
	fmt.Println(txLog)
	return nil
}
