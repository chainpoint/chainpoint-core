package abci

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/blake2s"
	"github.com/chainpoint/chainpoint-core/go-abci-service/lightning"
	"github.com/chainpoint/chainpoint-core/go-abci-service/proof"
	"github.com/chainpoint/chainpoint-core/go-abci-service/types"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"github.com/chainpoint/chainpoint-core/go-abci-service/uuid"
	"github.com/gorilla/mux"
	lnrpc2 "github.com/lightningnetwork/lnd/lnrpc"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Hash struct {
	Hash string `json:"hash`
}

type HashResponse struct {
	Hash            string          `json:"hash"`
	ProofId         string          `json:"proof_id"`
	HashReceived    string          `json:"hash_received"`
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
	if util.LogError(err) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

func (app *AnchorApplication) respondLSAT(w http.ResponseWriter, r *http.Request) bool {
	authorization := r.Header.Get("Authorization")
	if len(authorization) == 0 {
		lsat, err := app.lnClient.GenerateHodlLSAT(util.GetClientIP(r))
		if app.LogError(err) != nil {
			errorMessage := map[string]interface{}{"error": "Could not generate LSAT"}
			respondJSON(w, http.StatusInternalServerError, errorMessage)
		}
		challenge := lsat.ToChallenge()
		app.logger.Info(fmt.Sprintf("LSAT toChallenge: %s", challenge))
		w.Header().Set("www-authenticate", challenge)
		errorMessage := map[string]interface{}{"error": "Could not generate LSAT"}
		respondJSON(w, http.StatusPaymentRequired, errorMessage)
		return true
	} else {
		lsat, err := lightning.FromChallence(&r.Header)
		app.logger.Info(fmt.Sprintf("LSAT fromChallenge: %#v", lsat))
		if app.LogError(err) != nil {
			errorMessage := map[string]interface{}{"error": "Invalid LSAT provided in Authorization header"}
			respondJSON(w, http.StatusInternalServerError, errorMessage)
			return true
		}
		invoice, err := app.lnClient.LookupInvoice(lsat.PayHash)
		if app.LogError(err) != nil {
			errorMessage := map[string]interface{}{"error": fmt.Sprintf("No matching invoice found for payhash %s", lsat.PayHash)}
			respondJSON(w, http.StatusNotFound, errorMessage)
			return true
		}
		switch invoice.State {
		case lnrpc2.Invoice_SETTLED:
			errorMessage := map[string]interface{}{"error": "Unauthorized: Invoice has already been settled. Try again with a different LSAT"}
			respondJSON(w, http.StatusUnauthorized, errorMessage)
			return true
		case lnrpc2.Invoice_OPEN:
			lsat.Invoice = invoice.PaymentRequest
			challenge := lsat.ToChallenge()
			app.logger.Info(fmt.Sprintf("LSAT toChallenge Invoice Open: %s", challenge))
			w.Header().Set("www-authenticate", challenge)
			errorMessage := map[string]interface{}{"message": "Payment Required"}
			respondJSON(w, http.StatusPaymentRequired, errorMessage)
			return true
		case lnrpc2.Invoice_CANCELED:
			errorMessage := map[string]interface{}{"error": "Unauthorized: Invoice has been cancelled. Try again with a different LSAT"}
			respondJSON(w, http.StatusUnauthorized, errorMessage)
			return true
		default:
			errorMessage := map[string]interface{}{"error": "Unauthorized: Invoice has expired or been canceled. Try again with a different LSAT"}
			respondJSON(w, http.StatusUnauthorized, errorMessage)
			return true
		}
		token := lsat.ToToken()
		app.logger.Info(fmt.Sprintf("LSAT toToken: %s", token))
		w.Header().Set("authorization", token)
		return false // don't exit top level handler
	}

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
		Version:             "0.0.2",
		Time:                time.Now().UTC().Format("2006-01-02T15:04:05.999Z07:00"),
		BaseURI:             util.GetEnv("CHAINPOINT_CORE_BASE_URI", "http://0.0.0.0"),
		Network:             app.config.BitcoinNetwork,
		IdentityPubkey:      info.IdentityPubkey,
		LightningAddress:    app.config.LightningConfig.WalletAddress,
		Uris:                info.Uris,
		ActiveChannelsCount: int(info.NumActiveChannels),
		Alias:               info.Alias,
		HashPriceSatoshis:   app.config.HashPrice,
	}
	apiStatus.LightningBalance.UnconfirmedBalance = strconv.FormatInt(balance.UnconfirmedBalance, 10)
	apiStatus.LightningBalance.ConfirmedBalance = strconv.FormatInt(balance.ConfirmedBalance, 10)
	apiStatus.LightningBalance.TotalBalance = strconv.FormatInt(balance.TotalBalance, 10)
	apiStatus.TotalStakePrice = app.state.LnStakePrice
	apiStatus.ValidatorStakePrice = app.state.LnStakePerVal
	apiStatus.Jwk = app.JWK
	apiStatus.NodeInfo = status.NodeInfo
	apiStatus.ValidatorInfo = status.ValidatorInfo
	apiStatus.SyncInfo = status.SyncInfo
	respondJSON(w, http.StatusOK, apiStatus)
}

func (app *AnchorApplication) HashHandler(w http.ResponseWriter, r *http.Request) {
	ip := util.GetClientIP(r)
	app.logger.Info(fmt.Sprintf("Client IP: %s", ip))
	if !(app.config.UseAllowlist && util.ArrayContains(app.config.GatewayAllowlist, ip)){
		if app.respondLSAT(w, r){
			//TODO lsat validation
			return
		}
	} else {
		app.logger.Info("IP allowed access without LSAT")
	}
	contentType := r.Header.Get("Content-type")
	if contentType != "application/json" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid content type"})
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	hash := Hash{}
	err := d.Decode(&hash)
	app.logger.Info(fmt.Sprintf("Received hash %s", hash.Hash))
	if app.LogError(err) != nil || len(hash.Hash) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body: missing hash"})
	}
	match, err := regexp.MatchString("^([a-fA-F0-9]{2}){20,64}$", hash.Hash)
	if app.LogError(err) != nil || !match {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body: bad hash submitted"})
	}

	// compute uuid using blake2s
	t := time.Now()
	unixTimeMS := strconv.FormatInt(t.UnixNano()/int64(time.Millisecond), 10)
	timeLength := strconv.Itoa(len(unixTimeMS))
	hashStr := strings.Join([]string{unixTimeMS, timeLength, hash.Hash, strconv.Itoa(len(hash.Hash))}, ":")
	blakeHash, err := blake2s.New256WithPersonalization(nil, []byte("CHAINPNT"))
	blakeHash.Write([]byte(hashStr))
	blakeHashSum := blakeHash.Sum([]byte{})
	truncHashSum := blakeHashSum[len(blakeHashSum)-5:]
	if app.LogError(err) != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cannot compute blake2s hash"})
	}
	node := append([]byte{0x01}, truncHashSum...)
	app.logger.Info(fmt.Sprintf("hashStr is %s, hash bytes are %s", hashStr, hex.EncodeToString(node)))
	uuid := uuid.UUIDFromTimeNode(t, node)
	if app.LogError(err) != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cannot compute uuid"})
	}
	proofId := uuid.String()
	hashResponse := HashResponse{
		Hash:         hash.Hash,
		ProofId:      proofId,
		HashReceived: time.Now().Format(time.RFC3339),
		ProcessingHints: ProcessingHints{
			CalHint: time.Now().Add(140 * time.Second).Format(time.RFC3339),
			BtcHint: time.Now().Add(90 * time.Minute).Format(time.RFC3339),
		},
	}
	// Add hash item to aggregator
	app.aggregator.AddHashItem(types.HashItem{Hash: hash.Hash, ProofID: proofId})
	respondJSON(w, http.StatusOK, hashResponse)
}

func (app *AnchorApplication) ProofHandler(w http.ResponseWriter, r *http.Request) {
	proofidHeader := r.Header.Get("proofids")
	if len(proofidHeader) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request, at least one hash id required"})
	}
	proofids := strings.Split(strings.ReplaceAll(proofidHeader, " ", ""), ",")
	if len(proofids) > 250 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request, too many hash ids (250 max)"})
	}
	for _, id := range proofids {
		_, err := uuid.ParseUUID(id)
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
			var rawJSON map[string]interface{}
			if err := json.Unmarshal([]byte(val.Proof), &rawJSON); err != nil {
				response = append(response, map[string]interface{}{"proof_id": id, "proof": nil})
			}
			response = append(response, map[string]interface{}{"proof_id": id, "proof": rawJSON})
		} else {
			response = append(response, map[string]interface{}{"proof_id": id, "proof": nil})
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
			return
		}
		respondJSON(w, http.StatusOK, result)
		return
	}
	respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": "tx parameter required"})
}

func (app *AnchorApplication) CalDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, exists := vars["txid"]; exists {
		result, err := app.rpc.GetTxByHash(vars["txid"])
		if app.LogError(err) != nil {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not retrieve tx"})
			return
		}
		tx, err := util.DecodeTx(result.Tx)
		if app.LogError(err) != nil {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not decode tx"})
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(tx.Data))
		return
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
		firstOctet := ip[0:strings.Index(ip, ".")]
		privateRanges := map[string]bool{
			"0":   true,
			"10":  true,
			"172": true,
			"192": true,
			"127": true,
		}
		if _, exists := privateRanges[firstOctet]; exists {
			listenAddr := peer.NodeInfo.ListenAddr
			if strings.Contains(listenAddr, "//") {
				finalIp = listenAddr[strings.LastIndex(listenAddr, "/")+1 : strings.LastIndex(listenAddr, ":")]
			}
			finalIp = listenAddr[0:strings.LastIndex(listenAddr, ":")]
		} else {
			finalIp = ip
		}
		peerList = append(peerList, finalIp)
	}
	if len(app.config.CoreURI) != 0 {
		selfIp := strings.ReplaceAll(app.config.CoreURI, "http://", "")
		peerList = append(peerList, selfIp)
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
