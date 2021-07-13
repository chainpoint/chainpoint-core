package analytics

import (
	"errors"
	"fmt"
	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"github.com/tendermint/tendermint/libs/log"
	"strings"
)

type UniversalAnalytics struct {
	CategoryName string
	GoogleUaID   string
	logger      log.Logger
}

func NewClient (CoreName string, GoogleUaId string, logger log.Logger) UniversalAnalytics {
	return UniversalAnalytics{
		CoreName,
		GoogleUaId,
		 logger,
	}
}

func (ua *UniversalAnalytics) SendEvent(drand, action, label, cd1, cd2, ip string) error {
	var err error
	if ua.GoogleUaID == "" {
		err =  errors.New("analytics: GA_TRACKING_ID environment variable is missing")
	}
	if ua.CategoryName == "" || action == "" {
	    err = errors.New("analytics: category and action are required")
	}
	if drand == "" {
		err = errors.New("analytics: no drand beacon yet")
	} else {
		arr := strings.Split(drand, ":")
		if len(arr) == 2 {
			drand = arr[0]
		}
	}
	if ua.LogError(err) != nil {
		return err
	}

	v := url.Values{
		"v":   {"1"},
		"tid": {ua.GoogleUaID},
		"cid": {drand},
		"t":   {"event"},
		"ec":  {ua.CategoryName},
		"ea":  {action},
	}

	if label != "" {
		v.Set("el", label)
	}

	if remoteIP, _, err := net.SplitHostPort(ip); err != nil {
		if remoteIP == "" && ip != "" {
			v.Set("uip", ip)
		} else {
			v.Set("uip", remoteIP)
		}
	}

	if cd1 != "" {
		v.Set("cd1", cd1)
	}

	if cd2 != "" {
		v.Set("cd2", cd2)
	}
	ua.logger.Info("Sending Event: " + v.Encode())

	// NOTE: Google Analytics returns a 200, even if the request is malformed.
	resp, err := http.PostForm("https://www.google-analytics.com/debug/collect", v)
	b, err := ioutil.ReadAll(resp.Body)
	if err == nil && len(string(b)) != 0 {
		ua.logger.Info(string(b))
	}
	ua.LogError(err)
	return err
}

func (ua *UniversalAnalytics) LogError(err error) error {
	if err != nil {
		 ua.logger.Error(fmt.Sprintf("Error in %s: %s", util.GetCurrentFuncName(2), err.Error()))
	}
	return err
}