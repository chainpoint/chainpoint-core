package abci

import (
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/blake2s"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/google/uuid"
	"net/http"
	"regexp"
	"strings"
	"time"
	"encoding/json"
	)

type Hash struct {
	Hash string `json:"hash`
}

type HashResponse struct {
	Hash 			string 			`json:"hash`
	ProofId 		string 			`json:"proof_id"`
	HashReceived 	string 			`json:"hash_received"`
	ProcessingHints ProcessingHints `json:"processing_hints"`
}

type ProcessingHints struct {
	CalHint string `json:"cal"`
	BtcHint string `json:"btc"`
}

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
	contentType := r.Header.Get("Content-type")
	if contentType != "application/json" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid content type"})
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	hash := Hash{}
	err := d.Decode(&hash)
	if app.LogError(err) != nil || len(hash.Hash) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body: missing hash"})
	}
	match, err := regexp.MatchString("^([a-fA-F0-9]{2}){20,64}$", hash.Hash)
	if app.LogError(err) != nil || !match {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body: bad hash submitted"})
	}

	// TODO: add LSAT/WHITELIST distinction here

	// compute uuid using blake2s
	unixTimeMS := string(time.Now().UnixNano() / int64(time.Millisecond))
	timeLength := string(len(unixTimeMS))
	hashStr := strings.Join([]string{unixTimeMS, timeLength, hash.Hash, string(len(hash.Hash))}, ":")
	blakeHash, err := blake2s.New256WithPersonalization(nil, []byte("CHAINPNT"))
	blakeHash.Write([]byte(hashStr))
	if app.LogError(err) != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cannot compute blake2s hash"})
	}
	node := append([]byte{0x01}, blakeHash.Sum([]byte{})...)
	uuid.SetNodeID(node)
	uuid, err := uuid.NewUUID()
	if app.LogError(err) != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cannot compute uuid"})
	}
	proofId := uuid.String()
	hashResponse := HashResponse{
		Hash:            hash.Hash,
		ProofId:         proofId,
		HashReceived:    time.Now().Format(time.RFC3339),
		ProcessingHints: ProcessingHints{
			CalHint: time.Now().Add(140 * time.Second).Format(time.RFC3339),
			BtcHint: time.Now().Add(90 * time.Minute).Format(time.RFC3339),
		},
	}
	//hashItem := types.HashItem{
	//	ProofID: proofId,
	//	Hash:    hashStr,
	//}
	// Add hash item to aggregator

	respondJSON(w, http.StatusOK, hashResponse)
}