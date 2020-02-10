package main

import (
	"github.com/sirupsen/logrus"
	"downloader-go/downloader"
)

func main(){
	err:=downloader.Downloader("https://www.nasa.gov/sites/default/files/thumbnails/image/tn-p_lorri_fullframe_color.jpg", 20)
	if err != nil{
		logrus.WithError(err).Fatal("Received error while performing the HTTP Request")
	}
}