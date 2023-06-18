/*
Copyright © 2023 Daniel Charpentier <Daniel.Charpentier@gmail.com>
*/
package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type GeneralMeta struct {
	Name        string              `json:"name"`
	Symbol      string              `json:"symbol"`
	Description string              `json:"description"`
	Image       string              `json:"image"`
	ExtURL      string              `json:"external_url"`
	Attributes  []GeneralAttributes `json:"attributes"`
}

type GeneralAttributes struct {
	TraitType  string          `json:"trait_type"`
	TraitValue json.RawMessage `json:"value"`
}

func createDirectory(mintfilename string) error {

	path := ".\\" + mintfilename
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else {
		// already exists
		return nil
	}
}

func downloadFile(URL, fileName string) error {

	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMin = time.Second
	retryClient.RetryWaitMax = time.Second * 10
	retryClient.RetryMax = 10
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}

		// My retry logic here
		if (resp != nil && resp.StatusCode == 429) || (resp == nil) {
			ctx.Done()
			return false, nil
		}

		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", URL, nil)

	if err != nil {
		return err
	}

	resp, err := retryClient.Do(req)
	if err != nil {
		return err
	} else {
		if resp.StatusCode != 200 {
			return errors.New("received non 200 response code")
		} else {
			file, err := os.Create(fileName)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(file, resp.Body)
			if err != nil {
				return err
			}

			return nil
		}
	}
}

func saveFile(fileName string, resp *http.Response) error {

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func getMintListMap(fileNameMintList string) map[string]int {

	mintListMap := make(map[string]int)

	file, err := os.Open(fileNameMintList)
	if err != nil {
		log.Fatalf("Error while opening file. Err: %s", err)
	}
	defer file.Close()

	// read the file line by line using scanner
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		mintListMap[scanner.Text()] = 42
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return mintListMap
}

func createMintFile(MintInfoFileName string) error {

	f, err := os.Create(MintInfoFileName)
	if err != nil {
		// Waiting due to error
		time.Sleep(1 * time.Second)
		return err
	} else {
		// TODO update header so attribute list is dynamic
		_, err := f.WriteString("TokenID,Collection,MetadataURL,ImageURL,Name,Attribute01,Attribute02,Attribute03,Attribute04,Attribute05,Attribute06,Attribute07,Attribute08,Attribute09,Attribute10\n")
		if err != nil {
			// Waiting due to error
			time.Sleep(1 * time.Second)
			return err
		} else {
			err = f.Close()
			if err != nil {
				// Waiting due to error
				time.Sleep(1 * time.Second)
				return err
			}
			return nil
		}
	}
}

func appendToMintInfoFile(mintInfoLine, mintInfoFile string) error {

	var mintFileNameFull string
	var startTime time.Time

	timeLocation, err := time.LoadLocation("America/New_York")
	if err != nil {
		startTime = time.Now()
	} else {
		startTime = time.Now().In(timeLocation)
	}

	mintFileNameFull = mintInfoFile + " - " + startTime.Format("2006-01-02") + ".csv"

	_, err = os.Stat(mintFileNameFull)
	if os.IsNotExist(err) {
		createResult := createMintFile(mintFileNameFull)
		if createResult != nil {
			// TODO What should happen if the file isnt created
		} else {
			// TODO what should happen if file is created
		}
	}

	f, err := os.OpenFile(mintFileNameFull, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// TODO what should happen if file isnt opened
		return err
	} else {
		_, err = fmt.Fprintln(f, mintInfoLine)
		return nil
	}
}
