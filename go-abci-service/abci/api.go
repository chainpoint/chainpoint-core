package abci

import (
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"net/http"
	"time"
)

func (app *AnchorApplication) HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	fmt.Fprintf(w, "This is an API endpoint. Please consult https://chainpoint.org")
}


func (app *AnchorApplication) StatusHandler(w http.ResponseWriter, r *http.Request) {
	status, err := app.rpc.GetStatus()
	info, err := app.lnClient.GetInfo()
	balance, err := app.lnClient.GetWalletBalance()
	apiStatus := types.CoreAPIStatus{
		Version: "0.0.2",
		Time: time.Now().UTC().Format("2006-01-02T15:04:05.999Z07:00"),
		BaseURI: util.GetEnv("CHAINPOINT_CORE_BASE_URI", "http://0.0.0.0"),
		Network: app.config.BitcoinNetwork,
		IdentityPubkey: info.IdentityPubkey,
		LightningAddress: app.config.LightningConfig.WalletAddress,
		Uris: info.Uris,
		ActiveChannelsCount:  int(info.NumActiveChannels),
		Alias: info.Alias,
		HashPriceSatoshis: app.config.HashPrice,

	}
	w.WriteHeader(http.StatusTeapot)
	fmt.Fprintf(w, "This is an API endpoint. Please consult https://chainpoint.org")
}