package main

import (
	"github.com/nixys/nxs-go-appctx"
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

func contextReload(ctx interface{}, cfgPath string) (appctx.CfgData, error) {

	log.Debug("reloading context")

	return contextInit(ctx, cfgPath)
}

func contextFree(ctx interface{}) int {

	log.Debug("freeing context")

	return 0
}
