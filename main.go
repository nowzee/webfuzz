package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type config struct {
	filename    string
	Target      string
	Delay       int
	Thread      int
	SaveFile    string
	maxtime     int
	lenghtbody  int
	Status_Code string
	// unused
	KeyWord string
}

var List_status = map[int]bool{
	200: true,
	204: true,
	301: true,
	302: true,
	307: true,
	401: true,
	500: true,
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
			if strings.HasSuffix(Config.Target, "/") {
				r := strings.TrimPrefix(scanner.Text(), "/")
				url = Config.Target + r
			} else {
				url = Config.Target + scanner.Text()
			}
		} else {
			if strings.HasSuffix(Config.Target, "/") {
				url = Config.Target + scanner.Text()
			} else {
				url = Config.Target + "/" + scanner.Text()
			}
		}

		sem <- struct{}{} // max goroutine

		wg.Add(1)

		go func(url string) {
			defer func() {
				// exit of the canal
				<-sem
				wg.Done()
			}()

			delay := time.Duration(Config.Delay) * time.Millisecond
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

			data := fmt.Sprintf("%-40s [Code: %d] || [Size: %d]", url, statusCode, body)

			fmt.Printf("%s %-40s [Code: %d] [Size: %d]\n", info, url, statusCode, body)

			if Config.SaveFile != "None" {
				writereport(Config.SaveFile, data)
			}
		}(url)
	}

	wg.Wait() // wait of all goroutine

	fmt.Println("\nTotal found:", total_found)
	if Config.SaveFile != "None" {
		fmt.Fprintln(os.Stdout, []any{"\033[1;96m[+]\033[0m", "File saved has:", Config.SaveFile}...)
	}
}

func getStatusCode(url string, Config config) (int, int, error) {

	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, err
	}

	defer resp.Body.Close()
	statusCode := resp.StatusCode
	if List_status[statusCode] {
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
	flag.IntVar(&Config.Delay, "d", 0, "Enter the delay in Millisecond")
	flag.IntVar(&Config.Thread, "t", 10, "Enter the number of thread.")
	flag.StringVar(&Config.SaveFile, "o", "None", "Save in output file.")
	flag.IntVar(&Config.maxtime, "time", 0, "Max time to fuzz in seconds")
	flag.IntVar(&Config.lenghtbody, "exclude-lenght", 0, "Exclude lenght")
	flag.StringVar(&Config.Status_Code, "X", "None", "Exclude status code with separator ;")

	flag.Parse()

	if Config.filename == "default" {
		fmt.Println("Usage: go run main.go -f <filename>")
		return
	}
	if Config.Target == "default" {
		fmt.Println("Usage: go run main.go -target <https://example.com>")
		return
	}
	if Config.Status_Code != "None" {
		s := Config.Status_Code
		arr := strings.Split(s, ";")

		for _, valueStr := range arr {
			intValue, err := strconv.Atoi(valueStr)
			if err != nil {
				fmt.Println(err)
				return
			}

			List_status[intValue] = false
		}

	}

	// info config
	info := "\033[1;96m[+]\033[0m"

	fmt.Println(info, "Delay in Millisecond:", Config.Delay)
	fmt.Println(info, "Max time to fuzz in second:", Config.maxtime)
	fmt.Println(info, "Exclude Lenght :", Config.lenghtbody)
	fmt.Println(info, "Thread:", Config.Thread)
	fmt.Println(info, "Wordlist:", Config.filename)
	fmt.Println(info, "Url Target:", Config.Target+"\n")

	// web fuzzer
	webfuzz(Config)

}
