package fee

import (
	"encoding/json"
	"github.com/btcsuite/btcd/blockchain"
	"net/http"
	"time"
)

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

// GetThirdPartyFeeEstimate : get sat/vbyte fee and convert to sat/kw
func GetThirdPartyFeeEstimate() (int64, error) {
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
