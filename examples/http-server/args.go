package main

import (
	"fmt"
	"os"

	"github.com/pborman/getopt/v2"
)

const (
	confPathDefault = "example.conf"
)

type argsOpts struct {
	configPath string
}

func argsRead() argsOpts {

	var o argsOpts

	args := getopt.New()

	helpFlag := args.BoolLong(
		"help",
		'h',
		"Show help")

	versionFlag := args.BoolLong(
		"version",
		'v',
		"Show program version")

	confPath := args.StringLong(
		"conf",
		'c',
		"",
		"Config file path")

	args.Parse(os.Args)

	/* Show help */
	if *helpFlag == true {
		argsHelp(args)
		os.Exit(0)
	}

	/* Show version */
	if *versionFlag == true {
		argsVersion()
		os.Exit(0)
	}

	/* Config path */
	if confPath == nil || len(*confPath) == 0 {
		o.configPath = confPathDefault
	} else {
		o.configPath = *confPath
	}

	return o
}

func argsHelp(args *getopt.Set) {

	additionalDescription := `
	
Additional description

  Just a sample.
`

	args.PrintUsage(os.Stdout)
	fmt.Println(additionalDescription)
}

func argsVersion() {
	fmt.Println("1.0")
}
