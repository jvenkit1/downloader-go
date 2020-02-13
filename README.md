# downloader-go
CLI File downloader written in Golang

# Installation:
1. Installing Golang:
  Please refer to the following link to download and setup Golang :
  https://golang.org/doc/install
2. Building a binary
  To rebuild a binary, run the following command: `go build main.go`

# Usage:
1. Run the downloader using the following command:
  `./main <URL> -c <nThreads>`
  nThreads here specifies the number of threads required. The default value for this is 1.

# Architecture:
The tool is written in the language Golang. General HTTP Servers are able to respond to a 'GET' request and send data in chunked format, upon specifying the Range_Request Header. If we ask each thread to download one chunk of the data, we can parallelize the operation and download data efficiently.
However if we simply decide the size of the chunk, based on the number of threads, it could lead to memory overflow issues. (If we download a very large file ~1G, with only 2 threads, each thread downloads ~500M).
Threads are supposed to perform lightweight tasks. Considering this, the size of each chunk is set to 256Kb. Thus, each thread at any given point of time can download only 256Kb of data.

Each thread writes its downloaded data to a file in /tmp/dump/ folder. Thus, when all threads have finished their operation, the data is read, consolidated and written to our downloaded file.

# Challenges:
1. Further testing/development is required in cases where the server sends a redirect request to another link which contains the data to be downloaded.
