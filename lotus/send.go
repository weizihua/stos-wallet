package lotus

import (
	"encoding/json"
	"fmt"
	"github.com/civet148/log"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/urfave/cli/v2"
	"os"
)

type Send struct {
	From      string        `json:"from"`
	To        string        `json:"to"`
	Amount    string        `json:"amount"`
	MethodNum abi.MethodNum `json:"method_num"`
}

//100FIL -> '{"AmountRequested":"100000000000000000000"}'
type SendJsonParam struct {
	AmountRequested abi.TokenAmount `json:"AmountRequested"`
}

func (m *SendJsonParam) Marshal() string {
	data, err := json.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("marshal error [%s]", err))
	}
	return string(data)
}

func GetFullNodeService(ctx *cli.Context) (ServicesAPI, error) {
	strFullNodeApi := os.Getenv(LOTUS_ENV_FULLNODE_API)
	lotusBuilder, err := NewLotusBuilder(strFullNodeApi, 3)
	if err != nil {
		log.Errorf(err.Error())
		return nil, err
	}
	fullApi, fullCloser, err := lotusBuilder(ctx.Context)
	if err != nil {
		panic(err.Error())
	}
	return &ServicesImpl{api: fullApi, closer: fullCloser}, nil
}

func LotusSend(cctx *cli.Context, send *Send) error {

	srv, err := GetFullNodeService(cctx)
	if err != nil {
		return err
	}
	defer srv.Close()

	ctx := cctx.Context
	var params SendParams

	params.To, err = address.NewFromString(send.To)
	if err != nil {
		return fmt.Errorf("failed to parse target address: %w", err)
	}

	val, err := types.ParseFIL(send.Amount)
	if err != nil {
		return fmt.Errorf("failed to parse amount: %w", err)
	}
	params.Val = abi.TokenAmount(val)

	if from := send.From; from != "" {
		addr, err := address.NewFromString(from)
		if err != nil {
			return err
		}
		params.From = addr
	}

	var jp = SendJsonParam{
		AmountRequested: params.Val,
	}
	decparams, err := srv.DecodeTypedParamsFromJSON(ctx, params.To, params.Method, jp.Marshal())
	if err != nil {
		return fmt.Errorf("failed to decode json params: %w", err)
	}
	params.Params = decparams

	params.Method = abi.MethodNum(cctx.Uint64("method"))
	log.Infof("send method [%d] params [%+v]...")
	//msgCid, err := srv.Send(ctx, params)
	//
	//if err != nil {
	//	if errors.Is(err, ErrSendBalanceTooLow) {
	//		return fmt.Errorf("--force must be specified for this action to have an effect; you have been warned: %w", err)
	//	}
	//	return xerrors.Errorf("executing send: %w", err)
	//}
	//
	//fmt.Fprintf(cctx.App.Writer, "%s\n", msgCid)
	return nil
}
