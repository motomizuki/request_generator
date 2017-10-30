package main

import (
	"os"
	"time"

	"bufio"
	"log"

	"bytes"
	"net/http"

	"sync"

	"sync/atomic"

	"gopkg.in/urfave/cli.v1"
)

const (
	MaxScanTokenSize = 512 * 1024
)

func getLines(filePath string) (lines []string, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	lines = make([]string, 0, 100)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(nil, MaxScanTokenSize)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	err = scanner.Err()
	return
}

func request(url string, method string, num int, filePath string, clientTimeout int) {
	var jsonl []string
	var err error

	var errorCount uint32
	var successCount uint32

	if filePath != "" {
		jsonl, err = getLines(filePath)
		if err != nil {
			log.Fatalln(err)
		}
	}
	jsonlSize := len(jsonl)
	sendData := jsonlSize > 0
	timeout := time.Duration(clientTimeout) * time.Second
	client := &http.Client{Timeout: timeout}
	wg := &sync.WaitGroup{}
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var req *http.Request
			if sendData {
				body := bytes.NewBuffer([]byte(jsonl[i%jsonlSize]))
				req, err = http.NewRequest(method, url, body)
			} else {
				req, err = http.NewRequest(method, url, nil)
			}
			if err != nil {
				atomic.AddUint32(&errorCount, 1)
				log.Println("create request error")
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				atomic.AddUint32(&errorCount, 1)
				log.Println(err)
				return
			} else if resp.StatusCode >= 400 {
				atomic.AddUint32(&errorCount, 1)
			} else {
				atomic.AddUint32(&successCount, 1)
			}

			defer resp.Body.Close()
			return
		}()
	}
	wg.Wait()
	log.Printf("success count is %d. error count is %d", successCount, errorCount)
}

func main() {
	app := cli.NewApp()
	app.Name = "HTTP Request Generator"
	app.Usage = "Generate HTTP Request for benchmark."
	app.Version = "0.0.1"
	app.Compiled = time.Now()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url, u",
			Usage: "URL to throw request",
		},
		cli.StringFlag{
			Name:  "method, m",
			Usage: "Request method(GET, POST, PUT, PATCH, DELETE). default GET",
			Value: "GET",
		},
		cli.StringFlag{
			Name:  "file, f",
			Usage: "request body data(jsonline data)",
		},
		cli.IntFlag{
			Name:  "num, n",
			Usage: "request nums. Default 1",
			Value: 1,
		},

		cli.IntFlag{
			Name:  "timeout, t",
			Usage: "request timeout secs. Default 5)",
			Value: 5,
		},
	}
	app.Action = func(c *cli.Context) {
		url := c.String("url")
		method := c.String("method")
		num := c.Int("num")
		file := c.String("file")
		timeout := c.Int("timeout")

		request(url, method, num, file, timeout)
	}

	app.Run(os.Args)
}
