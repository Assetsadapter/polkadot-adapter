package openwtester

import (
	"github.com/Assetsadapter/polkadot-adapter/polkadot"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openw"
)

func init() {
	//注册钱包管理工具
	log.Notice("Wallet Manager Load Successfully.")
	openw.RegAssets(polkadot.Symbol, polkadot.NewWalletManager())
}
