package abci

import (
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"net/http"
	"time"
	"encoding/json"
	)

func (app *AnchorApplication) HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	fmt.Fprintf(w, "This is an API endpoint. Please consult https://chainpoint.org")
}

// respondJSON makes the response with payload as json format
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

func (app *AnchorApplication) StatusHandler(w http.ResponseWriter, r *http.Request) {
	status, err := app.rpc.GetStatus()
	if app.LogError(err) != nil {
		errorMessage := map[string]interface{}{"error": "Could not query for status"}
		respondJSON(w, http.StatusInternalServerError, errorMessage)
		return
	}
	info, err := app.lnClient.GetInfo()
	if app.LogError(err) != nil {
		errorMessage := map[string]interface{}{"error": "Could not query for status"}
		respondJSON(w, http.StatusInternalServerError, errorMessage)
		return
	}
	balance, err := app.lnClient.GetWalletBalance()
	if app.LogError(err) != nil {
		errorMessage := map[string]interface{}{"error": "Could not query for balance"}
		respondJSON(w, http.StatusInternalServerError, errorMessage)
		return
	}
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
	apiStatus.LightningBalance.UnconfirmedBalance = string(balance.UnconfirmedBalance)
	apiStatus.LightningBalance.ConfirmedBalance = string(balance.ConfirmedBalance)
	apiStatus.LightningBalance.TotalBalance = string(balance.TotalBalance)
	apiStatus.TotalStakePrice = app.state.LnStakePrice
	apiStatus.ValidatorStakePrice = app.state.LnStakePerVal
	apiStatus.Jwk = app.JWK
	apiStatus.NodeInfo = status.NodeInfo
	apiStatus.ValidatorInfo = status.ValidatorInfo
	apiStatus.SyncInfo = status.SyncInfo
	respondJSON(w, http.StatusOK, apiStatus)
}

func (app *AnchorApplication) HashHandler(w http.ResponseWriter, r *http.Request) {

}