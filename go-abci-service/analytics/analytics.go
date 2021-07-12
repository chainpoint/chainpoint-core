package analytics

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

type UniversalAnalytics struct {
	CategoryName string
	GoogleUaID   string
}

func NewClient (CoreName string, GoogleUaId string) UniversalAnalytics {
	return UniversalAnalytics{
		CoreName,
		GoogleUaId,
	}
}

func (ua *UniversalAnalytics) SendEvent(drand, action, label, cd1, cd2, ip string) error {
	if ua.GoogleUaID == "" {
		return errors.New("analytics: GA_TRACKING_ID environment variable is missing")
	}
	if ua.CategoryName == "" || action == "" {
		return errors.New("analytics: category and action are required")
	}
	if drand == "" {
		return errors.New("analytics: no drand beacon yet")
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
		v.Set("uip", remoteIP)
	}

	if cd1 != "" {
		v.Set("cd1", cd1)
	}

	if cd2 != "" {
		v.Set("cd2", cd1)
	}

	// NOTE: Google Analytics returns a 200, even if the request is malformed.
	_, err := http.PostForm("https://www.google-analytics.com/collect", v)
	return err
}