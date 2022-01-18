package lightning

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"os"
	"runtime"
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
	ServerHostPort      string
	TlsPath             string
	MacPath             string
	MinConfs            int64
	TargetConfs         int64
	Logger              log.Logger
	LndLogLevel         string
	Testnet             bool
	WalletAddress       string
	WalletPass          string
	WalletSeed          []string
	LastFee             int64
	HashPrice           int64
	SessionSecret       string
	NoMacaroons         bool
	UseChainpointConfig bool
}

var (
	maxMsgRecvSize = grpc.MaxCallRecvMsgSize(1 * 1024 * 1024 * 200)
)

// LoggerError : Log error if it exists using a logger
func (ln *LnClient) LoggerError(err error) error {
	if err != nil {
		ln.Logger.Error(fmt.Sprintf("Error in %s: %s", GetCurrentFuncName(2), err.Error()))
	}
	return err
}

// GetCurrentFuncName : get name of function being called
func GetCurrentFuncName(numCallStack int) string {
	pc, _, _, _ := runtime.Caller(numCallStack)
	return fmt.Sprintf("%s", runtime.FuncForPC(pc).Name())
}

func (ln *LnClient) GetClient() (lnrpc.LightningClient, func(), error) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil, err
	}
	return lnrpc.NewLightningClient(conn), closeIt, nil
}

func (ln *LnClient) GetWalletUnlockerClient() (lnrpc.WalletUnlockerClient, func(), error) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil, err
	}
	return lnrpc.NewWalletUnlockerClient(conn), closeIt, nil
}

func (ln *LnClient) GetWalletClient() (walletrpc.WalletKitClient, func(), error) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil, err
	}
	return walletrpc.NewWalletKitClient(conn), closeIt, nil
}

func (ln *LnClient) GetInvoiceClient() (invoicesrpc.InvoicesClient, func(), error) {
	conn, err := ln.CreateConn()
	closeIt := func() {
		conn.Close()
	}
	if ln.LoggerError(err) != nil {
		return nil, nil, err
	}
	return invoicesrpc.NewInvoicesClient(conn), closeIt, nil
}

func (ln *LnClient) Unlocker() error {
	conn, close, err := ln.GetWalletUnlockerClient()
	defer close()
	if err != nil {
		return err
	}
	unlockReq := lnrpc.UnlockWalletRequest{
		WalletPassword:       []byte(ln.WalletPass),
		RecoveryWindow:       10000,
		ChannelBackups:       nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	_, err = conn.UnlockWallet(context.Background(), &unlockReq)
	if err != nil {
		if strings.Contains(err.Error(), "unknown service lnrpc.WalletUnlocker") {
			return nil
		}
		return ln.LoggerError(err)
	}
	return nil
}

func (ln *LnClient) InitWallet() error {
	conn, close, err := ln.GetWalletUnlockerClient()
	defer close()
	if err != nil {
		return err
	}
	initReq := lnrpc.InitWalletRequest{
		WalletPassword:       []byte(ln.WalletPass),
		CipherSeedMnemonic:   ln.WalletSeed,
		AezeedPassphrase:     nil,
		RecoveryWindow:       10000,
		ChannelBackups:       nil,
		StatelessInit:        false,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	_, err = conn.InitWallet(context.Background(), &initReq)
	if err != nil {
		return ln.LoggerError(err)
	}
	return nil
}

func (ln *LnClient) GenSeed() ([]string, error) {
	conn, close, err := ln.GetWalletUnlockerClient()
	defer close()
	if err != nil {
		return []string{}, err
	}
	seedReq := lnrpc.GenSeedRequest{}
	resp, err := conn.GenSeed(context.Background(), &seedReq)
	if err != nil {
		return []string{}, ln.LoggerError(err)
	}
	return resp.CipherSeedMnemonic, nil
}

func (ln *LnClient) NewAddress() (string, error) {
	conn, close, err := ln.GetClient()
	defer close()
	if ln.LoggerError(err) != nil {
		return "", err
	}
	addrReq := lnrpc.NewAddressRequest{Type: 0}
	resp, err := conn.NewAddress(context.Background(), &addrReq)
	if err != nil {
		return "", ln.LoggerError(err)
	}
	return resp.Address, nil
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
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return nil, err
	}
	defer closeFunc()
	resp, err := client.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	return resp, err
}

func (ln *LnClient) GetWalletBalance() (*lnrpc.WalletBalanceResponse, error) {
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return nil, err
	}
	defer closeFunc()
	resp, err := client.WalletBalance(context.Background(), &lnrpc.WalletBalanceRequest{})
	return resp, err
}

func (ln *LnClient) GetTransaction(id []byte) (lnrpc.TransactionDetails, error) {
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return lnrpc.TransactionDetails{}, err
	}
	defer closeFunc()
	txResponse, err := client.GetTransactions(context.Background(), &lnrpc.GetTransactionsRequest{
		Txid: id,
	})
	if ln.LoggerError(err) != nil {
		return lnrpc.TransactionDetails{}, err
	}
	return *txResponse, nil
}

func (ln *LnClient) GetTransactionFromStr(txid string ) (lnrpc.TransactionDetails, error){
	decodedId, err := hex.DecodeString(txid)
	if err != nil {
		return lnrpc.TransactionDetails{}, err
	}
	return ln.GetTransaction(decodedId)
}

func (ln *LnClient) GetBlockByHeight(height int64) (lnrpc.BlockDetails, error) {
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return lnrpc.BlockDetails{}, err
	}
	defer closeFunc()
	block, err := client.GetBlock(context.Background(), &lnrpc.GetBlockRequest{BlockHeight: uint32(height)})
	if ln.LoggerError(err) != nil {
		return lnrpc.BlockDetails{}, err
	}
	return *block, nil
}

func (ln *LnClient) GetBlockByHash(hash string) (lnrpc.BlockDetails, error) {
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return lnrpc.BlockDetails{}, err
	}
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
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return false, err
	}
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
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return err
	}
	defer closeFunc()
	_, err = client.ConnectPeer(context.Background(), &connectPeer)
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
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return nil, err
	}
	defer closeFunc()
	channels, err := client.ListChannels(context.Background(), &lnrpc.ListChannelsRequest{})
	return channels, err
}

func (ln *LnClient) GetPendingChannels() (*lnrpc.PendingChannelsResponse, error) {
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return nil, err
	}
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
	openSesame.LocalFundingAmount = satVal
	if ln.MinConfs != 0 {
		openSesame.MinConfs = int32(ln.MinConfs)
	}
	if ln.TargetConfs != 0 {
		openSesame.TargetConf = int32(ln.TargetConfs)
	}
	client, closeFunc, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return nil, err
	}
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

	if !ln.NoMacaroons {
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
	}
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

func (ln *LnClient) GetLndFeeEstimate() (int64, error) {
	wallet, closeFunc, err := ln.GetWalletClient()
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

func (ln *LnClient) AnchorData(hash []byte) (string, string, error) {
	b := txscript.NewScriptBuilder()
	b.AddOp(txscript.OP_RETURN)
	b.AddData(hash)
	outputScript, err := b.Script()
	if ln.LoggerError(err) != nil {
		return "", "", err
	}
	wallet, closeFunc, err := ln.GetWalletClient()
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
	if ln.LoggerError(err) != nil {
		return "", "", err
	}
	tx, err := btcutil.NewTxFromBytes(resp.RawTx)
	if ln.LoggerError(err) != nil {
		return "", "", err
	}
	ln.Logger.Info(fmt.Sprintf("Ln SendOutputs Response: %+v", resp))
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
	wallet, closeWalletFunc, err := ln.GetWalletClient()
	if ln.LoggerError(err) != nil {
		return lnrpc.SendCoinsResponse{}, err
	}
	defer closeWalletFunc()
	estimatedFee, err := wallet.EstimateFee(context.Background(), &walletrpc.EstimateFeeRequest{ConfTarget: 2})
	if err != nil {
		return lnrpc.SendCoinsResponse{}, err
	}
	client, closeFunc, err := ln.GetClient()
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
	lightning, close, err := ln.GetClient()
	if ln.LoggerError(err) != nil {
		return lnrpc.Invoice{}, err
	}
	defer close()
	invoice, err := lightning.LookupInvoice(context.Background(), &lnrpc.PaymentHash{RHash: payhash})
	if ln.LoggerError(err) != nil {
		return lnrpc.Invoice{}, err
	}
	return *invoice, nil
}

func (ln *LnClient) ReplaceByFee(txid string, OPRETURNIndex bool, newfee int) (walletrpc.BumpFeeResponse, error) {
	wallet, close, err := ln.GetWalletClient()
	if ln.LoggerError(err) != nil {
		return walletrpc.BumpFeeResponse{}, err
	}
	defer close()
	decodedId, err := hex.DecodeString(txid)
	if err != nil {
		return walletrpc.BumpFeeResponse{}, err
	}
	tx, err := ln.GetTransaction(decodedId)
	if err != nil {
		return walletrpc.BumpFeeResponse{}, err
	}
	if len(tx.Transactions) == 0 {
		return walletrpc.BumpFeeResponse{}, errors.New("no transaction found")
	}
	rawTxHex := tx.GetTransactions()[0].RawTxHex
	decodedTx, err := hex.DecodeString(rawTxHex)
	if err != nil {
		return walletrpc.BumpFeeResponse{}, err
	}
	var msgTx wire.MsgTx
	if ln.LoggerError(msgTx.BtcDecode(bytes.NewReader(decodedTx), 0, wire.WitnessEncoding)); err != nil {
		ln.Logger.Info("RBF Decoding for tx output failed")
		return walletrpc.BumpFeeResponse{}, err
	}
	chainParam := chaincfg.Params{}
	if ln.Testnet {
		chainParam = chaincfg.TestNet3Params
	} else {
		chainParam = chaincfg.MainNetParams
	}
	var outputIndex uint32
	for i, txOut := range msgTx.TxOut {
		script, _, _, err := txscript.ExtractPkScriptAddrs(
			txOut.PkScript, &chainParam,
		)
		ln.Logger.Info(fmt.Sprintf("Extracted script %s from output %d", script.String(), i))
		if ln.LoggerError(err) != nil {
			continue
		}
		if OPRETURNIndex {
			if script.String() == txscript.NullDataTy.String() {
				ln.Logger.Info("Selected OP_RETURN output")
				outputIndex = uint32(i)
				break
			}
		} else {
			if script.String() != txscript.NullDataTy.String() {
				ln.Logger.Info("Selected Payment output")
				outputIndex = uint32(i)
				break
			}
		}
	}
	txIdHash := (msgTx).TxHash()
	txIdBytes := txIdHash.CloneBytes()
	outpoint := lnrpc.OutPoint{
		TxidBytes:            txIdBytes,
		OutputIndex:          outputIndex,
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

func (ln *LnClient) WaitForConnection(d time.Duration) error {
	//Wait for lightning connection
	deadline := time.Now().Add(d)
	for !time.Now().After(deadline) {
		conn, err := ln.CreateConn()
		if err != nil {
			ln.Logger.Error("Waiting on lnd to be ready...")
			time.Sleep(5 * time.Second)
			continue
		} else {
			conn.Close()
			return nil
		}
	}
	return errors.New("Exceeded LND Connection deadline: check that LND has peers")
}

func (ln *LnClient) WaitForMacaroon(d time.Duration) error {
	//Wait for lightning connection
	deadline := time.Now().Add(d)
	for !time.Now().After(deadline) {
		if _, err := os.Stat(ln.MacPath); os.IsNotExist(err) {
			ln.Logger.Error("Waiting on lnd admin to be ready...")
			time.Sleep(5 * time.Second)
			continue
		} else {
			return nil
		}
	}
	return errors.New("Exceeded LND Macaroon deadline: check that LND has peers")
}

func (ln *LnClient) WaitForNewAddress(d time.Duration) (string, error) {
	//Wait for lightning connection
	deadline := time.Now().Add(d)
	for !time.Now().After(deadline) {
		addr, err := ln.NewAddress()
		if err != nil {
			ln.Logger.Error("Waiting on lnd to be ready to give new address...")
			time.Sleep(5 * time.Second)
			continue
		} else {
			return addr, nil
		}
	}
	return "", errors.New("Exceeded LND New Address deadline: check that LND has peers")
}
