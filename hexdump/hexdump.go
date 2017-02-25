package main

import (
	"crypto/tls"
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

	// initialize the page counter
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
			downloadRelease(pkgName, r.Version, &client)
		}
	}

	downloadRegistry(&client)
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
