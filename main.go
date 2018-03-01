package main

import (
	"flag"
	"fmt"

	"github.com/de0gee/de0gee-cloud/src"
)

func main() {
	var (
		doDebug                   bool
		port                      string
		dataFolder, serverAddress string
		useSSL                    bool
	)
	flag.StringVar(&port, "port", "8002", "port to run server")
	flag.StringVar(&dataFolder, "data", "data", "folder to data (default ./data)")
	flag.StringVar(&serverAddress, "address", "192.168.0.23:8002", "address of server")
	flag.BoolVar(&doDebug, "debug", false, "enable debugging")
	flag.BoolVar(&useSSL, "ssl", false, "enable SSL")
	flag.Parse()

	if doDebug {
		cloud.SetLogLevel("debug")
	} else {
		cloud.SetLogLevel("info")
	}

	if dataFolder != "" {
		cloud.DataFolder = dataFolder
	}
	cloud.Port = port
	cloud.ServerAddress = serverAddress
	cloud.UseSSL = useSSL

	err := cloud.Run()
	if err != nil {
		fmt.Println(err)
	}
}
