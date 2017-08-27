package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
)

type fetcher interface {
	Get(string) (*http.Response, error)
}

type httpFetcher struct{}

type action struct {
	metric        string
	threshold     float64
	direction     string
	command       string
	graphiteBase  string
	checkInterval int
	fetcher       fetcher
	logger        log.Logger
}

func newAction(c actionconfig, base string, interval int, fetcher fetcher, logger log.Logger) *action {
	if c.CheckInterval != 0 {
		interval = c.CheckInterval
	}
	return &action{
		metric:        cleanMetric(c.Metric),
		threshold:     c.Threshold,
		direction:     c.Direction,
		command:       c.Command,
		graphiteBase:  base,
		checkInterval: interval,
		fetcher:       fetcher,
		logger:        logger,
	}
}

func cleanMetric(metric string) string {
	re := regexp.MustCompile("[ \n\t\r]+")
	return re.ReplaceAllString(metric, "")
}

func (a *action) URL() string {
	return a.graphiteBase + "?target=keepLastValue(" + a.metric + ")&format=raw&from=-2hours"
}

func (h httpFetcher) Get(url string) (*http.Response, error) {
	return http.Get(url)
}

func (a *action) Fetch() (float64, error) {
	resp, err := a.fetcher.Get(a.URL())
	if err != nil {
		return 0.0, errors.New("graphite request failed")
	}
	if resp.Status != "200 OK" {
		return 0.0, errors.New("graphite did not return 200 OK")
	}
	b, _ := ioutil.ReadAll(resp.Body)
	s := fmt.Sprintf("%s", b)
	lv, err := extractLastValue(s)
	return lv, err
}

func extractLastValue(rawResponse string) (float64, error) {
	// just take the most recent value
	parts := strings.Split(strings.Trim(rawResponse, "\n\t "), ",")
	return strconv.ParseFloat(parts[len(parts)-1], 64)
}

func (a *action) Check() {
	v, err := a.Fetch()
	if err != nil {
		a.logger.Log("msg", "fetch failed", "error", err)
	} else {
		if a.direction == "above" {
			if v >= a.threshold {
				a.logger.Log("msg", "triggering", "value", v, "threshold", a.threshold)
				a.Execute()
			} // else, nothing to do
		} else {
			if v <= a.threshold {
				a.logger.Log("msg", "triggering", "value", v, "threshold", a.threshold)
				a.Execute()
			} // else, nothing to do
		}
	}
}

func (a *action) Execute() {
	// run the command
	a.logger.Log("command", a.command)
}

func (a *action) Run() {
	for {
		a.Check()
		// delay + jitter
		jitter := rand.Intn(a.checkInterval / 10)
		time.Sleep(time.Duration(a.checkInterval+jitter) * time.Second)
	}
}
