package main

import (
	"flag"
	"fmt"

	"github.com/de0gee/de0gee-cloud/src"
)

func main() {
	var (
		doDebug    bool
		port       string
		dataFolder string
	)
	flag.StringVar(&port, "port", "8002", "port to run server")
	flag.StringVar(&dataFolder, "data", "data", "folder to data (default ./data)")
	flag.BoolVar(&doDebug, "debug", false, "enable debugging")
	flag.Parse()

	if doDebug {
		cloud.SetLogLevel("debug")
	} else {
		cloud.SetLogLevel("info")
	}

	if dataFolder != "" {
		cloud.DataFolder = dataFolder
	}

	err := cloud.Run(port)
	if err != nil {
		fmt.Println(err)
	}
}
