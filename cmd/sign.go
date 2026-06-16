package cmd

import (
	"context"

	"github.com/bnb-chain/tss/client"
	"github.com/bnb-chain/tss/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

func init() {
	rootCmd.AddCommand(signCmd)
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "sign a transaction",
	Long:  "sign a transaction using local share, signers will be prompted to fill in",
	PreRun: func(cmd *cobra.Command, args []string) {
		vault := askVault()
		passphrase := askPassphrase()
		if err := common.ReadConfigFromHome(viper.GetViper(), false, viper.GetString(flagHome), vault, passphrase); err != nil {
			common.Panic(err)
		}

		initLogLevel(common.TssCfg)
	},
	Run: func(cmd *cobra.Command, args []string) {
		setChannelId()
		setChannelPasswd()
		setMessage()

		c := client.NewTssClient(&common.TssCfg, client.SignMode, false)
		c.Start()
	},
}

func setMessage() {
	ctx := context.Background()

	// get the rpc url, to address, and amount from the command line
	rpcUrl := viper.GetString("rpc_url")
	toAddress := viper.GetString("to_address")
	amount := viper.GetString("amount")
	wei, err := etherToWei(amount)
	if err != nil {
		common.Panic(err)
	}

	// get the from address
	pubkey, err := common.LoadEcdsaPubkey(common.TssCfg.Home, common.TssCfg.Vault, common.TssCfg.Password)
	if err != nil {
		common.Panic(err)
	}
	fromAddress := client.GetEvmAddress(*pubkey)

	client.Logger.Infof("rpcUrl: %s, fromAddress: %s, toAddress: %s, amount: %s\n, amount in wei: %s\n", rpcUrl, fromAddress, toAddress, amount, wei.String())

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
	tx := buildTransferTx(chainId, nonce, ethCommon.HexToAddress(toAddress), wei, 21000, bigGwei(2), bigGwei(30))

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
}
