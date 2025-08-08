package smartcontract

import (
	"context"
	"crypto/ecdsa"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var paymentABIJSON []byte

type PaymentClient struct {
	client       *ethclient.Client
	contract     *bind.BoundContract
	auth         *bind.TransactOpts
	address      common.Address
	chainID      *big.Int
	privateKey   *ecdsa.PrivateKey
	contractAddr common.Address
}

func NewPaymentClient(rpcURL, contractAddrHex, privateKeyHex string) (*PaymentClient, error) {
	conn, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect RPC: %w", err)
	}

	if len(paymentABIJSON) == 0 {
		return nil, errors.New("embedded ABI is empty; make sure abi/Payment.abi.json exists")
	}
	parsedABI, err := abi.JSON(strings.NewReader(string(paymentABIJSON)))
	if err != nil {
		return nil, fmt.Errorf("parse ABI: %w", err)
	}

	if !common.IsHexAddress(contractAddrHex) {
		return nil, fmt.Errorf("invalid contract address: %s", contractAddrHex)
	}
	contractAddr := common.HexToAddress(contractAddrHex)

	pk, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	pub, ok := pk.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("cannot cast public key")
	}
	from := crypto.PubkeyToAddress(*pub)

	chainID, err := conn.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get chain id: %w", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return nil, fmt.Errorf("new transactor: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h, _ := conn.HeaderByNumber(ctx, nil)
	if h != nil && h.BaseFee != nil {
		if tip, err := conn.SuggestGasTipCap(ctx); err == nil {
			auth.GasTipCap = tip
			feeCap := new(big.Int).Mul(h.BaseFee, big.NewInt(2))
			feeCap.Add(feeCap, tip)
			auth.GasFeeCap = feeCap
		}
	} else {
		if gp, err := conn.SuggestGasPrice(ctx); err == nil {
			auth.GasPrice = gp
		}
	}

	bound := bind.NewBoundContract(contractAddr, parsedABI, conn, conn, conn)

	return &PaymentClient{
		client:       conn,
		contract:     bound,
		auth:         auth,
		address:      from,
		chainID:      chainID,
		privateKey:   pk,
		contractAddr: contractAddr,
	}, nil
}

func (pc *PaymentClient) SendReward(to string, amount *big.Int) (txHash string, err error) {
	if !common.IsHexAddress(to) {
		return "", fmt.Errorf("invalid recipient: %s", to)
	}
	recipient := common.HexToAddress(to)

	pc.auth.Nonce = nil
	pc.auth.Value = big.NewInt(0)
	pc.auth.GasLimit = 0
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pc.auth.Context = ctx

	tx, err := pc.contract.Transact(pc.auth, "payMiner", recipient, amount)
	if err != nil {
		if revertReason := detectRevertReason(ctx, pc.client, pc.contractAddr, pc.address, "payMiner", recipient, amount); revertReason != "" {
			return "", fmt.Errorf("contract transact (payMiner) reverted: %s", revertReason)
		}
		return "", fmt.Errorf("contract transact (payMiner): %w", err)
	}

	log.Printf("TX submitted: %s -> contract %s", tx.Hash().Hex(), pc.contractAddr.Hex())
	return tx.Hash().Hex(), nil
}

func (pc *PaymentClient) CallPaused(ctx context.Context) (bool, error) {
	var out []interface{}
	if err := pc.contract.Call(&bind.CallOpts{Context: ctx}, &out, "paused"); err != nil {
		return false, err
	}
	if len(out) != 1 {
		return false, fmt.Errorf("unexpected output len: %d", len(out))
	}
	val, ok := out[0].(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type %T", out[0])
	}
	return val, nil
}

func (pc *PaymentClient) Owner(ctx context.Context) (common.Address, error) {
	var out []interface{}
	if err := pc.contract.Call(&bind.CallOpts{Context: ctx}, &out, "owner"); err != nil {
		return common.Address{}, err
	}
	if len(out) != 1 {
		return common.Address{}, fmt.Errorf("unexpected output len: %d", len(out))
	}
	addr, ok := out[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("unexpected type %T", out[0])
	}
	return addr, nil
}

func (pc *PaymentClient) Close() {
	if pc.client != nil {
		pc.client.Close()
	}
}

func detectRevertReason(ctx context.Context, cli *ethclient.Client, contract common.Address, from common.Address, method string, args ...interface{}) string {
	parsedABI, err := abi.JSON(strings.NewReader(string(paymentABIJSON)))
	if err != nil {
		return ""
	}
	data, err := parsedABI.Pack(method, args...)
	if err != nil {
		return ""
	}
	callMsg := ethereum.CallMsg{
		From: from, To: &contract, Gas: 0, GasPrice: nil,
		Value: big.NewInt(0), Data: data,
	}
	_, err = cli.CallContract(ctx, callMsg, nil)
	if err != nil {
		return err.Error()
	}
	return ""
}

func (pc *PaymentClient) ChainID() *big.Int { return new(big.Int).Set(pc.chainID) }

func (pc *PaymentClient) TxType() string {
	if pc.auth.GasFeeCap != nil || pc.auth.GasTipCap != nil {
		return "dynamic"
	}
	return "legacy"
}

type MockPaymentClient struct{}

func NewMockPaymentClient() *MockPaymentClient { return &MockPaymentClient{} }

func (m *MockPaymentClient) SendReward(to string, amount *big.Int) (string, error) {
	if to == "" || amount == nil || amount.Sign() <= 0 {
		return "", errors.New("invalid params")
	}
	time.Sleep(300 * time.Millisecond)
	return fakeTxHash(), nil
}

func fakeTxHash() string {
	const hex = "0123456789abcdef"
	b := make([]byte, 64)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return "0x" + string(b)
}
