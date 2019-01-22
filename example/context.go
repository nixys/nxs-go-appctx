package main

import (
	"os"

	"github.com/nixys/nxs-go-appctx"
	"github.com/sirupsen/logrus"
)

func contextInit(ctx interface{}, cfgPath string) (appctx.CfgData, error) {

	var cfgData appctx.CfgData

	// Read config file
	conf, err := confRead(cfgPath)
	if err != nil {
		return cfgData, err
	}

	// Set application context
	c := ctx.(*selfContext)
	c.timeInterval = conf.TimeInterval

	// Fill `appctx.CfgData`
	cfgData.LogFile = conf.LogFile
	cfgData.LogLevel = conf.LogLevel
	cfgData.PidFile = conf.PidFile

	return cfgData, nil
}

func contextReload(ctx interface{}, cfgPath string, singnal os.Signal) (appctx.CfgData, error) {

	log.WithFields(logrus.Fields{
		"signal": singnal,
	}).Debug("program reload")

	return contextInit(ctx, cfgPath)
}

func contextFree(ctx interface{}, signal os.Signal) int {

	log.WithFields(logrus.Fields{
		"signal": signal,
	}).Debug("got termination signal")

	return 0
}
