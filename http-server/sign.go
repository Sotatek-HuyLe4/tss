package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/cmd"
	"github.com/bnb-chain/tss/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

type SignRequest struct {
	Home      string `form:"home" json:"home" xml:"home" binding:"required"`
	Vault     string `form:"vault" json:"vault" xml:"vault" binding:"required"`
	Password  string `form:"password" json:"password" xml:"password" binding:"required"`
	ChannelId string `form:"channel_id" json:"channel_id" xml:"channel_id" binding:"required"`
	RpcUrl    string `form:"rpc_url" json:"rpc_url" xml:"rpc_url" binding:"required"`
	ToAddress string `form:"to_address" json:"to_address" xml:"to_address" binding:"required"`
	Amount    string `form:"amount" json:"amount" xml:"amount" binding:"required"`
}

func (signRequest SignRequest) validate() error {
	if signRequest.Home == "" {
		return fmt.Errorf("home directory is required")
	}

	if signRequest.Vault == "" {
		return fmt.Errorf("vault is required")
	}

	if signRequest.Password == "" || len(signRequest.Password) < 9 {
		return fmt.Errorf("password is required and must be at least 9 characters long")
	}

	if signRequest.ChannelId == "" {
		return fmt.Errorf("channel id is required")
	}

	if signRequest.RpcUrl == "" {
		return fmt.Errorf("rpc url is required")
	}

	if signRequest.ToAddress == "" {
		return fmt.Errorf("to address is required")
	}

	if signRequest.Amount == "" {
		return fmt.Errorf("amount is required")
	}

	return nil
}

func sign(ctx *gin.Context) {
	var signRequest SignRequest
	if err := ctx.ShouldBindJSON(&signRequest); err != nil {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// validate request
	if err := signRequest.validate(); err != nil {
		Error(ctx, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	// prepare for sign
	home := fmt.Sprintf("%s/%s", HOME_DIR, signRequest.Home)
	initSignViper(
		home,
		signRequest.Vault,
		signRequest.Password,
		signRequest.ChannelId,
		signRequest.RpcUrl,
		signRequest.ToAddress,
		signRequest.Amount,
	)

	if err := common.ReadConfigFromHome(viper.GetViper(), false, home, signRequest.Vault, signRequest.Password); err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	initLogLevel(common.TssCfg)

	// run signing
	err := setMessage(signRequest.RpcUrl, signRequest.ToAddress, signRequest.Amount)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error())
		return
	}

	// start signing
	c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
	c.Start()

	Ok(ctx, gin.H{
		"raw_tx": fmt.Sprintf("0x%s", c.GetRawTx()),
	})
}

func setMessage(rpcUrl, toAddress, amount string) error {
	ctx := context.Background()
	wei, err := cmd.EtherToWei(amount)
	if err != nil {
		return err
	}

	// get the from address
	pubkey, err := common.LoadEcdsaPubkey(common.TssCfg.Home, common.TssCfg.Vault, common.TssCfg.Password)
	if err != nil {
		common.Panic(err)
	}
	fromAddress := client.GetEvmAddress(*pubkey)

	client.Logger.Infof(
		"rpcUrl: %s, fromAddress: %s, toAddress: %s, amount: %s, amount in wei: %s\n",
		rpcUrl, fromAddress, toAddress, amount, wei.String(),
	)

	// get evm client
	evmClient, err := ethclient.Dial(rpcUrl)
	if err != nil {
		common.Panic(err)
	}
	defer evmClient.Close()

	// get the chain id of the network
	chainId, err := evmClient.ChainID(ctx)
	if err != nil {
		common.Panic(err)
	}

	// get the nonce of the from address
	nonce, err := evmClient.PendingNonceAt(ctx, ethCommon.HexToAddress(fromAddress))
	if err != nil {
		common.Panic(err)
	}

	// build the transfer tx
	tx := cmd.BuildTransferTx(chainId, nonce, ethCommon.HexToAddress(toAddress), wei, 21000, cmd.BigGwei(2), cmd.BigGwei(30))

	// get the signer type
	signer := types.NewLondonSigner(chainId)
	txHash := signer.Hash(tx)

	// set the message to the tx hash
	common.TssCfg.Message = txHash.Hex()

	// set the sign config
	common.TssCfg.SignConfig = common.SignConfig{
		Tx:     tx,
		Signer: signer,
	}

	return nil
}
