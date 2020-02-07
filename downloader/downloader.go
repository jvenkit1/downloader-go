package downloader

import(
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"github.com/dustin/go-humanize"
	"os"
	"path"
)

func Download(url string) error{
	logrus.WithFields(logrus.Fields{
		"URL": url,
	}).Info("Reached downloader function")

	resp, err := http.Get(url)
	if(err != nil){
		return err
	}
	defer resp.Body.Close()

	logrus.Info("Parsing the URL and generating file now.")

	//@TODO: Success check for the response
	filename := path.Base(resp.Request.URL.String())
	logrus.WithFields(logrus.Fields{
		"Filename": filename,
		"Status code": resp.StatusCode,
	}).Info("Received response from the server")

	out, err := os.Create(filename)
	if err != nil{
		return err
	}
	defer out.Close()

	// io.copy copies the response body which is Bytes format, 32 Bytes at a time.
	// @TODO: Resolve this bottleneck
	_, err = io.Copy(out, resp.Body)
	if err != nil{
		return err
	}
	stats, err := out.Stat()
	// @TODO: Verify this conversion of int64 to uint64
	filesize:=uint64(stats.Size())
	logrus.WithFields(logrus.Fields{
		"Size": humanize.Bytes(filesize),
	}).Info("Successfully completed the request")
	return nil
}