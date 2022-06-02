package abci

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/chainpoint/chainpoint-core/leaderelection"
	"github.com/chainpoint/chainpoint-core/proof"
	"github.com/chainpoint/chainpoint-core/types"
	"github.com/chainpoint/chainpoint-core/util"
	"github.com/gorilla/mux"
	"github.com/lightningnetwork/lnd/lnrpc"
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

func (app *AnchorApplication) LnPaymentHandler(quit chan struct{}) {
	for {
		if !app.state.AppReady {
			continue
		}
		hashRegex := regexp.MustCompile("^[a-fA-F0-9]{64}$")
		errors := make(chan error)
		results := make(chan lnrpc.Invoice)
		subscribe := true
		go app.LnClient.SubscribeInvoicesChannel(quit, errors, results)
		for subscribe {
			select {
			case err := <-errors:
				app.LogError(err)
				subscribe = false
				time.Sleep(30 * time.Second)
				break
			case res := <-results:
				app.logger.Info("Received Invoice", "Invoice", hex.EncodeToString(res.RHash), "Keysend", res.IsKeysend)
				if res.IsKeysend && res.State == lnrpc.Invoice_SETTLED && (res.Value == int64(app.config.HashPrice) || res.ValueMsat == int64(app.config.HashPrice)*1000) {
					var hash string
					if len(res.Htlcs) >= 1 {
						for _, htlc := range res.Htlcs {
							for _, value := range htlc.CustomRecords {
								if hashRegex.MatchString(hex.EncodeToString(value)) {
									hash = hex.EncodeToString(value)
								}
							}
						}
					}
					if hashRegex.MatchString(res.Memo) {
						hash = res.Memo
					}
					if hash != "" {
						id := sha256.Sum256([]byte(hash))
						idStr := hex.EncodeToString(id[:])
						app.aggregator.AddHashItem(types.HashItem{
							ProofID: idStr,
							Hash:    hash,
						})
						alternateId := hex.EncodeToString(res.RHash)
						app.aggregator.AddHashItem(types.HashItem{
							ProofID: alternateId,
							Hash:    hash,
						})
						app.logger.Info("Accepted Hash from invoice", "ProofId", idStr, "AlternateProofId", alternateId, "hash", hash)
					}
				}
			}
		}
	}
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

func (app *AnchorApplication) StatusHandler(w http.ResponseWriter, r *http.Request) {
	ip := util.GetClientIP(r)
	app.logger.Info(fmt.Sprintf("Status Client IP: %s", ip))
	status := app.state.TMState
	info, err := app.LnClient.GetInfo()
	if app.LogError(err) != nil {
		errorMessage := map[string]interface{}{"error": "Could not query for status"}
		respondJSON(w, http.StatusInternalServerError, errorMessage)
		return
	}
	balance, err := app.LnClient.GetWalletBalance()
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
	ipStrs := strings.Split(ip, ":")
	if len(ipStrs) == 2 {
		ip = ipStrs[0]
	}
	if app.config.UseAllowlist && util.ArrayContains(app.config.GatewayAllowlist, ip) {
		app.logger.Info("IP allowed access without LSAT")
	} else if app.LnClient.RespondLSAT(w, r) {
		return
	}
	contentType := r.Header.Get("Content-type")
	if contentType != "application/json" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid content type"})
		return
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	hash := Hash{}
	err := d.Decode(&hash)
	if app.LogError(err) != nil || len(hash.Hash) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body: missing hash"})
		return
	}
	match, err := regexp.MatchString("^([a-fA-F0-9]{2}){20,64}$", hash.Hash)
	if app.LogError(err) != nil || !match {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid JSON body: bad hash submitted"})
		return
	}

	proofId, err := app.ULIDGenerator.NewUlid()
	if app.LogError(err) != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "cannot compute ulid"})
		return
	}
	proofIdStr := proofId.String()
	hashResponse := HashResponse{
		Hash:         hash.Hash,
		ProofId:      proofIdStr,
		HashReceived: time.Now().Format(time.RFC3339),
		ProcessingHints: ProcessingHints{
			CalHint: time.Now().Add(140 * time.Second).Format(time.RFC3339),
			BtcHint: time.Now().Add(90 * time.Minute).Format(time.RFC3339),
		},
	}
	go app.Analytics.SendEvent(app.state.LatestTimeRecord, "HashReceived", hashResponse.ProofId, hashResponse.HashReceived, ip, "", ip)
	// Append hash item to aggregator
	app.aggregator.AddHashItem(types.HashItem{Hash: hash.Hash, ProofID: proofIdStr})
	respondJSON(w, http.StatusOK, hashResponse)
}

func (app *AnchorApplication) ProofHandler(w http.ResponseWriter, r *http.Request) {
	ip := util.GetClientIP(r)
	app.logger.Info(fmt.Sprintf("Proof Client IP: %s", ip))
	proofidHeader := r.Header.Get("proofids")
	if len(proofidHeader) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request, at least one hash id required"})
		return
	}
	proofids := strings.Split(strings.ReplaceAll(proofidHeader, " ", ""), ",")
	if len(proofids) > 250 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request, too many hash ids (250 max)"})
		return
	}
	uuidOrUlidRegex := regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})|([0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26})|([a-f0-9]{64})$`)
	for _, id := range proofids {
		if !uuidOrUlidRegex.MatchString(id) {
			errStr := fmt.Sprintf("invalid request, bad proof_id: %s", id)
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": errStr})
			return
		}
	}
	proofStates, err := app.ChainpointDb.GetProofsByProofIds(proofids)
	if app.LogError(err) != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not retrieve proofs"})
		return
	}
	response := make([]proof.P, 0)
	for _, id := range proofids {
		if val, exists := proofStates[id]; exists {
			var rawJSON map[string]interface{}
			if err := json.Unmarshal([]byte(val.Proof), &rawJSON); err != nil {
				response = append(response, map[string]interface{}{"proof_id": id, "proof": nil})
			}
			go app.Analytics.SendEvent(app.state.LatestTimeRecord, "GetProof", id, time.Now().Format(time.RFC3339), ip, "", ip)
			response = append(response, map[string]interface{}{"proof_id": id, "proof": rawJSON})
		} else {
			response = append(response, map[string]interface{}{"proof_id": id, "proof": nil})
		}
	}
	respondJSON(w, http.StatusOK, response)
}

func (app *AnchorApplication) ProofUpgradeHandler(w http.ResponseWriter, r *http.Request) {
	ip := util.GetClientIP(r)
	app.logger.Info(fmt.Sprintf("Proof Upgrade Client IP: %s", ip))
	vars := mux.Vars(r)
	if _, exists := vars["txid"]; exists {
		app.logger.Info("Upgrading proof", "cal", vars["txid"])
		proof, err := app.Anchor.ConstructProof(vars["txid"])
		if app.LogError(err) != nil {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not reconstruct core proof"})
			return
		}
		respondJSON(w, http.StatusOK, proof)
		return
	}
	respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": "txid parameter required"})
}

func (app *AnchorApplication) CalHandler(w http.ResponseWriter, r *http.Request) {
	ip := util.GetClientIP(r)
	app.logger.Info(fmt.Sprintf("Cal Client IP: %s", ip))
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
	ip := util.GetClientIP(r)
	app.logger.Info(fmt.Sprintf("CalData Client IP: %s", ip))
	vars := mux.Vars(r)
	tx := types.Tx{}
	if _, exists := vars["txid"]; exists {
		result, err := app.rpc.GetTxByHash(vars["txid"])
		if err != nil {
			root, err := app.Cache.Get(vars["txid"])
			if app.LogError(err) != nil {
				respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not retrieve tx"})
				return
			}
			if len(root) != 0 {
				tx.Data = root
			} else {
				respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not retrieve tx from cache"})
				return
			}
		} else {
			tx, err = util.DecodeTx(result.Tx)
			if app.LogError(err) != nil {
				respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "could not decode tx"})
				return
			}
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(tx.Data))
		return
	}
	respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": "tx parameter required"})
}

func (app *AnchorApplication) PeerHandler(w http.ResponseWriter, r *http.Request) {
	//ip := util.GetClientIP(r)
	//app.logger.Info(fmt.Sprintf("Peers Client IP: %s", ip))
	peers := leaderelection.GetPeers(*app.state, app.state.TMState, app.state.TMNetInfo)
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
		uriParts := strings.Split(selfIp, ":")
		selfIp = uriParts[0]
		peerList = append(peerList, selfIp)
	}
	peerList = util.UniquifyStrings(peerList)
	respondJSON(w, http.StatusOK, peerList)
}

func (app *AnchorApplication) GatewaysHandler(w http.ResponseWriter, r *http.Request) {
	//ip := util.GetClientIP(r)
	//app.logger.Info(fmt.Sprintf("Gateways Client IP: %s", ip))
	if !app.config.UseAllowlist {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte{})
	}
	respondJSON(w, http.StatusOK, app.config.GatewayAllowlist)
}
