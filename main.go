package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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
	Thread   int
	SaveFile string
	maxtime  int
	// unused
	KeyWord    string
	lenghtbody int
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
		var url = ""

		if strings.HasPrefix(scanner.Text(), "/") {
			url = Config.Target + scanner.Text()
		} else {
			url = Config.Target + "/" + scanner.Text()
		}

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

			statusCode, body, err := getStatusCode(url, Config)
			if err != nil {
				return
			}
			if statusCode == 0 {
				return
			}

			total_found++
			info := "\033[1;92m[+]\033[0m"
			fmt.Printf("[Code: %d  Lenght-body: %d]", statusCode, body)
			fmt.Printf("\n%s %s\n", info, url)
			data := fmt.Sprintf("%s [%d || %d]", url, statusCode, body)
			if Config.SaveFile != "None" {
				writereport(Config.SaveFile, data)
			}
		}(url)
	}

	wg.Wait() // wait of all goroutine

	fmt.Println("\nTotal found:", total_found)
	if Config.SaveFile != "None" {
		fmt.Fprintln(os.Stdout, []any{"File saved has:", Config.SaveFile}...)
	}
}

func getStatusCode(url string, Config config) (int, int, error) {
	List_status := map[int]bool{
		200: true,
		301: true,
	}

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, err
	}

	defer resp.Body.Close()

	statusCode := resp.StatusCode
	if _, ok := List_status[statusCode]; ok {

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, 0, nil
		}

		if Config.lenghtbody != 0 {
			if Config.lenghtbody != len(body) {
				return statusCode, len(body), nil
			}

		} else {
			return statusCode, len(body), nil
		}

	}
	return 0, 0, nil

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
	var Config config

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		fmt.Println("\nExisting")
		os.Exit(0)
	}()

	go func() {
		if Config.maxtime != 0 {
			delay := time.Duration(Config.maxtime) * time.Second
			time.Sleep(delay)
			fmt.Println("\nExisting")
			os.Exit(0)
		}
	}()

	// command
	flag.StringVar(&Config.filename, "f", "default", "filename")
	flag.StringVar(&Config.Target, "target", "default", "url_target")
	flag.IntVar(&Config.Delay, "d", 0, "Enter the delay in seconds")
	flag.IntVar(&Config.Thread, "t", 10, "Enter the number of thread.")
	flag.StringVar(&Config.SaveFile, "o", "None", "Save in output file.")
	flag.IntVar(&Config.maxtime, "time", 0, "Max time to fuzz in seconds")
	flag.IntVar(&Config.lenghtbody, "exclude-lenght", 0, "Exclude lenght")

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
	fmt.Println(info, "Thread:", Config.Thread)
	fmt.Println(info, "Wordlist:", Config.filename)
	fmt.Println(info, "Url Target:", Config.Target+"\n")

	// web fuzzer
	webfuzz(Config)

}
