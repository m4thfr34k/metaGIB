/*
Copyright © 2023 Daniel Charpentier <Daniel.Charpentier@gmail.com>
*/

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/portto/solana-go-sdk/client"
	"github.com/portto/solana-go-sdk/common"
	"github.com/portto/solana-go-sdk/program/metaplex/tokenmeta"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func getGenericMetadata(mintfilename string, rpc string, includeImages bool) error {

	// TODO Add progress bar for user to see percent complete

	// Auto retries until all mint list items are found

	const (
		MintInfoFilename      string = "MintInfo.csv"
		MintInfoErrorFilename string = "MintInfoErrors.csv"
		ImageSaveLocation     string = "Images\\"
		ImageFileType         string = ".png"
	)

	var rpcProviderURL = rpc
	var workToDo = true
	var metadataDLerrorCount = 0

	directoryName := ""
	fileNameSep := strings.Index(mintfilename, ".")
	if fileNameSep == -1 {
		directoryName = mintfilename
	} else {
		directoryName = mintfilename[0:fileNameSep]
	}
	if len(directoryName) == 0 {
		directoryName = time.Now().UTC().Format("YYYYMMDDhhmmss")
	}

	err := createDirectory(directoryName)

	if err == nil {
		ProjectMintListMap := make(map[string]int)
		projectMintFilename := ".\\" + mintfilename
		projectMintInfoFilename := ".\\" + directoryName + " - " + MintInfoFilename
		projectMintInfoErrorFilename := ".\\" + directoryName + " - " + MintInfoErrorFilename
		projectImageSaveLocation := ".\\" + directoryName + "\\" + ImageSaveLocation
		ProjectMintListMap = getMintListMap(projectMintFilename)
		MintListTotalCountMap := len(ProjectMintListMap)
		if MintListTotalCountMap > 0 {
			workToDo = true
		} else {
			workToDo = false
		}

		for workToDo {
			var currentIteration = 0
			for k, v := range ProjectMintListMap {
				currentIteration++
				fmt.Println("****************************************************")
				fmt.Println("Answer to the Ultimate Question of Life, the Universe, and Everything:", v)
				fmt.Println("Original total number of items:", MintListTotalCountMap)
				fmt.Println("Working on Mint number ", currentIteration, " out of a current total of ", len(ProjectMintListMap))
				mint := common.PublicKeyFromString(k)
				// TODO Refactor Step 1 - get all metadata accounts first
				metadataAccount, err := tokenmeta.GetTokenMetaPubkey(mint)
				if err != nil {
					fmt.Printf("failed to get metadata account, err: %v\n", err)
					infoErrorLine := mint.String() + ",failed to get metadata account"
					appendErrorResult := appendToMintInfoFile(infoErrorLine, projectMintInfoErrorFilename)
					if appendErrorResult != nil {
						fmt.Printf("Error with append info, err: %v\n", appendErrorResult)
					}
				} else {
					c := client.NewClient(rpcProviderURL)
					// TODO Refactor Step 2 - Use GetMultipleAccounts to get 100 metadata accounts' data at a time
					accountInfo, err := c.GetAccountInfo(context.Background(), metadataAccount.ToBase58())
					if err != nil {
						fmt.Printf("failed to get accountInfo, err: %v\n", err)
						infoErrorLine := mint.String() + ",failed to get accountInfo"
						appendErrorResult := appendToMintInfoFile(infoErrorLine, projectMintInfoErrorFilename)
						if appendErrorResult != nil {
							fmt.Printf("Error with append info, err: %v\n", appendErrorResult)
						}
					} else {
						// TODO Refactor Step 3 - deserialize all metadata accounts
						metadata, err := tokenmeta.MetadataDeserialize(accountInfo.Data)
						if err != nil {
							fmt.Printf("Failed to parse metaAccount, err: %v\n", err)
							infoErrorLine := mint.String() + ",failed to parse metaAccount"
							appendErrorResult := appendToMintInfoFile(infoErrorLine, projectMintInfoErrorFilename)
							if appendErrorResult != nil {
								fmt.Printf("Error with append info, err: %v\n", appendErrorResult)
							}
						} else {
							fmt.Println("")
							fmt.Println("Mint:", metadata.Mint)
							fmt.Println(metadata.Data.SellerFeeBasisPoints)
							fmt.Println("Is mutable:", metadata.IsMutable)
							fmt.Println("Name:", metadata.Data.Name)
							fmt.Println("URI:", metadata.Data.Uri)
							fmt.Println("Symbol:", metadata.Data.Symbol)

							var creatorData = ""
							if metadata.Data.Creators != nil {
								for _, creator := range *metadata.Data.Creators {
									creatorData = creatorData + creator.Address.String() +
										"," + strconv.Itoa(int(creator.Share)) +
										"," + strconv.FormatBool(creator.Verified) + ","
								}
							}

							// TODO Need to move entire get/http meta piece into its own function
							// TODO Refactor Step 4 - Use goroutines & wait groups to download offchain metadata in batches
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
							req, err := retryablehttp.NewRequestWithContext(ctx, "GET", metadata.Data.Uri, nil)
							if err != nil {
								fmt.Printf("Error creating http request, err: %v\n", err)
							} else {
								req.Header.Set("Accept", "application/json")
								req.Header.Set("Content-Type", "application/json")

								resp, err := retryClient.Do(req)
								if err != nil {
									fmt.Println("Error is:", err.Error())
									time.Sleep(100 * time.Millisecond)
									infoErrorLine := mint.String() + ",Error in metadata GET," + metadata.Data.Uri
									appendErrorResult := appendToMintInfoFile(infoErrorLine, projectMintInfoErrorFilename)
									if appendErrorResult != nil {
										fmt.Printf("Error with append info, err: %v\n", appendErrorResult)
									}
								} else {
									if resp.StatusCode != 200 {
										fmt.Println("Received a status code of", resp.StatusCode, "when grabbing the metadata")
									} else {
										// Read the content
										var bodyBytes []byte
										if resp.Body != nil {
											bodyBytes, err = io.ReadAll(resp.Body)
											if err != nil {
												fmt.Printf("Error reading resp body, err: %v\n", err)
											} else {
												// Restore the io.ReadCloser to its original state
												// TODO update everything that uses the body to just use the bodyBytes so I don't have to restore the resp
												resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
												decoder := json.NewDecoder(resp.Body)

												if decoder != nil {
													var metadataForProject GeneralMeta
													err = decoder.Decode(&metadataForProject)
													if err != nil {
														fmt.Printf("Error decoding metadata, err: %v\n", err)
													} else {
														metaURL := metadata.Data.Uri
														fmt.Println("Metadata URL is:", metaURL)
														fmt.Println("Name is:", metadataForProject.Name)
														fmt.Println("Symbol is:", metadataForProject.Symbol)
														fmt.Println("Image URL is:", metadataForProject.Image)

														attributeDataLine := ""
														for counter, kreeValue := range metadataForProject.Attributes {
															fmt.Println("Trait counter:", counter,
																kreeValue.TraitType, " : ",
																string(kreeValue.TraitValue))
															attributeDataLine = attributeDataLine + string(kreeValue.TraitValue) + ","
														}
														// TODO Need to normalize metadata so columns/fields align correctly in saved file
														// TODO Add metadata to map of structs
														// TODO KeyType is NFT address
														// Need to keep track of all unique trait names. Will use this to walk through map values for saving to file

														infoLine := metadata.Mint.String() + "," + metadata.Data.Name + "," + metaURL + "," +
															metadataForProject.Image + "," + metadataForProject.Name + "," + attributeDataLine

														appendResult := appendToMintInfoFile(infoLine, projectMintInfoFilename)
														if appendResult != nil {
															fmt.Printf("Error with append info, err: %v\n", appendResult)
														}

														metadataFileName := ".\\" + directoryName + "\\" + metadata.Mint.String() + ".json"

														// Restore the io.ReadCloser to its original state
														resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
														err = saveFile(metadataFileName, resp)
														if err != nil {
															metadataDLerrorCount++
															fmt.Printf("Error in saving metadata, err: %v\n", err)
															fmt.Println("Total errors so far is:", metadataDLerrorCount)
														} else {
															if includeImages {
																// TODO Refactor Step 5 - Use goroutines & wait groups to download images in batches
																err = createDirectory(projectImageSaveLocation)
																ImagemetaURL := metadataForProject.Image
																fileName := projectImageSaveLocation + metadata.Mint.String() + ImageFileType
																fmt.Println("Image filename will be:", fileName)
																fmt.Println("Getting image from:", ImagemetaURL)
																err = downloadFile(ImagemetaURL, fileName)
																if err != nil {
																	fmt.Println("Error in downloading image:", err.Error())
																	fmt.Println("File:", fileName)
																	infoErrorLine := mint.String() + ",Error in downloading image," + err.Error() + "," + metadataForProject.Image
																	appendErrorResult := appendToMintInfoFile(infoErrorLine, projectMintInfoErrorFilename)
																	if appendErrorResult != nil {
																		fmt.Printf("Error with append info, err: %v\n", appendErrorResult)
																	}
																} else {
																	fmt.Println("Successfully grabbed metadata AND image")
																	delete(ProjectMintListMap, k)
																}
															} else {
																fmt.Println("Successfully grabbed metadata")
																delete(ProjectMintListMap, k)
															}
															fmt.Println("***************************************************")
															time.Sleep(200 * time.Millisecond)
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
				if len(ProjectMintListMap) > 0 {
					workToDo = true
				} else {
					workToDo = false
				}
			}
		}
		return nil
	} else {
		return errors.New("unable to create folder for images")
	}
}
