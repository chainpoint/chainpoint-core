package lightning

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"strings"

	"github.com/btcsuite/btcutil"

	"github.com/lightningnetwork/lnd/lnrpc/signrpc"

	"github.com/btcsuite/btcd/txscript"
	"github.com/lightningnetwork/lnd/lnrpc/walletrpc"

	"github.com/lightningnetwork/lnd/lnrpc"

	"github.com/jacohend/lnd/lncfg"
	"github.com/jacohend/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon "gopkg.in/macaroon.v2"
)

type LnClient struct {
	ServerHostPort string
	TlsPath        string
	MacPath        string
	Conn           *grpc.ClientConn
}

var (
	maxMsgRecvSize = grpc.MaxCallRecvMsgSize(1 * 1024 * 1024 * 200)
)

func (ln *LnClient) GetClient() lnrpc.LightningClient {
	return lnrpc.NewLightningClient(ln.Conn)
}

func (ln *LnClient) GetWalletUnlockerClient() lnrpc.WalletUnlockerClient {
	return lnrpc.NewWalletUnlockerClient(ln.Conn)
}

func (ln *LnClient) GetWalletClient() walletrpc.WalletKitClient {
	return walletrpc.NewWalletKitClient(ln.Conn)
}

func CreateClient(serverHostPort string, tlsPath string, macPath string) LnClient {
	return LnClient{
		ServerHostPort: serverHostPort,
		TlsPath:        tlsPath,
		MacPath:        macPath,
	}
}

func (ln *LnClient) CreateConn() error {
	// Load the specified TLS certificate and build transport credentials
	// with it.
	creds, err := credentials.NewClientTLSFromFile(ln.TlsPath, "")
	if err != nil {
		return err
	}

	// Create a dial options array.
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	macBytes, err := ioutil.ReadFile(ln.MacPath)
	if err != nil {
		return err
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		return err
	}

	macConstraints := []macaroons.Constraint{
		macaroons.TimeoutConstraint(60),
	}

	// Apply constraints to the macaroon.
	constrainedMac, err := macaroons.AddConstraints(mac, macConstraints...)
	if err != nil {
		return err
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
	opts = append(opts, grpc.WithDialer(genericDialer))
	opts = append(opts, grpc.WithDefaultCallOptions(maxMsgRecvSize))

	conn, err := grpc.Dial(ln.ServerHostPort, opts...)
	if err != nil {
		return err
	}
	ln.Conn = conn
	return nil
}

func (ln *LnClient) SendOpReturn(hash []byte) (string, string, error) {
	ln.CreateConn()
	defer ln.Conn.Close()
	b := txscript.NewScriptBuilder()
	b.AddOp(txscript.OP_RETURN)
	b.AddData(hash)
	outputScript, err := b.Script()
	if err != nil {
		return "", "", err
	}
	wallet := ln.GetWalletClient()
	estimatedFee, err := wallet.EstimateFee(context.Background(), &walletrpc.EstimateFeeRequest{ConfTarget: 2})
	opReturnOutput := []*signrpc.TxOut{&signrpc.TxOut{
		Value:    0,
		PkScript: outputScript,
	}}
	outputRequest := walletrpc.SendOutputsRequest{SatPerKw: estimatedFee.SatPerKw, Outputs: opReturnOutput}
	resp, err := wallet.SendOutputs(context.Background(), &outputRequest)
	if err != nil {
		return "", "", err
	}
	tx, err := btcutil.NewTxFromBytes(resp.RawTx)
	if err != nil {
		return "", "", err
	}
	return tx.Hash().String(), hex.EncodeToString(resp.RawTx), nil
}
