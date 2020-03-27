//Package beacon implements an easy to use, but feature rich NIST Randomness Beacon API Wrapper in go
package beacon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Pulse struct {
		URI              string    `json:"uri"`
		Version          string    `json:"version"`
		CipherSuite      int       `json:"cipherSuite"`
		Period           int       `json:"period"`
		CertificateID    string    `json:"certificateId"`
		ChainIndex       int       `json:"chainIndex"`
		PulseIndex       int       `json:"pulseIndex"`
		TimeStamp        time.Time `json:"timeStamp"`
		LocalRandomValue string    `json:"localRandomValue"`
		External         struct {
			SourceID   string `json:"sourceId"`
			StatusCode int    `json:"statusCode"`
			Value      string `json:"value"`
		} `json:"external"`
		ListValues []struct {
			URI   string `json:"uri"`
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"listValues"`
		PrecommitmentValue string `json:"precommitmentValue"`
		StatusCode         int    `json:"statusCode"`
		SignatureValue     string `json:"signatureValue"`
		OutputValue        string `json:"outputValue"`
	} `json:"pulse"`
}

var defaultClient = &http.Client{}

// SetClient is useful if you want to use your own http client, it adds the possibility to use a proxy to fetch the data for example.
func SetClient(cli *http.Client) {
	defaultClient = cli
}

func getRecord(url string) (Record, error) {
	r, err := defaultClient.Get(url)
	if err != nil {
		err = errors.New("Couldn't get the record from the API: " + err.Error())
		return Record{}, err
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = errors.New("Couldn't read the API's response: " + err.Error())
		return Record{}, err
	}

	var rec Record
	err = json.Unmarshal(buf, &rec)
	if err != nil {
		err = errors.New("Couldn't unmarshal the API's response: " + err.Error())
		return Record{}, err
	}

	if time.Now().Unix() - rec.Pulse.TimeStamp.Unix() > 60 {
		return rec, errors.New(fmt.Sprintf("Beacon is stale: current=%d, pulse=%d", time.Now().Unix(), rec.Pulse.TimeStamp.Unix()))
	}

	return rec, nil
}

// LastRecord fetches the latest record from the beacon and returns the record
func LastRecord() (Record, error) {
	return getRecord("https://beacon.nist.gov/beacon/2.0/pulse/last")
}

// CurrentRecord fetches the record closest to the given timestamp
func CurrentRecord(t time.Time) (Record, error) {
	return getRecord("https://beacon.nist.gov/beacon/2.0/pulse/time/" + strconv.FormatInt(t.Unix(), 10))
}

// PreviousRecord fetches the record previous to the given timestamp
func PreviousRecord(t time.Time) (Record, error) {
	return getRecord("https://beacon.nist.gov/beacon/2.0/pulse/time/previous/" + strconv.FormatInt(t.Unix(), 10))
}

// NextRecord fetches the record after the given timestamp
func NextRecord(t time.Time) (Record, error) {
	return getRecord("https://beacon.nist.gov/beacon/2.0/pulse/time/next/" + strconv.FormatInt(t.Unix(), 10))
}

func (rec *Record) ChainpointFormat() string {
	if rec.Pulse.LocalRandomValue != "" {
		return fmt.Sprintf("%d:%s", rec.Pulse.TimeStamp.Unix(), strings.ToLower(rec.Pulse.OutputValue))
	}
	return ""
}