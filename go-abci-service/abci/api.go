package abci

import (
	"encoding/hex"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/blake2s"
	"github.com/chainpoint/chainpoint-core/go-abci-service/proof"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

	// Add hash item to aggregator
	app.aggregator.AddHashItem(types.HashItem{Hash:hash.Hash, ProofID: proofId})
	respondJSON(w, http.StatusOK, hashResponse)
}

func (app *AnchorApplication) ProofHandler(w http.ResponseWriter, r *http.Request) {
	proofidHeader := r.Header.Get("proofids")
	if len(proofidHeader) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request, at least one hash id required"})
	}
	proofids := strings.Split(strings.ReplaceAll(proofidHeader," ","") , ",")
	if len(proofids) > 250 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request, too many hash ids (250 max)"})
	}
	for _, id := range proofids {
		_, err := uuid.Parse(id)
		if app.LogError(err) != nil {
			errStr := fmt.Sprintf("invalid request, bad proof_id: %s", id)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": errStr})
		}
	}
	proofStates, err := app.pgClient.GetProofsByProofIds(proofids)
	if app.LogError(err) != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not retrieve proofs"})
	}
	response := make([]proof.P, 0)
	for _, id := range proofids {
		if val, exists := proofStates[id]; exists {
			response = append(response, map[string]interface{}{"proof_id":id, "proof":val.Proof})
		} else {
			response = append(response, map[string]interface{}{"proof_id":id, "proof":nil})
		}
	}
	respondJSON(w, http.StatusOK, response)
}

func (app *AnchorApplication) CalHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, exists := vars["txid"]; exists {
		result, err := app.rpc.GetTxByHash(vars["txid"])
		if app.LogError(err) != nil {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not retrieve tx"})
		}
		respondJSON(w, http.StatusOK, result)
	}
	respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": "tx parameter required"})
}

func (app *AnchorApplication) CalDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, exists := vars["txid"]; exists {
		result, err := app.rpc.GetTxByHash(vars["txid"])
		if app.LogError(err) != nil {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not retrieve tx"})
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		data := hex.EncodeToString(result.TxResult.Data) //TODO: check encoding on this
		w.Write([]byte(data))
	}
	respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": "tx parameter required"})
}

func (app *AnchorApplication) PeerHandler(w http.ResponseWriter, r *http.Request) {
	peers := app.GetPeers()
	peerList := []string{}
	for _, peer := range peers {
		var finalIp string
		ip := peer.RemoteIP
		if len(ip) == 0 {
			continue
		}
		firstOctet := ip[0 : strings.Index(ip, ".")]
		privateRanges := map[string]bool{
			"10": true,
			"172": true,
			"192": true,
		}
		if _, exists := privateRanges[firstOctet]; exists {
			listenAddr := peer.NodeInfo.ListenAddr
			if strings.Contains(listenAddr, "//"){
				finalIp = listenAddr[strings.LastIndex(listenAddr, "/") + 1 : strings.LastIndex(listenAddr, ":")]
			}
			finalIp = listenAddr[0 : strings.LastIndex(listenAddr, ":")]
		} else {
			finalIp = ip
		}
		peerList = append(peerList, finalIp)
	}
	respondJSON(w, http.StatusOK, peerList)
}

func (app *AnchorApplication) GatewaysHandler(w http.ResponseWriter, r *http.Request) {
	if !app.config.UseAllowlist {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte{})
	}
	respondJSON(w, http.StatusOK, app.config.GatewayAllowlist)
}

