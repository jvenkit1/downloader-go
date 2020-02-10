package downloader

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

var wg sync.WaitGroup

func getChunkSize(url string, numThreads int) (int, int, error) {
	res, err:=http.Head(url)
	if err != nil{
		return 0, 0, err
	}
	contentLength:=res.ContentLength
	logrus.WithFields(logrus.Fields{
		"URL": url,
		"ContentSize": humanize.Bytes(uint64(contentLength)),
	}).Info("Reached downloader function")
	return int(contentLength)/numThreads, int(contentLength)%numThreads, nil
}

/* @TODO: Find the optimal number of chunks to be created.
Mainly, compare the two following cases:
1. Chunks=NumThreads=10 and FileSize=100MB. Size of each chunk is 10Mb
2. Chunks=50 > NumThreads=10 and FileSize=100Mb. Size of each chunk is 2Mb and each thread performs the same activity 5 times
 */

func Downloader(url string, numThreads int) error{

	chunkSize, remainder, err := getChunkSize(url, numThreads)  // chunkSize signifies the size of each chunk. remainder is to be added to the last chunk
	if err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"ChunkSize": chunkSize,
	}).Info("Divided data into chunks")

	filename:=path.Base(url)

	out, err := os.Create(filename)
	if err != nil{
		return err
	}
	defer out.Close()

	// We have the required number of chunks along with the chunksize of each part.
	temporaryData := make([]string, numThreads+1)

	for i:=0; i<numThreads;i++{
		wg.Add(1)

		range_left := i*chunkSize
		range_right := (i+1)*chunkSize
		if(i==numThreads-1){
			range_right += remainder
		}

		// Anonymous function
		go func(range_left, range_right int, current_index int){
			client:=&http.Client{}
			req, _ := http.NewRequest("GET", url, nil)

			range_header := "bytes="+strconv.Itoa(range_left)+"-"+strconv.Itoa(range_right-1)
			req.Header.Add("Range", range_header)
			resp, _ := client.Do(req)
			defer resp.Body.Close()

			// @TODO: CHANGE THE BELOW IMPLEMENTATION TO MAKE IT MEMORY SAFE
			reader, _ := ioutil.ReadAll(resp.Body)
			temporaryData[current_index]=string(reader)

			wg.Done()
		}(range_left, range_right, i)
	}
	wg.Wait()

	//@TODO: Success check for the response
	logrus.WithFields(logrus.Fields{
		"Filename": filename,
	}).Info("Received response from the server")

	// io.copy copies the response body which is Bytes format, 32 Bytes at a time.
	// @TODO: Resolve this bottleneck. Move away from io.ReadAll to an alternate solution
	for i:=0;i<numThreads;i++ {
		_, err = io.Copy(out, strings.NewReader(temporaryData[i]))
		if err != nil {
			return err
		}
		fmt.Println(i)
	}

	stats, err := out.Stat()
	if err != nil {
		return err
	}
	// @TODO: Verify this conversion of int64 to uint64
	filesize:=uint64(stats.Size())
	logrus.WithFields(logrus.Fields{
		"Size": humanize.Bytes(filesize),
	}).Info("Successfully completed the request")
	return nil
}