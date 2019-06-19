package main

import (
	"github.com/nixys/nxs-go-conf"
)

type confOpts struct {
	TimeInterval int    `conf:"time_int" conf_extraopts:"default=5"` // in seconds
	LogFile      string `conf:"logfile" conf_extraopts:"default=stdout"`
	LogLevel     string `conf:"loglevel" conf_extraopts:"default=info"`
	PidFile      string `conf:"pidfile" conf_extraopts:"default=/tmp/example.pid"`
}

func confRead(confPath string) (confOpts, error) {

	var c confOpts

	err := conf.Load(&c, conf.Settings{
		ConfPath:    confPath,
		ConfType:    conf.ConfigTypeYAML,
		UnknownDeny: true,
	})

	return c, err
}
