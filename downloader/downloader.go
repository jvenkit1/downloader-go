package downloader

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
)

var wg sync.WaitGroup

func sendRequest(range_left, range_right, current_index int, url, pathPrefix string, failedThreads []bool) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)

	range_header := "bytes=" + strconv.Itoa(range_left) + "-" + strconv.Itoa(range_right-1)
	req.Header.Add("Range", range_header)
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	// writing to a file
	currFilePath := pathPrefix + strconv.Itoa(current_index)
	outputFile, err := os.Create(currFilePath)
	_, err = io.Copy(outputFile, resp.Body)
	if err != nil {
		failedThreads[current_index]=true;
	}

	wg.Done()
}

func countNumChunks(url string, chunkSize int) (int, int, error) {
	res, err := http.Head(url)
	if err != nil {
		return 0, 0, err
	}
	contentLength:=res.ContentLength
	numWholeChunks:=int(contentLength)/chunkSize
	remainderChunkSize:=int(contentLength)%chunkSize
	return numWholeChunks, remainderChunkSize, nil
}

func Downloader(url string, numThreads int) error{
	// create dump folder
	pathPrefix:="/tmp/dump/"
	err := os.MkdirAll(pathPrefix, os.ModePerm)  // creating files at mode 0777
	if err != nil {
		return err
	}

	chunkSize := 256000  // each chunksize is 256Kb
	numWholeChunks, remainderChunkSize, err := countNumChunks(url, chunkSize)
	filename:=path.Base(url)  // parsing the output file name

	out, err := os.Create(filename)
	if err != nil{
		return err
	}
	defer out.Close()

	numChunksPerThread := numWholeChunks/numThreads
	remainingThreads := numWholeChunks % numThreads
	if numWholeChunks == 0{
		remainingThreads=1
		numWholeChunks=1
	}

	// We have the required number of chunks along with the chunksize of each part.
	failedThreads := make([]bool, numWholeChunks+1)  // true if ith thread failed

	currFilePath := pathPrefix + string(1)
	fmt.Print(currFilePath)

	for j:=0; j<numChunksPerThread; j++ {
		for i := 0; i < numThreads; i++ {
			wg.Add(1)

			range_left := (j+i) * chunkSize
			range_right := (j+i + 1) * chunkSize

			go sendRequest(range_left, range_right, i+j, url, pathPrefix, failedThreads)
		}
	}
	for i := 0; i < remainingThreads; i++ {
		wg.Add(1)

		range_left := (i+numChunksPerThread) * chunkSize
		range_right := (i+numChunksPerThread + 1) * chunkSize
		if( i==remainingThreads-1){
			range_right+=remainderChunkSize
		}
		go sendRequest(range_left, range_right, i+numChunksPerThread, url, pathPrefix, failedThreads)
	}
	wg.Wait()

	// Retry : If any retries fail, return failure

	//@TODO: Success check for the response
	logrus.WithFields(logrus.Fields{
		"Filename": filename,
	}).Info("Received response from the server")

	// io.copy copies the response body which is Bytes format, 32 Bytes at a time.
	for i:=0;i<numWholeChunks;i++ {
		file, err := os.Open(pathPrefix+strconv.Itoa(i))
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(out, file)
		if err != nil {
			return err
		}
	}

	stats, err := out.Stat()
	if err != nil {
		return err
	}

	filesize:=uint64(stats.Size())
	logrus.WithFields(logrus.Fields{
		"Size": humanize.Bytes(filesize),
	}).Info("Successfully completed the request")

	err = os.RemoveAll(pathPrefix) // Removes the directory /tmp/dump/
	if err != nil {
		return err
	}
	return nil
}