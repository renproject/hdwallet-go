package hdwallet

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/google/uuid"
	"github.com/republicprotocol/libbtc-go"
)

const DefaultBitcoinFee = 10000

type btcGenerator struct {
	address string
	network string
}

func NewBTCGenerator(address, network string) Generator {
	return &btcGenerator{address, strings.ToLower(network)}
}

func (generator *btcGenerator) GenerateAddress(uuid uuid.UUID) (string, error) {
	net, err := networkParams(generator.network)
	if err != nil {
		return "", err
	}

	pkHash, err := addressToPubKeyHash(generator.address, net)
	if err != nil {
		return "", err
	}

	userSpecificScript, err := newScript(uuid, pkHash.Hash160()[:])
	if err != nil {
		return "", err
	}

	address, err := btcutil.NewAddressScriptHash(userSpecificScript, net)
	if err != nil {
		return "", err
	}

	return address.EncodeAddress(), nil
}

func addressToPubKeyHash(addrString string, chainParams *chaincfg.Params) (*btcutil.AddressPubKeyHash, error) {
	btcAddr, err := btcutil.DecodeAddress(addrString, chainParams)
	if err != nil {
		return nil, fmt.Errorf("address %s is not "+
			"intended for use on %v", addrString, chainParams.Name)
	}
	addr, ok := btcAddr.(*btcutil.AddressPubKeyHash)
	if !ok {
		return nil, errors.New("%s is not p2pkh address")
	}
	return addr, nil
}

func newScript(uuid uuid.UUID, pkHash []byte) ([]byte, error) {
	b := txscript.NewScriptBuilder()
	b.AddData(uuid[:])
	b.AddOp(txscript.OP_DROP)
	b.AddOp(txscript.OP_DUP)
	b.AddOp(txscript.OP_HASH160)
	b.AddData(pkHash[:])
	b.AddOp(txscript.OP_EQUALVERIFY)
	b.AddOp(txscript.OP_CHECKSIG)
	return b.Script()
}

type btcCollector struct {
	wif     string
	network string
}

func NewBTCCollector(wif, network string) Collector {
	return &btcCollector{wif, strings.ToLower(network)}
}

func (collector *btcCollector) Collect(ctx context.Context, address string, uuids []uuid.UUID) error {
	net, err := networkParams(collector.network)
	if err != nil {
		return err
	}
	wif, err := btcutil.DecodeWIF(collector.wif)
	if err != nil {
		return err
	}
	pubKeyBytes, err := SerializedPublicKey(wif.PrivKey.PubKey(), net)
	if err != nil {
		return err
	}
	AddrPubKey, err := btcutil.NewAddressPubKey(pubKeyBytes, net)
	if err != nil {
		return err
	}
	addr, err := btcutil.DecodeAddress(address, net)
	if err != nil {
		return err
	}

	pkHash := AddrPubKey.AddressPubKeyHash().Hash160()
	scripts := [][]byte{}

	for _, uuid := range uuids {
		script, err := newScript(uuid, pkHash[:])
		if err != nil {
			return err
		}
		scripts = append(scripts, script)
	}
	return collectBTC(ctx, addr, scripts, libbtc.NewAccount(libbtc.NewBlockchainInfoClient(collector.network), wif.PrivKey.ToECDSA()))
}

func collectBTC(ctx context.Context, address btcutil.Address, scripts [][]byte, account libbtc.Account) error {
	payToAddrScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return err
	}

	for _, script := range scripts {
		scriptAddress, err := btcutil.NewAddressScriptHash(script, account.NetworkParams())
		if err != nil {
			return err
		}

		if err := account.SendTransaction(
			ctx,
			script,
			DefaultBitcoinFee,
			nil,
			func(msgTx *wire.MsgTx) bool {
				balance, err := account.Balance(ctx, scriptAddress.EncodeAddress(), 0)
				if err != nil {
					return false
				}
				if balance < DefaultBitcoinFee {
					return false
				}
				msgTx.AddTxOut(wire.NewTxOut(balance-DefaultBitcoinFee, payToAddrScript))
				return true
			},
			nil,
			func(msgTx *wire.MsgTx) bool {
				spent, err := account.ScriptSpent(ctx, scriptAddress.EncodeAddress())
				if err != nil {
					return false
				}
				return spent
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func SerializedPublicKey(pubKey *btcec.PublicKey, net *chaincfg.Params) ([]byte, error) {
	switch net {
	case &chaincfg.MainNetParams:
		return pubKey.SerializeCompressed(), nil
	case &chaincfg.TestNet3Params:
		return pubKey.SerializeUncompressed(), nil
	default:
		return nil, fmt.Errorf("unsupported network: %v", net.Name)
	}
}

func networkParams(network string) (*chaincfg.Params, error) {
	switch network {
	case "testnet", "testnet3":
		return &chaincfg.TestNet3Params, nil
	case "mainnet":
		return &chaincfg.MainNetParams, nil
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}
}
