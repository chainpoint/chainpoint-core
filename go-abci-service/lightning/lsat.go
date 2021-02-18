package lightning

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"gopkg.in/macaroon.v2"
	"math/rand"
	"net/http"
	"regexp"
)

type LSAT struct {
	ID       TokenID
	Preimage []byte
	PayHash  []byte
	Invoice  string
	Value    int64
	Macaroon macaroon.Macaroon
}

func (ln *LnClient) GenerateHodlLSAT(ip string) (LSAT, error) {
	preimage := make([]byte, 32)
	rand.Read(preimage)
	hash := sha256.Sum256(preimage)
	invoice, closeInvFunc := ln.GetInvoiceClient()
	defer closeInvFunc()
	addInvoiceReq, err := invoice.AddHoldInvoice(context.Background(), &invoicesrpc.AddHoldInvoiceRequest{
		Memo:                 fmt.Sprintf("HODL invoice payment from Chainpoint Core %s", ln.ServerHostPort),
		Hash:                 hash[:],
		Value:                ln.HashPrice,
		ValueMsat:            0,
		DescriptionHash:      nil,
		Expiry:               0,
		FallbackAddr:         "",
		CltvExpiry:           0,
		RouteHints:           nil,
		Private:              false,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	})
	if ln.LoggerError(err) != nil {
		return LSAT{}, err
	}
	tID, err := MakeIDFromString(hex.EncodeToString(preimage))
	if ln.LoggerError(err) != nil {
		return LSAT{}, err
	}
	identifier := Identifier{
		Version:     0,
		PaymentHash: hash,
		TokenID:     tID,
	}
	secBytes, err := hex.DecodeString(ln.SessionSecret)
	if ln.LoggerError(err) != nil {
		return LSAT{}, err
	}
	var buf bytes.Buffer
	EncodeIdentifier(&buf, &identifier)
	mac, err := macaroon.New(secBytes, buf.Bytes(), ip, macaroon.V2)
	if ln.LoggerError(err) != nil {
		return LSAT{}, err
	}
	return LSAT{
		ID:       tID,
		Preimage: preimage,
		PayHash:  hash[:],
		Invoice:  addInvoiceReq.PaymentRequest,
		Value:    ln.HashPrice,
		Macaroon: *mac,
	}, nil
}

func (lsat *LSAT) ToChallenge() string {
	mac, _ := lsat.Macaroon.MarshalBinary()
	macStr := base64.StdEncoding.EncodeToString(mac)
	challenge := fmt.Sprintf("LSAT macaroon=\"%s\", invoice=\"%s\"", macStr, lsat.Invoice)
	return challenge
}

func (lsat *LSAT) ToToken() string {
	mac, _ := lsat.Macaroon.MarshalBinary()
	macStr := base64.StdEncoding.EncodeToString(mac)
	token := fmt.Sprintf("LSAT %s:%s", macStr, base64.StdEncoding.EncodeToString(lsat.Preimage))
	return token
}

func FromChallence(header *http.Header) (LSAT, error) {
	var authHeader string
	authRegex  := regexp.MustCompile("LSAT (.*?):([a-f0-9]{64})")
	HeaderMacaroonMD := "Grpc-Metadata-Macaroon"
	HeaderMacaroon := "Macaroon"
	switch {
	// Header field 1 contains the macaroon and the preimage as distinct
	// values separated by a colon.
	case header.Get("Authorization") != "":
		// Parse the content of the header field and check that it is in
		// the correct format.
		authHeader = header.Get("Authorization")
		if !authRegex.MatchString(authHeader) {
			return LSAT{}, fmt.Errorf("invalid "+
				"auth header format: %s", authHeader)
		}
		matches := authRegex.FindStringSubmatch(authHeader)
		if len(matches) != 3 {
			return LSAT{}, fmt.Errorf("invalid "+
				"auth header format: %s", authHeader)
		}

		// Decode the content of the two parts of the header value.
		macBase64, preimageHex := matches[1], matches[2]
		macBytes, err := base64.StdEncoding.DecodeString(macBase64)
		if err != nil {
			return LSAT{}, fmt.Errorf("base64 "+
				"decode of macaroon failed: %v", err)
		}
		mac := &macaroon.Macaroon{}
		err = mac.UnmarshalBinary(macBytes)
		if err != nil {
			return LSAT{}, fmt.Errorf("unable to "+
				"unmarshal macaroon: %v", err)
		}
		preimage, err := hex.DecodeString(preimageHex)
		if err != nil {
			return LSAT{}, fmt.Errorf("hex "+
				"decode of preimage failed: %v", err)
		}
		tID, err := MakeIDFromString(preimageHex)
		if err != nil {
			return LSAT{}, fmt.Errorf("hex "+
				"decode of preimage into TokenID failed: %v", err)
		}

		hash := sha256.Sum256(preimage)
		// All done, we don't need to extract anything from the
		// macaroon since the preimage was presented separately.
		return LSAT{
			ID:       tID,
			Preimage: preimage[:],
			PayHash:  hash[:],
			Invoice:  "",
			Value:    0,
			Macaroon: *mac,
		}, nil

	// Header field 2: Contains only the macaroon.
	case header.Get(HeaderMacaroonMD) != "":
		authHeader = header.Get(HeaderMacaroonMD)

	// Header field 3: Contains only the macaroon.
	case header.Get(HeaderMacaroon) != "":
		authHeader = header.Get(HeaderMacaroon)

	default:
		return LSAT{}, fmt.Errorf("no auth header " +
			"provided")
	}

	// For case 2 and 3, we need to actually unmarshal the macaroon to
	// extract the preimage.
	macBytes, err := hex.DecodeString(authHeader)
	if err != nil {
		return LSAT{}, fmt.Errorf("hex decode of "+
			"macaroon failed: %v", err)
	}
	mac := &macaroon.Macaroon{}
	err = mac.UnmarshalBinary(macBytes)
	if err != nil {
		return LSAT{}, fmt.Errorf("unable to "+
			"unmarshal macaroon: %v", err)
	}
	preimageHex, ok := HasCaveat(mac, "preimage")
	if !ok {
		return LSAT{}, errors.New("preimage caveat " +
			"not found")
	}
	preimage, err := hex.DecodeString(preimageHex)
	if err != nil {
		return LSAT{}, fmt.Errorf("hex decode of "+
			"preimage failed: %v", err)
	}
	tID, err := MakeIDFromString(preimageHex)
	if err != nil {
		return LSAT{}, fmt.Errorf("hex "+
			"decode of preimage into TokenID failed: %v", err)
	}
	hash := sha256.Sum256(preimage)

	return LSAT{
		ID:       tID,
		Preimage: preimage[:],
		PayHash:  hash[:],
		Invoice:  "",
		Value:    2,
		Macaroon: *mac,
	}, nil}
