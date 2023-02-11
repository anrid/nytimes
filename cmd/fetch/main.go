package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

const (
	NYTimesArchiveAPIURL = "https://api.nytimes.com/svc/archive/v1/%d/%d.json?api-key="
)

func main() {
	year := pflag.Int("year", 0, "Start fetching NY Times articles from this year (YYYY, integer, required)")
	month := pflag.Int("month", 0, "Start fetching NY Times articles from this month (M, integer, required)")

	pflag.Parse()

	key := os.Getenv("SHINY_NYTIMES_API_KEY")
	if key == "" {
		log.Panicf("env var SHINY_NYTIMES_API_KEY not set")
	}

	if *year == 0 || *month == 0 {
		pflag.Usage()
		os.Exit(-1)
	}

	thisYear, _ := strconv.Atoi(time.Now().Format("2006"))
	thisMonth, _ := strconv.Atoi(time.Now().Format("1"))

	for {
		outfile := fmt.Sprintf("data/articles-%d-%d.json.gz", *year, *month)
		var fileExists bool
		var fileExistsAndIsValid bool
		var performedAPICall bool
		retries := 3

		stat, err := os.Stat(outfile)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Panic(err)
			}
		} else {
			fileExists = true
		}

		fileExistsAndIsValid = fileExists && stat.Size() > 1_000

		if !fileExistsAndIsValid {
			url := fmt.Sprintf(NYTimesArchiveAPIURL, *year, *month)

			var prettyJ []byte

			for ; retries > 0; retries-- {
				fmt.Printf("Fetching URL: %s\n", url)

				resp, err := http.Get(url + key)
				if err != nil {
					log.Panic(err)
				}

				performedAPICall = true

				data, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Panic(err)
				}

				j := make(map[string]interface{})
				err = json.Unmarshal(data, &j)
				if err != nil {
					log.Panic(err)
				}

				prettyJ, err = json.MarshalIndent(j, "", "  ")
				if err != nil {
					log.Panic(err)
				}

				if len(prettyJ) < 1_000 {
					if strings.Contains(string(prettyJ), "\"docs\": [],") {
						fmt.Println("No docs returned, skipping this year and month!")
						break
					}

					if strings.Contains(string(prettyJ), "policies.ratelimit.QuotaViolation") {
						fmt.Printf("Rate limited! Sleeping for 6 sec before trying again (%d retries left) ..\n", retries)
						time.Sleep(6 * time.Second)
						continue
					}

					log.Panicf("Got some kind of error:\n%s\n\n", prettyJ)
				}

				// Write JSON to a compressed file.
				o, err := os.Create(outfile)
				if err != nil {
					log.Panic(err)
				}

				gw := gzip.NewWriter(o)

				n, err := gw.Write(prettyJ)
				if err != nil {
					log.Panic(err)
				}

				gw.Close()
				o.Close()

				fmt.Printf("Wrote %d / %d compressed bytes to file %s\n", n, len(prettyJ), outfile)
				break
			}
		}

		// Check if we're all caught up in time!
		if *year == thisYear && *month == thisMonth {
			fmt.Printf("We're at %d-%d - we're all caught up in time! Donezo!\n", *year, *month)
			break
		} else {
			*month++
			if *month > 12 {
				*month = 1
				*year++
			}
		}

		if performedAPICall {
			fmt.Printf("Sleeping for 6 sec to avoid rate limit ..\n")
			time.Sleep(6 * time.Second)
		}
	}
}
