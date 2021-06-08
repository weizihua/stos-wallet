package main

import (
	"fmt"
	"github.com/civet148/log"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/urfave/cli/v2"
	"os"
	"stos-wallet/build"
	"stos-wallet/lotus"
	"stos-wallet/types"
)

var (
	BuildTime = "2021-06-08"
	GitCommit = ""
)

const (
	PROGRAM_NAME = "stos-wallet"
	CMD_NAME_RUN = "run"
)

const (
	LOTUS_ENV_FULLNODE_API = "FULLNODE_API_INFO"
	LOTUS_ENV_MINER_API    = "MINER_API_INFO"
)

func checkApiEnv() (err error) {

	strFullNodeApi := os.Getenv(LOTUS_ENV_FULLNODE_API)
	if strFullNodeApi == "" {
		err = fmt.Errorf("environment %s not set", LOTUS_ENV_FULLNODE_API)
		return
	}
	return
}

func main() {
	if err := checkApiEnv(); err != nil {
		log.Error(err.Error())
		return
	}
	local := []*cli.Command{
		runCmd,
		sendCmd,
	}
	app := &cli.App{
		Name:     PROGRAM_NAME,
		Usage:    "stos manager",
		Version:  fmt.Sprintf("v%s %s commit %s", build.Version, BuildTime, GitCommit),
		Flags:    []cli.Flag{},
		Commands: local,
		Action:   nil,
	}
	if err := app.Run(os.Args); err != nil {
		log.Errorf("exit in error %s", err)
		os.Exit(1)
		return
	}
}

var runCmd = &cli.Command{
	Name:  CMD_NAME_RUN,
	Usage: "run as a web service",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {

		var strHttpAddr string

		strHttpAddr = cctx.Args().First()
		if strHttpAddr == "" {
			strHttpAddr = types.DEFAULT_HTTP_LISTEN_ADDR
		}

		return nil
	},
}

var sendCmd = &cli.Command{
	Name:      "send",
	Usage:     "Send funds between accounts",
	ArgsUsage: "[target address] [amount]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "optionally specify the account to send funds from",
		},
		&cli.Uint64Flag{
			Name:  "method",
			Usage: "specify method to invoke",
			Value: lotus.LOTUS_SEND_METHOD,
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "specify gas price to use in AttoFIL",
			Value: "0",
		},
		&cli.StringFlag{
			Name:  "gas-feecap",
			Usage: "specify gas fee cap to use in AttoFIL",
			Value: "0",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "specify gas limit",
			Value: 0,
		},
		&cli.Uint64Flag{
			Name:  "nonce",
			Usage: "specify the nonce to use",
			Value: 0,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() != 2 {
			return fmt.Errorf("'send' expects two arguments, target and amount")
		}
		var s = &lotus.SendReq{
			From:       cctx.String("from"),
			To:         cctx.Args().Get(0),
			Amount:     cctx.Args().Get(1),
			GasPremium: cctx.String("gas-premium"),
			GasFeeCap:  cctx.String("gas-feecap"),
			GasLimit:   cctx.Int64("gas-limit"),
			MethodNum:  abi.MethodNum(cctx.Uint64("method")),
		}
		_, err := lotus.Send(cctx, s)
		return err
	},
}
