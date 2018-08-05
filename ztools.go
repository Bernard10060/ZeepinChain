package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/mileschao/ZeepinChain/account"
	cmdcom "github.com/mileschao/ZeepinChain/cmd/common"
	clisvrcom "github.com/mileschao/ZeepinChain/cmd/sigsvr/common"
	sighandler "github.com/mileschao/ZeepinChain/cmd/sigsvr/handlers"
	"github.com/mileschao/ZeepinChain/cmd/utils"
	"github.com/mileschao/ZeepinChain/common"
	"github.com/mileschao/ZeepinChain/common/config"
	"github.com/mileschao/ZeepinChain/common/log"
	"github.com/mileschao/ZeepinChain/core/types"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/urfave/cli"
)

var (
	toolsConfigFlag = cli.StringFlag{
		Name:  "config",
		Usage: "Use `<filename>` to specifies the genesis block config file. If doesn't specifies the genesis block config, ZeepinChain will use Polaris config with VBFT consensus as default.",
	}
)

type toolsConfig struct {
	Asset  string
	Amount uint64
	Wallet []struct {
		FilePath string
		Passwd   string
		Address  string
	}
}

func sigMultiRawTx(cfg *toolsConfig) error {

	var accountList = make([]*account.Account, 0)
	pubKeys := make([]keypair.PublicKey, 0)
	var spubKeys = make([]string, 0)
	for _, walletFile := range cfg.Wallet {

		if !common.FileExisted(walletFile.FilePath) {
			log.Infof("Cannot find wallet file:%s. Please create wallet first", walletFile)
			return errors.New("Cannot find wallet file")
		}
		wallet, err := account.Open(walletFile.FilePath)
		if err != nil {
			return err
		}
		acc, err := cmdcom.GetAccountMulti(wallet, []byte(walletFile.Passwd), walletFile.Address)
		if err != nil {
			log.Infof("GetAccount error:%s", err)
			return err
		}
		log.Infof("Using account:%s", acc.Address.ToBase58())
		accountList = append(accountList, acc)
		pubKeys = append(pubKeys, acc.PublicKey)
		spubKeys = append(spubKeys, hex.EncodeToString(keypair.SerializePublicKey(acc.PublicKey)))
	}

	m := (5*len(pubKeys) + 6) / 7
	fromAddr, err := types.AddressFromMultiPubKeys(pubKeys, m)
	if err != nil {
		log.Errorf("TestSigMutilRawTransaction AddressFromMultiPubKeys error:%s", err)
		return err
	}
	defAcc := accountList[0]
	tx, err := utils.TransferTx(0, 0, cfg.Asset, fromAddr.ToBase58(), defAcc.Address.ToBase58(), cfg.Amount)
	if err != nil {
		log.Errorf("TransferTx error:%s", err)
		return err
	}
	buf := bytes.NewBuffer(nil)
	err = tx.Serialize(buf)
	if err != nil {
		log.Errorf("tx.Serialize error:%s", err)
		return err
	}

	rawReq := &sighandler.SigMutilRawTransactionReq{
		RawTx:   hex.EncodeToString(buf.Bytes()),
		M:       m,
		PubKeys: spubKeys,
	}
	data, err := json.Marshal(rawReq)
	if err != nil {
		log.Errorf("json.Marshal SigRawTransactionReq error:%s", err)
		return err
	}
	req := &clisvrcom.CliRpcRequest{
		Qid:    "q",
		Method: "sigmutilrawtx",
		Params: data,
	}
	for i, acct := range accountList {
		resp := &clisvrcom.CliRpcResponse{}
		clisvrcom.DefAccount = acct
		sighandler.SigMutilRawTransaction(req, resp)
		if resp.ErrorCode != clisvrcom.CLIERR_OK {
			log.Errorf("SigMutilRawTransaction failed,ErrorCode:%d ErrorString:%s", resp.ErrorCode, resp.ErrorInfo)
			return errors.New("SigMutilRawTransaction failed")
		}
		log.Infof("sigmultiRawTx: %d: %+v", i, resp)
	}
	return nil
}

func startToolsSvr(ctx *cli.Context) {
	logLevel := ctx.GlobalInt(utils.GetFlagName(utils.LogLevelFlag))
	log.InitLog(logLevel, log.PATH, log.Stdout)

	configFile := ctx.GlobalString(utils.GetFlagName(toolsConfigFlag))
	if common.FileExisted(configFile) {

	}
	var config = new(toolsConfig)
	msh, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Errorf("%s", err)
		return
	}
	if err := json.Unmarshal(msh, config); err != nil {
		log.Errorf("%s", err)
	}
	if err := sigMultiRawTx(config); err != nil {
		log.Errorf("%s", err)
	}
}

func setupToolsSvr() *cli.App {
	app := cli.NewApp()
	app.Usage = "ZeepinChain tools"
	app.Action = startToolsSvr
	app.Version = config.Version
	app.Copyright = "Copyright in 2018 The ZeepinChain Authors"
	app.Flags = []cli.Flag{
		toolsConfigFlag,
	}
	app.Before = func(context *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}
	return app
}

func main() {
	if err := setupToolsSvr().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
