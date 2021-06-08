package lotus

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/civet148/log"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
)

const (
	LOTUS_SEND_METHOD   = 0
	LOTUS_SEND_WITHDRAW = 16
)

type SendReq struct {
	From       string        `json:"from"`
	To         string        `json:"to"`
	Amount     string        `json:"amount"`
	GasPremium string        `json:"gas-premium"`
	GasFeeCap  string        `json:"gas-feecap"`
	GasLimit   int64         `json:"gas_limit"`
	MethodNum  abi.MethodNum `json:"method_num"`
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
	log.Infof("SendJsonParam [%s]", string(data))
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

func Send(cctx *cli.Context, s *SendReq) (cid.Cid, error) {

	srv, err := GetFullNodeService(cctx)
	if err != nil {
		return cid.Cid{}, err
	}
	defer srv.Close()

	ctx := cliutil.ReqContext(cctx) //cctx.Context
	var params SendParams

	params.To, err = address.NewFromString(s.To)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("failed to parse target address: %w", err)
	}

	val, err := types.ParseFIL(s.Amount)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("failed to parse amount: %w", err)
	}
	params.Val = abi.TokenAmount(val)

	if from := s.From; from != "" {
		addr, err := address.NewFromString(from)
		if err != nil {
			return cid.Cid{}, err
		}
		params.From = addr
	}

	var jp = SendJsonParam{
		AmountRequested: params.Val,
	}

	gp, _ := types.BigFromString(s.GasPremium)
	gfc, _ := types.BigFromString(s.GasFeeCap)
	params.GasPremium = &gp
	params.GasFeeCap = &gfc
	params.GasLimit = &s.GasLimit
	params.Method = s.MethodNum

	if s.MethodNum == LOTUS_SEND_WITHDRAW {
		decparams, err := srv.DecodeTypedParamsFromJSON(ctx, params.To, params.Method, jp.Marshal())
		params.Params = decparams
		if err != nil {
			return cid.Cid{}, fmt.Errorf("failed to decode json params: %w", err)
		}
	}

	log.Infof("method [%s] params [%+v]...", params.Method.String(), params)
	msgCid, err := srv.Send(ctx, params)

	if err != nil {
		if errors.Is(err, ErrSendBalanceTooLow) {
			return cid.Cid{}, fmt.Errorf("send amount is too low, --force must be specified for this action, error %w", err)
		}
		return cid.Cid{}, xerrors.Errorf("executing send error %w", err)
	}
	log.Infof("send message CID [%s] waiting...\n", msgCid)
	var receipt *api.MsgLookup
	if receipt, err = srv.GetAPI().StateWaitMsg(ctx, msgCid, build.MessageConfidence); err != nil {
		log.Error(err.Error())
		return cid.Cid{}, err
	}
	log.Infof("send message CID [%s] receipt [OK] at height [%d]", msgCid, receipt.Height)
	return msgCid, nil
}
