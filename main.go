package main

import (
	"downloader-go/downloader"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

func main(){
	var nThread int

	if(len(os.Args)<1){
		logrus.Fatal("Please provide input in correct order")
	}

	url := os.Args[1]
	if len(os.Args) < 3 {
		logrus.Info("No thread count provided. Defaulting to 1")
		nThread=1
	} else {
		arg2 := os.Args[3]
		nThread, _ = strconv.Atoi(arg2)

		if (nThread < 1) {
			logrus.Error("Invalid number of threads provided. Defaulting to 1")
			nThread = 1
		}
	}

	logrus.WithFields(logrus.Fields{
		"URL": url,
		"NTHREADS": nThread,
	}).Info("Starting Download")

	err:=downloader.Downloader(url, nThread)
	if err != nil{
		logrus.Fatal("HTTP Request failed. Please check URL and Retry")
	}
}