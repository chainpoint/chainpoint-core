package lightning

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc/invoicesrpc"
	"gopkg.in/macaroon.v2"
	"net/http"
	"strings"
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
	preimage, err := GenerateRandomBytes(32)
	if ln.LoggerError(err) != nil {
		return LSAT{}, err
	}
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
	mac, err := macaroon.New(secBytes, buf.Bytes(), "lsat", macaroon.LatestVersion)
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

func FromHeader(header *http.Header) (LSAT, error) {
	var authHeader string
	HeaderMacaroonMD := "Grpc-Metadata-Macaroon"
	HeaderMacaroon := "Macaroon"
	switch {
	// Header field 1 contains the macaroon and the preimage as distinct
	// values separated by a colon.
	case header.Get("Authorization") != "":
		// Parse the content of the header field and check that it is in
		// the correct format.
		var macBase64 string
		var preimageHex string
		authHeader = header.Get("Authorization")
		lsatAuth := strings.Split(authHeader, " ")
		if len(lsatAuth) != 2 {
			return LSAT{}, fmt.Errorf("invalid "+
				"auth header format: %s", authHeader)
		}
		content := strings.Split(lsatAuth[1], ":")
		if len(content) == 2 && content[0] != "" {
			macBase64 = content[0]
		}
		if len(content) == 2 && content[1] != "" {
			preimageHex = content[1]
		}
		macBytes, err := base64.StdEncoding.DecodeString(macBase64)
		if err != nil {
			return LSAT{}, fmt.Errorf("base64 "+
				"decode of macaroon failed: %v\nrequest: %s", err, authHeader)
		}
		mac := &macaroon.Macaroon{}
		err = mac.UnmarshalBinary(macBytes)
		buf := bytes.NewReader(mac.Id())
		id, err := DecodeIdentifier(buf)
		if err != nil {
			return LSAT{}, fmt.Errorf("unable to "+
				"unmarshal macaroon: %v", err)
		}
		if preimageHex == "" {
			return LSAT{
				ID:       id.TokenID,
				Preimage: nil,
				PayHash:  id.PaymentHash[:],
				Invoice:  "",
				Value:    2,
				Macaroon: *mac,
			}, nil
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
			Value:    2,
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
	}, nil
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}
