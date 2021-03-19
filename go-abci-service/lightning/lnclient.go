package lightning

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"net/http"
	"time"

	/*	"github.com/btcsuite/btcd/chaincfg"
		"github.com/btcsuite/btcwallet/wallet/txrules"
		"github.com/btcsuite/btcwallet/wallet/txsizes"*/
	"io/ioutil"
	"net"
	"strings"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"

	"github.com/lightningnetwork/lnd/lnrpc/signrpc"

	"github.com/btcsuite/btcd/txscript"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/lightningnetwork/lnd/lnrpc"

	"github.com/lightningnetwork/lnd/lncfg"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon "gopkg.in/macaroon.v2"
)

type LnClient struct {
	ServerHostPort string
	TlsPath        string
	MacPath        string
	MinConfs       int64
	TargetConfs    int64
	LocalSats      int64
	PushSats       int64
	Logger         log.Logger
	Testnet        bool
	WalletAddress  string
	WalletPass     string
	FeeMultiplier  float64
	LastFee        int64
	HashPrice      int64
	SessionSecret  string
}



// BitcoinerFee : estimates fee from bitcoiner service
type BitcoinerFee struct {
	Timestamp int `json:"timestamp"`
	Estimates struct {
		Num30 struct {
			SatPerVbyte float64 `json:"sat_per_vbyte"`
			Total       struct {
				P2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2wpkh"`
				P2ShP2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2sh-p2wpkh"`
				P2Pkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2pkh"`
			} `json:"total"`
		} `json:"30"`
		Num60 struct {
			SatPerVbyte float64 `json:"sat_per_vbyte"`
			Total       struct {
				P2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2wpkh"`
				P2ShP2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2sh-p2wpkh"`
				P2Pkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2pkh"`
			} `json:"total"`
		} `json:"60"`
		Num120 struct {
			SatPerVbyte float64 `json:"sat_per_vbyte"`
			Total       struct {
				P2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2wpkh"`
				P2ShP2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2sh-p2wpkh"`
				P2Pkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2pkh"`
			} `json:"total"`
		} `json:"120"`
		Num180 struct {
			SatPerVbyte float64 `json:"sat_per_vbyte"`
			Total       struct {
				P2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2wpkh"`
				P2ShP2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2sh-p2wpkh"`
				P2Pkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2pkh"`
			} `json:"total"`
		} `json:"180"`
		Num360 struct {
			SatPerVbyte float64 `json:"sat_per_vbyte"`
			Total       struct {
				P2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2wpkh"`
				P2ShP2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2sh-p2wpkh"`
				P2Pkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2pkh"`
			} `json:"total"`
		} `json:"360"`
		Num720 struct {
			SatPerVbyte float64 `json:"sat_per_vbyte"`
			Total       struct {
				P2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2wpkh"`
				P2ShP2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2sh-p2wpkh"`
				P2Pkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2pkh"`
			} `json:"total"`
		} `json:"720"`
		Num1440 struct {
			SatPerVbyte float64 `json:"sat_per_vbyte"`
			Total       struct {
				P2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2wpkh"`
				P2ShP2Wpkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2sh-p2wpkh"`
				P2Pkh struct {
					Usd     float64 `json:"usd"`
					Satoshi float64 `json:"satoshi"`
				} `json:"p2pkh"`
			} `json:"total"`
		} `json:"1440"`
	} `json:"estimates"`
}

var (
	maxMsgRecvSize = grpc.MaxCallRecvMsgSize(1 * 1024 * 1024 * 200)
)

// LoggerError : Log error if it exists using a logger
func (ln *LnClient) LoggerError(err error) error {
	if err != nil {
		ln.Logger.Error(fmt.Sprintf("Error: %s", err.Error()))
	}
	return err
}

func (ln *LnClient) GetClient() (lnrpc.LightningClient, func()) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil
	}
	return lnrpc.NewLightningClient(conn), closeIt
}

func (ln *LnClient) GetWalletUnlockerClient() (lnrpc.WalletUnlockerClient, func()) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil
	}
	return lnrpc.NewWalletUnlockerClient(conn), closeIt
}

func (ln *LnClient) GetWalletClient() (walletrpc.WalletKitClient, func()) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil
	}
	return walletrpc.NewWalletKitClient(conn), closeIt
}

func (ln *LnClient) GetInvoiceClient() (invoicesrpc.InvoicesClient, func()) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil
	}
	return invoicesrpc.NewInvoicesClient(conn), closeIt
}

func (ln *LnClient) Unlocker() error {
	conn, close := ln.GetWalletUnlockerClient()
	if conn == nil {
		return errors.New("unable to obtain client")
	}
	unlockReq := lnrpc.UnlockWalletRequest{
		WalletPassword:       []byte(ln.WalletPass),
		RecoveryWindow:       10000,
		ChannelBackups:       nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	_, err := conn.UnlockWallet(context.Background(), &unlockReq)
	if err != nil {
		if strings.Contains(err.Error(), "unknown service lnrpc.WalletUnlocker") {
			return nil
		}
		return ln.LoggerError(err)
	}
	close()
	return nil
}

func CreateClient(serverHostPort string, tlsPath string, macPath string) LnClient {
	return LnClient{
		ServerHostPort: serverHostPort,
		TlsPath:        tlsPath,
		MacPath:        macPath,
	}
}

func IsLnUri(uri string) bool {
	peerParts := strings.Split(uri, "@")
	if len(peerParts) != 2 {
		return false
	}
	if _, err := hex.DecodeString(peerParts[0]); err != nil {
		return false
	}
	if _, _, err := net.SplitHostPort(peerParts[1]); err != nil {
		return false
	}
	return true
}

func GetIpFromUri(uri string) string {
	peerParts := strings.Split(uri, "@")
	if len(peerParts) != 2 {
		return ""
	}
	addrPort := peerParts[1]
	ipArr := strings.Split(addrPort, ":")
	if len(ipArr) != 2 {
		return ""
	}
	return ipArr[0]
}

func (ln *LnClient) GetInfo() (*lnrpc.GetInfoResponse, error) {
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	resp, err := client.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	return resp, err
}

func (ln *LnClient) GetWalletBalance() (*lnrpc.WalletBalanceResponse, error) {
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	resp, err := client.WalletBalance(context.Background(), &lnrpc.WalletBalanceRequest{})
	return resp, err
}

func (ln *LnClient) GetTransaction(id []byte) (lnrpc.TransactionDetails, error) {
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	txResponse, err := client.GetTransactions(context.Background(), &lnrpc.GetTransactionsRequest{
		Txid: id,
	})
	if ln.LoggerError(err) != nil {
		return lnrpc.TransactionDetails{}, err
	}
	return *txResponse, nil
}

func (ln *LnClient) GetBlockByHeight(height int64) (lnrpc.BlockDetails, error) {
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	block, err := client.GetBlock(context.Background(), &lnrpc.GetBlockRequest{BlockHeight: uint32(height)})
	if ln.LoggerError(err) != nil {
		return lnrpc.BlockDetails{}, err
	}
	return *block, nil
}

func (ln *LnClient) GetBlockByHash(hash string) (lnrpc.BlockDetails, error) {
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	block, err := client.GetBlock(context.Background(), &lnrpc.GetBlockRequest{BlockHash: hash})
	if ln.LoggerError(err) != nil {
		return lnrpc.BlockDetails{}, err
	}
	return *block, nil
}

func (ln *LnClient) PeerExists(peer string) (bool, error) {
	peerParts := strings.Split(peer, "@")
	if len(peerParts) != 2 {
		return false, errors.New("Malformed peer string (must be pubKey@host)")
	}
	pubKey := peerParts[0]
	addr := peerParts[1]
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	peers, err := client.ListPeers(context.Background(), &lnrpc.ListPeersRequest{})
	if err != nil {
		return false, err
	}
	for _, peer := range peers.Peers {
		if peer.PubKey == pubKey && peer.Address == addr {
			return true, nil
		}
	}
	return false, nil
}

func (ln *LnClient) AddPeer(peer string) error {
	peerParts := strings.Split(peer, "@")
	if len(peerParts) != 2 {
		return errors.New("Malformed peer string (must be pubKey@host)")
	}
	peerAddr := lnrpc.LightningAddress{
		Pubkey: peerParts[0],
		Host:   peerParts[1],
	}
	connectPeer := lnrpc.ConnectPeerRequest{
		Addr: &peerAddr,
	}
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	_, err := client.ConnectPeer(context.Background(), &connectPeer)
	if err != nil {
		return err
	}
	return nil
}

func (ln *LnClient) ChannelExists(peer string, satVal int64) (bool, error) {
	peerParts := strings.Split(peer, "@")
	if len(peerParts) != 2 {
		return false, errors.New("Malformed peer string (must be pubKey@host)")
	}
	remotePubkey := peerParts[0]
	channels, err := ln.GetChannels()
	if ln.LoggerError(err) != nil {
		return false, err
	}
	for _, chann := range channels.Channels {
		if chann.RemotePubkey == remotePubkey {
			ln.Logger.Info("Channel found")
			if chann.Capacity >= satVal {
				ln.Logger.Info("Funding is correct value ", "Capacity", chann.Capacity)
				return true, nil
			}
		}
	}
	pending, err := ln.GetPendingChannels()
	if ln.LoggerError(err) != nil {
		return false, err
	}
	for _, chann := range pending.PendingOpenChannels {
		if chann.Channel.RemoteNodePub == remotePubkey {
			ln.Logger.Info("Pending Channel found")
			if chann.Channel.Capacity >= satVal {
				ln.Logger.Info("Funding is correct value ", "Capacity", chann.Channel.Capacity)
				return true, nil
			}
		}
	}
	return false, nil
}

func (ln *LnClient) OurChannelOpenAndFunded(peer string, satVal int64) (bool, error) {
	peerParts := strings.Split(peer, "@")
	if len(peerParts) != 2 {
		return false, errors.New("Malformed peer string (must be pubKey@host)")
	}
	remotePubkey := peerParts[0]
	channels, err := ln.GetChannels()
	if ln.LoggerError(err) != nil {
		return false, err
	}
	for _, chann := range channels.Channels {
		if chann.RemotePubkey == remotePubkey {
			ln.Logger.Info("Channel found")
			if chann.Capacity >= satVal {
				ln.Logger.Info("Funding is correct value ", "Capacity", chann.Capacity)
				return true, nil
			}
		}
	}
	return false, nil
}

func (ln *LnClient) RemoteChannelOpenAndFunded(peer string, satVal int64) (bool, error) {
	peerParts := strings.Split(peer, "@")
	if len(peerParts) != 2 {
		return false, errors.New("Malformed peer string (must be pubKey@host)")
	}
	remotePubkey := peerParts[0]
	channels, err := ln.GetChannels()
	if ln.LoggerError(err) != nil {
		return false, err
	}
	for _, chann := range channels.Channels {
		if chann.RemotePubkey == remotePubkey {
			ln.Logger.Info("Channel found")
			if chann.Capacity >= satVal {
				ln.Logger.Info("Funding is correct value ", "Capacity", chann.Capacity)
				return true, nil
			}
		}
	}
	return false, nil
}

func (ln *LnClient) GetChannels() (*lnrpc.ListChannelsResponse, error) {
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	channels, err := client.ListChannels(context.Background(), &lnrpc.ListChannelsRequest{})
	return channels, err
}

func (ln *LnClient) GetPendingChannels() (*lnrpc.PendingChannelsResponse, error) {
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	channels, err := client.PendingChannels(context.Background(), &lnrpc.PendingChannelsRequest{})
	return channels, err
}

func (ln *LnClient) CreateChannel(peer string, satVal int64) (lnrpc.Lightning_OpenChannelClient, error) {
	peerParts := strings.Split(peer, "@")
	if len(peerParts) != 2 {
		return nil, errors.New("Malformed peer string (must be pubKey@host)")
	}
	pubKey, err := hex.DecodeString(peerParts[0])
	if ln.LoggerError(err) != nil {
		return nil, err
	}
	openSesame := lnrpc.OpenChannelRequest{
		NodePubkey: pubKey,
	}
	if ln.LocalSats != 0 {
		openSesame.LocalFundingAmount = satVal
	}
	if ln.PushSats != 0 {
		openSesame.PushSat = ln.PushSats
	}
	if ln.MinConfs != 0 {
		openSesame.MinConfs = int32(ln.MinConfs)
	}
	if ln.TargetConfs != 0 {
		openSesame.TargetConf = int32(ln.TargetConfs)
	}
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	resp, err := client.OpenChannel(context.Background(), &openSesame)
	if ln.LoggerError(err) != nil {
		return nil, err
	}
	return resp, nil
}

func (ln *LnClient) CreateConn() (*grpc.ClientConn, error) {
	// Load the specified TLS certificate and build transport credentials
	// with it.
	creds, err := credentials.NewClientTLSFromFile(ln.TlsPath, "")
	if ln.LoggerError(err) != nil {
		return nil, err
	}

	// Create a dial options array.
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	macBytes, err := ioutil.ReadFile(ln.MacPath)
	if ln.LoggerError(err) != nil {
		return nil, err
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		return nil, err
	}

	macConstraints := []macaroons.Constraint{
		macaroons.TimeoutConstraint(60),
	}

	// Apply constraints to the macaroon.
	constrainedMac, err := macaroons.AddConstraints(mac, macConstraints...)
	if ln.LoggerError(err) != nil {
		return nil, err
	}

	// Now we append the macaroon credentials to the dial options.
	cred := macaroons.NewMacaroonCredential(constrainedMac)
	opts = append(opts, grpc.WithPerRPCCredentials(cred))

	// We need to use a custom dialer so we can also connect to unix sockets
	// and not just TCP addresses.
	hostPortArr := strings.Split(ln.ServerHostPort, ":")
	defaultRPCPort := "10009"
	if len(hostPortArr) > 1 {
		defaultRPCPort = hostPortArr[1]
	}
	genericDialer := lncfg.ClientAddressDialer(defaultRPCPort)
	opts = append(opts, grpc.WithContextDialer(genericDialer))
	opts = append(opts, grpc.WithDefaultCallOptions(maxMsgRecvSize))

	conn, err := grpc.Dial(ln.ServerHostPort, opts...)
	if ln.LoggerError(err) != nil {
		return nil, err
	}
	return conn, nil
}

func (ln *LnClient) feeSatByteToWeight() int64 {
	return int64(ln.LastFee * 1000 / blockchain.WitnessScaleFactor)
}

// GetThirdPartyFeeEstimate : get sat/vbyte fee and convert to sat/kw
func (ln *LnClient) GetThirdPartyFeeEstimate() (int64, error) {
	var httpClient = &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Get("https://bitcoiner.live/api/fees/estimates/latest")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	fee := BitcoinerFee{}
	err = json.NewDecoder(resp.Body).Decode(&fee)
	if err != nil {
		return 0, err
	}
	return int64(int64(fee.Estimates.Num30.SatPerVbyte) * 1000 / blockchain.WitnessScaleFactor), nil
}

func (ln *LnClient) GetLndFeeEstimate() (int64, error) {
	wallet, closeFunc := ln.GetWalletClient()
	defer closeFunc()
	fee, err := wallet.EstimateFee(context.Background(), &walletrpc.EstimateFeeRequest{ConfTarget: 2})
	if err != nil {
		return 0, err
	}
	if fee.SatPerKw == 12500 {
		return fee.SatPerKw, errors.New("static fee has been returned")
	}
	return fee.SatPerKw, nil
}

func (ln *LnClient) SendOpReturn(hash []byte) (string, string, error) {
	b := txscript.NewScriptBuilder()
	b.AddOp(txscript.OP_RETURN)
	b.AddData(hash)
	outputScript, err := b.Script()
	if ln.LoggerError(err) != nil {
		return "", "", err
	}
	wallet, closeFunc := ln.GetWalletClient()
	defer closeFunc()
	ln.Logger.Info("Ln Wallet client created")
	outputs := []*signrpc.TxOut{
		{
			Value:    0,
			PkScript: outputScript,
		},
	}
	ln.Logger.Info(fmt.Sprintf("Sending Outputs: %v", outputs))
	ln.Logger.Info(fmt.Sprintf("Anchoring with FEE: %d", ln.LastFee))
	outputRequest := walletrpc.SendOutputsRequest{SatPerKw: ln.LastFee, Outputs: outputs}
	resp, err := wallet.SendOutputs(context.Background(), &outputRequest)
	ln.Logger.Info(fmt.Sprintf("Ln SendOutputs Response: %v", resp))
	if ln.LoggerError(err) != nil {
		return "", "", err
	}
	tx, err := btcutil.NewTxFromBytes(resp.RawTx)
	if ln.LoggerError(err) != nil {
		return "", "", err
	}
	var msgTx wire.MsgTx
	if ln.LoggerError(msgTx.BtcDecode(bytes.NewReader(resp.RawTx), 0, wire.WitnessEncoding)); err != nil {
		return "", "", err
	}
	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSizeStripped()))
	if ln.LoggerError(msgTx.SerializeNoWitness(buf)); err != nil {
		return "", "", err
	}
	return tx.Hash().String(), hex.EncodeToString(buf.Bytes()), nil
}

func (ln *LnClient) SendCoins(addr string, amt int64, confs int32) (lnrpc.SendCoinsResponse, error) {
	wallet, closeWalletFunc := ln.GetWalletClient()
	defer closeWalletFunc()
	estimatedFee, err := wallet.EstimateFee(context.Background(), &walletrpc.EstimateFeeRequest{ConfTarget: 2})
	if err != nil {
		return lnrpc.SendCoinsResponse{}, err
	}
	client, closeFunc := ln.GetClient()
	defer closeFunc()
	sendCoinsReq := lnrpc.SendCoinsRequest{
		Addr:       addr,
		Amount:     amt,
		TargetConf: confs,
		SatPerByte: estimatedFee.SatPerKw,
	}
	resp, err := client.SendCoins(context.Background(), &sendCoinsReq)
	ln.LoggerError(err)
	return *resp, err
}

func (ln *LnClient) LookupInvoice(payhash []byte) (lnrpc.Invoice, error) {
	lightning, close := ln.GetClient()
	defer close()
	invoice, err := lightning.LookupInvoice(context.Background(), &lnrpc.PaymentHash{RHash:payhash})
	if ln.LoggerError(err) != nil {
		return lnrpc.Invoice{}, err
	}
	return *invoice, nil
}

func (ln *LnClient) ReplaceByFee(txid []byte, txstr string, output int, newfee int) (walletrpc.BumpFeeResponse, error) {
	wallet, close := ln.GetWalletClient()
	defer close()
	outpoint := lnrpc.OutPoint{
		TxidBytes:            txid,
		TxidStr:              txstr,
		OutputIndex:          uint32(output),
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	rbfReq := walletrpc.BumpFeeRequest{
		Outpoint:             &outpoint,
		TargetConf:           0,
		SatPerByte:           uint32(newfee),
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	resp, err := wallet.BumpFee(context.Background(), &rbfReq)
	if ln.LoggerError(err) != nil {
		return walletrpc.BumpFeeResponse{}, err
	}
	return *resp, nil
}