package cmd

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/bnb-chain/tss/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/libp2p/go-libp2p"
	"github.com/multiformats/go-multiaddr"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

func getListenAddrs(listenAddr string) string {
	addr, err := multiaddr.NewMultiaddr(listenAddr)
	if err != nil {
		common.Panic(err)
	}
	host, err := libp2p.New(context.Background(), libp2p.ListenAddrs(addr))
	if err != nil {
		common.Panic(err)
	}

	builder := strings.Builder{}
	for i, addr := range host.Addrs() {
		if i > 0 {
			fmt.Fprint(&builder, ", ")
		}
		fmt.Fprintf(&builder, "%s", addr)
	}
	host.Close()
	return builder.String()
}

func etherToWei(amount string) (*big.Int, error) {
	// parse the amount as a float64
	etherFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, err
	}

	// convert the amount to wei
	wei := new(big.Int)
	floatWei := new(big.Float).Mul(big.NewFloat(etherFloat), big.NewFloat(1e18))

	// convert the float wei to a big.Int
	floatWei.Int(wei)

	return wei, nil
}

func buildTransferTx(
	chainId *big.Int,
	nonce uint64,
	to ethCommon.Address,
	value *big.Int,
	gasLimit uint64,
	gasTipCap *big.Int,
	gasFeeCap *big.Int,
) *types.Transaction {
	txData := &types.DynamicFeeTx{
		ChainID:   chainId,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     value,
		Data:      nil,
	}

	return types.NewTx(txData)
}

func bigGwei(g int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(g), big.NewInt(1000000000))
}
