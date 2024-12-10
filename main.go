package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

const host = "http://srv.msk01.gigacorp.local/_stats"
const loadLimit = 30
const memoryLimit = 0.8
const diskLimit = 0.9
const networkLimit = 0.9

var requestErrorCounter = 0

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c)

	ticker := time.NewTicker(time.Second)
	stop := make(chan bool)

	go func() {
		defer func() { stop <- true }()
		for {
			select {
			case <-ticker.C:
				MakeRequest()
				if requestErrorCounter >= 3 {
					fmt.Println("Unable to fetch server statistic")
				}
			case <-stop:
				return
			}
		}
	}()

	<-c
	ticker.Stop()

	stop <- true
	<-stop
}

func MakeRequest() {
	response, err := http.Get(host)

	if err != nil || response.StatusCode != http.StatusOK {
		requestErrorCounter += 1
		return
	}

	bodyBytes, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		requestErrorCounter += 1
		return
	}
	ParseRequestBody(string(bodyBytes))
}

func ParseRequestBody(body string) {
	arrayValues := strings.Split(body, ",")
	var memory float64
	var disk float64
	var network float64

	if len(arrayValues) != 7 {
		requestErrorCounter += 1
		return
	}

	for index, value := range arrayValues {
		number, err := strconv.Atoi(value)

		if err != nil {
			requestErrorCounter += 1
			return
		}

		switch index {
		case 0:
			if number > loadLimit {
				fmt.Printf("Load Average is too high: %v\n", number)
			}
		case 1:
			memory = float64(number)
		case 2:
			memoryUsage := float64(number) / memory
			if memoryUsage > memoryLimit {
				fmt.Printf("Memory usage too high: %v%%\n", int(memoryUsage*100))
			}
		case 3:
			disk = float64(number)
		case 4:
			diskUsage := float64(number) / disk
			if diskUsage > diskLimit {
				freeSpace := (int(disk) - number) / 1000000
				fmt.Printf("Free disk space is too low: %v Mb left\n", freeSpace)
			}
		case 5:
			network = float64(number)
		case 6:
			networkUsage := float64(number) / network
			if networkUsage > networkLimit {
				availableNetwork := (int(network) - number) / 125000
				fmt.Printf("Network bandwidth usage high: %v Mbit/s available\n", availableNetwork)
			}
		}
	}

	requestErrorCounter = 0
}
