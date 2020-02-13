package downloader

import (
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"syscall"
)

var wg sync.WaitGroup
var stat syscall.Statfs_t  // works only on POSIX Systems

func sendRequest(range_left, range_right, current_index int, url, pathPrefix string, failedThreads []bool) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	flag:=false

	range_header := "bytes=" + strconv.Itoa(range_left) + "-" + strconv.Itoa(range_right-1)
	req.Header.Add("Range", range_header)
	resp, _ := client.Do(req)
	if resp==nil {
		failedThreads[current_index]=true;
		flag=true
	}

	 if !flag {
		 defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode!=http.StatusAccepted && resp.StatusCode != http.StatusPartialContent {
			logrus.WithFields(logrus.Fields{
				"Status Code": resp.StatusCode,
			}).Error("Error while sending http request")
			failedThreads[current_index]=true;
		} else{
			// writing to a file
			currFilePath := pathPrefix + strconv.Itoa(current_index)
			outputFile, err := os.Create(currFilePath)
			_, err = io.Copy(outputFile, resp.Body)
			if err != nil {
				failedThreads[current_index]=true;
			}
		}
	}

	wg.Done()
}

func countNumChunks(url string, chunkSize int) (int64, int, int, error) {
	res, err := http.Head(url)
	if err != nil {
		return 0, 0, 0, err
	}
	contentLength:=res.ContentLength
	numWholeChunks:=int(contentLength)/chunkSize
	remainderChunkSize:=int(contentLength)%chunkSize
	return contentLength, numWholeChunks, remainderChunkSize, nil
}

func getAvailableSpace() (uint64, error) {
	wd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	syscall.Statfs(wd, &stat)
	availableSpace := stat.Bavail * uint64(stat.Bsize)
	return availableSpace, nil
}

func Downloader(url string, numThreads int) error{

	availableSpace, err := getAvailableSpace()
	if err != nil {
		return err
	}

	// create dump folder
	pathPrefix:="/tmp/dump/"
	err = os.MkdirAll(pathPrefix, os.ModePerm)  // creating files at mode 0777
	if err != nil {
		return err
	}

	chunkSize := 256000  // each chunksize is 256Kb
	contentLength, numWholeChunks, remainderChunkSize, err := countNumChunks(url, chunkSize)
	filename:=path.Base(url)  // parsing the output file name

	if contentLength >= int64(availableSpace) {
		logrus.WithError(err).Error("Not enough space available on disk")
		return err
	}

	numChunksPerThread := numWholeChunks/numThreads
	remainingThreads := numWholeChunks % numThreads
	if numWholeChunks == 0{
		remainingThreads=1
		numWholeChunks=1
	}

	// We have the required number of chunks along with the chunksize of each part.
	failedThreads := make([]bool, numWholeChunks+1)  // true if ith thread failed

	for i:=1;i<=numWholeChunks;i++{
		failedThreads[i]=false
	}

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
		if i==remainingThreads-1 {
			range_right+=remainderChunkSize
		}
		go sendRequest(range_left, range_right, i+numChunksPerThread, url, pathPrefix, failedThreads)
	}
	wg.Wait()

	// Retry : If any retries fail, return failure
	for i:=0;i<numWholeChunks;i++ {
		if failedThreads[i]==true {
			wg.Add(1)
			go sendRequest(i, i+1, i, url, pathPrefix, failedThreads)
		}
	}
	wg.Wait()

	// Verify if failed threads succeeded
	for i:=0;i<numWholeChunks;i++{
		if failedThreads[i]==true{
			logrus.Error("Response from server was nil. Please try later")
			os.RemoveAll(pathPrefix)
			return err
		}
	}

	out, err := os.Create(filename)
	if err != nil{
		return err
	}
	defer out.Close()

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

	logrus.Info("Successfully completed the request")

	err = os.RemoveAll(pathPrefix) // Removes the directory /tmp/dump/
	if err != nil {
		return err
	}
	return nil
}