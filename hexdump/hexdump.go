package main

import (
	"bufio"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type pkg struct {
	Name     string    `json:"name"`
	Releases []release `json:"releases"`
}

type release struct {
	Version      string                 `json:"version"`
	URL          string                 `json:"url"`
	Requirements map[string]requirement `json:"requirements"`
}

type requirement struct {
	App string `json:"app"`
}

type pkgs []pkg

var libs map[string]pkg

// start with whitelist, get all releases
// for each release, get all dependencies
// for each dependency, get all releases
// and on and on...

var DefaultTransport http.RoundTripper = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
}

func main() {
	allPkgs := make([]pkg, 0)
	libs = make(map[string]pkg)

	client := &http.Client{}
	client.Timeout = time.Second * 30
	client.Transport = DefaultTransport

	whitelist := make([]string, 0)

	file, err := os.Open("/app/packages.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		whitelist = append(whitelist, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for _, lib := range whitelist {
		capturePackage(lib, client, 0)
	}

	log.Println(allPkgs)
	log.Println("Downloading packages and tarballs...")

	for _, p := range libs {
		pkgName := p.Name

		if pkgName == "" {
			continue
		}

		downloadPackage(pkgName, client, 0)

		for _, r := range p.Releases {
			downloadRelease(pkgName, r.Version, client, 0)
		}
	}

	downloadRegistry(client)
	downloadCSV(client, "hex", "hex-1.x.csv", true)
	downloadCSV(client, "hex", "hex-1.x.csv.signed", false)
	downloadCSV(client, "rebar", "rebar-1.x.csv", true)
	downloadCSV(client, "rebar", "rebar-1.x.csv.signed", false)
}

func capturePackage(lib string, client *http.Client, attempt int) {
	// if the lib is already in the map, we can return
	if _, exists := libs[lib]; exists {
		return
	}

	attempt++
	fmt.Println("Attempt", attempt, "to get", lib, "package info")

	url := "https://hex.pm/api/packages/" + lib
	response, err := client.Get(url)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			fmt.Println("Request timed out, retrying")
			capturePackage(lib, client, attempt)
			return
		}

		log.Fatal(err.Error())
	}
	defer response.Body.Close()

	fmt.Println("Received code:", response.StatusCode);

	if response.StatusCode != 200 {
		if attempt <= 3 {
			fmt.Println("Received", response.StatusCode, "going to sleep for one minute")
			time.Sleep(time.Second * 60)
			capturePackage(lib, client, attempt)
			return
		} else {
			log.Fatal("Received 429 on attempt 3, exiting")
		}
	}

	var pkgResult pkg
	err = json.NewDecoder(response.Body).Decode(&pkgResult)
	if err != nil {
		log.Fatal(err.Error())
	}

	libs[pkgResult.Name] = pkgResult

	for _, rel := range pkgResult.Releases {
		getReleaseRequirements(rel.URL, client, 0)
	}
}

func getReleaseRequirements(releaseURL string, client *http.Client, attempt int) {
	attempt++

	response, err := client.Get(releaseURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer response.Body.Close()

	fmt.Println("Attempt", attempt, "to get requirements")

	if response.StatusCode != 200 {
                if attempt <= 3 {
                        fmt.Println("Received", response.StatusCode, "going to sleep for one minute")
                        time.Sleep(time.Second * 60)
                        getReleaseRequirements(releaseURL, client, attempt)
                        return
                } else {
                        log.Fatal("Received 429 on attempt 3, exiting")
                }
        }

	var r release
	err = json.NewDecoder(response.Body).Decode(&r)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, re := range r.Requirements {
		capturePackage(re.App, client, 0)
	}
}

// downloadPackage downloads a single package and places its file in
// /hexdump/packages.
func downloadPackage(pkg string, client *http.Client, attempts int) {
	if attempts >= 3 {
		log.Fatal("Exiting after", attempts, "attempts")
	}

	url := "https://repo.hex.pm/packages/" + pkg
	log.Println("Downloading", pkg, "package from", url)
	response, err := client.Get(url)
	if err != nil {
		fmt.Println("Package download resulted in error:", err.Error())
		fmt.Println("Retrying...")
		downloadPackage(pkg, client, attempts+1)
		return
	}
	defer response.Body.Close()

	out, err := os.Create("/hexdump/packages/" + pkg)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func downloadRelease(pkg, version string, client *http.Client, attempts int) {
	if attempts >= 3 {
		log.Fatal("Exiting after", attempts, "attempts")
	}

	// the tarball filename
	release := pkg + "-" + version + ".tar"
	filePath := "/hexdump/tarballs/" + release

	// download the package version tarball
	url := "https://repo.hex.pm/tarballs/" + release
	log.Println("Downloading", release, "from", url)

	response, err := client.Get(url)
	if err != nil {
		fmt.Println("Tarball download resulted in error:", err.Error())
		fmt.Println("Retrying...")
		downloadRelease(pkg, version, client, attempts+1)
		return
	}
	defer response.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		fmt.Println("Timeout reading response body, retrying")
		downloadRelease(pkg, version, client, attempts+1)
		//log.Fatal(err.Error())
	}
}

// downloadRegistry grabs the latest registry file. Mix versions
// <= to 1.4 still use this file.
func downloadRegistry(client *http.Client) {
	url := "https://repo.hex.pm/registry.ets.gz"
	log.Println("Downloading registry from", url)

	response, err := client.Get(url)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer response.Body.Close()

	out, err := os.Create("/hexdump/registry.ets.gz")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func downloadCSV(client *http.Client, tool string, uri string, getInstalls bool) {
	url := "https://repo.hex.pm/installs/" + uri
	log.Println("Downloading", tool, "csv file from", url)

	response, err := client.Get(url)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer response.Body.Close()

	out, err := os.Create("/hexdump/installs/" + uri)
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	out.Close()

	if getInstalls {
		downloadInstalls(client, tool, uri)
	}
}

func downloadInstalls(client *http.Client, tool, csvFile string) {
	f, err := os.Open("/hexdump/installs/" + csvFile)
	if err != nil {
		log.Fatal("Error opening", csvFile, ":", err.Error())
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReader(f))
	log.Println("Parsing", csvFile)

	for {
		record, err := r.Read()
		// Stop at EOF.
		if err == io.EOF {
			log.Println("Reached end of CSV file")
			break
		}

		if len(record) != 3 {
			log.Println("Invalid CSV line", record)
			continue
		}

		version, num := record[0], record[2]
		filename := tool + "-" + version + ".ez"
		url := "https://repo.hex.pm/installs/" + num + "/" + filename
		log.Println("Downloading", filename, "from", url)

		response, err := client.Get(url)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer response.Body.Close()

		out, err := os.Create("/hexdump/installs/" + num + "-" + filename)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer out.Close()

		_, err = io.Copy(out, response.Body)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}
