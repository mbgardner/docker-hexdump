package main

import (
	"bufio"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type pkg struct {
	Name     string    `json:"name"`
	Releases []release `json:"releases"`
}

type release struct {
	Version string `json:"version"`
}

type pkgs []pkg

func main() {
	allPkgs := make([]pkg, 0)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{
		Timeout:   time.Second * 120,
		Transport: tr,
	}

	//initialize the page counter
	i := 1

	for {
		url := "https://hex.pm/api/packages?page=" + strconv.Itoa(i)
		log.Println("Querying packages page" + strconv.Itoa(i) + "@" + url)
		response, err := client.Get(url)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer response.Body.Close()

		var pkgResults pkgs
		err = json.NewDecoder(response.Body).Decode(&pkgResults)
		if err != nil {
			log.Fatal(err.Error())
		}

		// if the body is an empty array then we're done
		if len(pkgResults) == 0 {
			log.Println("Reached end of the results at page", i)
			break
		}

		// append the current page's releases
		for _, p := range pkgResults {
			allPkgs = append(allPkgs, p)
		}

		// increment the page counter
		i++
	}

	log.Println(allPkgs)
	log.Println("Downloading packages and tarballs...")

	for _, p := range allPkgs {
		pkgName := p.Name
		downloadPackage(pkgName, &client)

		for _, r := range p.Releases {
			// hardcoded blacklist b/c of bad file(s)
			if pkgName == "udia" && r.Version == "0.0.1" {
				continue
			}
			downloadRelease(pkgName, r.Version, &client)
		}
	}

	downloadRegistry(&client)
	downloadCSV(&client, "hex", "hex-1.x.csv", true)
	downloadCSV(&client, "hex", "hex-1.x.csv.signed", false)
	downloadCSV(&client, "rebar", "rebar-1.x.csv", true)
	downloadCSV(&client, "rebar", "rebar-1.x.csv.signed", false)
}

// downloadPackage downloads a single package and places its file in
// /hexdump/packages.
func downloadPackage(pkg string, client *http.Client) {
	url := "https://repo.hex.pm/packages/" + pkg
	log.Println("Downloading", pkg, "package from", url)
	response, err := client.Get(url)
	if err != nil {
		log.Fatal(err.Error())
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

func downloadRelease(pkg, version string, client *http.Client) {
	// the tarball filename
	release := pkg + "-" + version + ".tar"
	filePath := "/hexdump/tarballs/" + release

	// download the package version tarball
	url := "https://repo.hex.pm/tarballs/" + release
	log.Println("Downloading", release, "from", url)

	response, err := client.Get(url)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer response.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer out.Close()

	_, err = io.Copy(out, response.Body)
	if err != nil {
		log.Fatal(err.Error())
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
