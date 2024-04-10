package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type config struct {
	filename string
	Target   string
	Delay    int
	Method   string
	Thread   int
	SaveFile string
}

func webfuzz(Config config) {
	var wg sync.WaitGroup
	total_found := 0

	file, err := os.Open(Config.filename)
	if err != nil {
		fmt.Println("filepath not valid:", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	sem := make(chan struct{}, Config.Thread)

	for scanner.Scan() {
		url := Config.Target + "/" + scanner.Text()

		sem <- struct{}{} // max goroutine

		wg.Add(1)

		go func(url string) {
			defer func() {
				// exit of the canal
				<-sem
				wg.Done()
			}()

			delay := time.Duration(Config.Delay) * time.Second
			time.Sleep(delay)

			statusCode, err := getStatusCode(url, Config.Method)
			if err != nil {
				fmt.Println("Error sending request:", err)
				return
			}

			if statusCode == 200 {
				total_found++
				fmt.Println("\033[1;92m[+]\033[0m", url, statusCode)
				data := fmt.Sprintf("%s %d", url, statusCode)
				if Config.SaveFile != "None" {
					writereport(Config.SaveFile, data)
				}
			}
		}(url)
	}

	wg.Wait() // wait of all goroutine

	fmt.Println("\nTotal found:", total_found)
	if Config.SaveFile != "None" {
		fmt.Fprintln(os.Stdout, []any{"File saved has:", Config.SaveFile}...)
	}
}

func getStatusCode(url string, method string) (int, error) {
	var resp *http.Response
	var err error

	if method == "GET" {
		resp, err = http.Get(url)
	} else {
		resp, err = http.Post(url, "", nil)
	}

	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	return statusCode, nil
}

func writereport(filepath string, data string) {
	err := func() error {
		f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		writer := bufio.NewWriter(f)
		defer writer.Flush()

		lines := strings.Split(data, "\n")
		for _, line := range lines {
			_, err := writer.WriteString(line + "\n")
			if err != nil {
				return err
			}
		}

		return nil
	}()
	if err != nil {
		panic(err)
	}
}

func main() {
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		fmt.Println("\nExisting")
		os.Exit(0)
	}()

	var Config config

	// command
	flag.StringVar(&Config.filename, "f", "default", "filename")
	flag.StringVar(&Config.Target, "target", "default", "url_target")
	flag.IntVar(&Config.Delay, "d", 0, "Enter the delay in seconds")
	flag.StringVar(&Config.Method, "m", "GET", "GET or POST")
	flag.IntVar(&Config.Thread, "t", 10, "Enter the number of thread.")
	flag.StringVar(&Config.SaveFile, "s", "None", "Save in output file.")

	flag.Parse()

	if Config.filename == "default" {
		fmt.Println("Usage: go run main.go -f <filename>")
		return
	}
	if Config.Target == "default" {
		fmt.Println("Usage: go run main.go -target <https://exemple.com>")
		return
	}

	// info config
	info := "\033[1;96m[+]\033[0m"

	fmt.Println(info, "Delay in seconds:", Config.Delay)
	fmt.Println(info, "Wordlist:", Config.filename)
	fmt.Println(info, "Url Target:", Config.Target+"\n")

	// web fuzzer
	webfuzz(Config)

}
