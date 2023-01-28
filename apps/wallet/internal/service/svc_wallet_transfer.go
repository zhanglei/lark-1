package service

import (
	"context"
	"lark/domain/po"
	"lark/pkg/proto/pb_wallet"
)

func (s *walletService) Transfer(ctx context.Context, req *pb_wallet.TransferReq) (resp *pb_wallet.TransferResp, err error) {
	tr := po.TransRecord{
		Tsn:          0,
		Uid:          req.FromUid,
		FromWid:      req.FromWid,
		ToWid:        req.ToWid,
		ToAccount:    req.ToAccount,
		AccountType:  int(req.AccountType),
		Amount:       req.Amount,
		ExcRate:      "",
		ExcAmount:    0,
		Status:       0,
		TraderRole:   0,
		TransType:    0,
		Appid:        "",
		MchId:        "",
		OutTradeNo:   "",
		TradeNo:      "",
		Description:  "",
		Attach:       "",
		NotifyUrl:    "",
		NotifyResult: "",
		NotifyTs:     0,
	}
	if tr.ToWid > 0 {

	}
	return
}
