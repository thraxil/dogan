package main // github.com/thraxil/dogan
import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/go-kit/kit/log"
)

type actionconfig struct {
	Metric        string
	Threshold     float64
	Direction     string
	Command       string
	CheckInterval int
}

type config struct {
	GraphiteBase  string
	CheckInterval int

	Actions map[string]actionconfig
}

func main() {
	configFile := flag.String("config", "/etc/dogan/config.toml", "config file location")
	flag.Parse()

	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	logger.Log("msg", "starting")

	sigs := make(chan os.Signal, 1)

	var conf config
	if _, err := toml.DecodeFile(*configFile, &conf); err != nil {
		logger.Log("msg", "error loading config file", "error", err)
		return
	}

	for k, action := range conf.Actions {
		alogger := log.With(logger, "action", k)
		a := newAction(action, conf.GraphiteBase, conf.CheckInterval, httpFetcher{}, alogger)
		go a.Run()
	}

	// wait around for a SIGINT/SIGTERM
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	logger.Log("msg", "exiting")
}
