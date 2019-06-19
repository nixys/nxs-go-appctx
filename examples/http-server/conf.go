package main

import (
	"github.com/nixys/nxs-go-conf"
)

type confOpts struct {
	Bind     string `conf:"bind" conf_extraopts:"default=0.0.0.0:8080"`
	LogFile  string `conf:"logfile" conf_extraopts:"default=stdout"`
	LogLevel string `conf:"loglevel" conf_extraopts:"default=info"`
	PidFile  string `conf:"pidfile" conf_extraopts:"default=/tmp/example.pid"`
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
