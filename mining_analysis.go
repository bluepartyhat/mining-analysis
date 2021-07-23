package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bitclout/backend/routes"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

func main() {
	flagOutputCSVFileName := flag.String("output_csv_file", "", "An optional csv output" +
		" file where sorted data on public key -> # of blocks mined is stored.")
	flagNode := flag.String("node", "https://api.bitclout.com",
		"Specifies the node from which to collect data.")
	flagStartingBlockHeight := flag.Int("starting_block_height", -1,
		"Specifies the block height to start at when collecting data. The script will step backwards from this" +
		" starting block node to max(genesis, starting_block_height - blocks_to_collect) collecting data along the way. " +
		"If this flag is not set, the script will set starting_block_height to the consensus tip.")
	flagBlocksToCollect := flag.Int("blocks_to_collect", 1000, "Specifies the number of blocks" +
		"to collect moving backwards from --starting_block_height.")
	flagDelayMilliseconds:= flag.Int("delay_milliseconds", 1500,
		"The delay in milliseconds to wait between failed requests.")
	flag.Parse()

	// Process flags.
	outputToFile := false
	outputCSVFileName := *flagOutputCSVFileName
	var csvOutputWriter *csv.Writer
	if *flagOutputCSVFileName != "" {
		outputToFile = true
		outputCSVFileName = *flagOutputCSVFileName

		// Open output CSV file.
		csvOutputFile, err := os.OpenFile(outputCSVFileName, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil { panic(errors.Wrap(err, "main() failed to open the specified output file")) }
		defer csvOutputFile.Close()
		csvOutputWriter = csv.NewWriter(csvOutputFile)
	}
	node := *flagNode
	startingBlockHeight := int64(*flagStartingBlockHeight)
	timeDelayMilliseconds := *flagDelayMilliseconds
	blockToCollect := *flagBlocksToCollect

	// Collect the starting block from the explorer.
	blocksCollected := 0
	var startingBlock *routes.APIBlockResponse
	var err error
	if startingBlockHeight != -1 {
		startingBlockRequest := routes.APIBlockRequest{
			Height:    startingBlockHeight,
			HashHex:   "",
			FullBlock: true,
		}
		startingBlock, err = GetBlockResponseForBlockRequest(&startingBlockRequest, node)
		for err != nil {
			fmt.Println(errors.Wrap(err, "main() failed to fetch the starting block from the server"))
			time.Sleep(time.Millisecond * time.Duration(timeDelayMilliseconds))
			startingBlock, err = GetBlockResponseForBlockRequest(&startingBlockRequest, node)
		}
	} else {
		startingBlock, err = GetBlockResponseForTip(node)
		for err != nil {
			fmt.Println(errors.Wrap(err, "main() failed to fetch the starting block from the server"))
			time.Sleep(time.Millisecond * time.Duration(timeDelayMilliseconds))
			startingBlock, err = GetBlockResponseForTip(node)
		}
	}
	fmt.Printf("Got starting block with hash: %s height: %d\n", startingBlock.Header.BlockHashHex, startingBlock.Header.Height)
	blocksCollected++

	// Collect the miner information from the block.
	miners, err := GetMinersFromBlockResponse(startingBlock)
	if err != nil { panic(errors.Wrap(err, "main() found an invalid block")) }
	pkToBlocksMined := make(map[string]uint64)
	for _, miner := range miners {
		pkToBlocksMined[miner] = 1
	}

	// Begin walking backward until the genesis block is hit or we've collected enough blocks.
	prevHeader := startingBlock.Header
	for prevHeader.Height != 0 && blockToCollect != blocksCollected {
		fmt.Printf("Starting block %d of %d\n", blocksCollected + 1, blockToCollect)
		currentBlockRequest := routes.APIBlockRequest{
			Height:    0,
			HashHex:   prevHeader.PrevBlockHashHex,
			FullBlock: true,
		}
		currentBlockResponse, err := GetBlockResponseForBlockRequest(&currentBlockRequest, node)

		// Continue trying until the error is resolved waiting between requests in case of rate limiting.
		for err != nil {
			fmt.Println(errors.Errorf("main() failed to get response for block request: %v", err))
			time.Sleep(time.Millisecond * time.Duration(timeDelayMilliseconds))
			currentBlockResponse, err = GetBlockResponseForBlockRequest(&currentBlockRequest, node)
		}

		// Add the results to the pkToBlocksMined map.
		minerPublicKeys, err := GetMinersFromBlockResponse(currentBlockResponse)
		minerSeenThisBlock := make(map[string]interface{})
		if err != nil { panic(errors.Wrap(err, "main() found an invalid block")) }
		for _, minerPk := range minerPublicKeys {
			if _, minerAlreadySeen := minerSeenThisBlock[minerPk]; minerAlreadySeen {
				continue
			} else {
				minerSeenThisBlock[minerPk] = struct{}{}
			}

			if _, keyExists := pkToBlocksMined[minerPk]; keyExists {
				pkToBlocksMined[minerPk] = pkToBlocksMined[minerPk] + 1
			} else {
				pkToBlocksMined[minerPk] = 1
			}
		}
		fmt.Printf("Got block with hash: %s height: %d\n", currentBlockResponse.Header.BlockHashHex, currentBlockResponse.Header.Height)

		// Setup for next block.
		prevHeader = currentBlockResponse.Header
		blocksCollected++
		time.Sleep(time.Millisecond * time.Duration(timeDelayMilliseconds))
	}

	// Sort and print the results to stdout.
	var minerPublicKeys []string
	for pk := range pkToBlocksMined {
		minerPublicKeys = append(minerPublicKeys, pk)
	}
	sort.Slice(minerPublicKeys, func(ii, jj int) bool {
		return pkToBlocksMined[minerPublicKeys[ii]] > pkToBlocksMined[minerPublicKeys[jj]]
	})
	for _, pk := range minerPublicKeys {
		fmt.Println(pk, pkToBlocksMined[pk])
	}

	// Output to a file if specified.
	if outputToFile && csvOutputWriter != nil {
		for _, pk := range minerPublicKeys {
			// Construct the entry.
			var outputEntry []string
			outputEntry = append(outputEntry, pk)
			outputEntry = append(outputEntry, strconv.FormatInt(int64(pkToBlocksMined[pk]), 10))

			// Write the output to the csv writer.
			err = csvOutputWriter.Write(outputEntry)
			if err != nil {
				panic(errors.Wrap(err, "main() failed to write to the csv file"))
			}
		}
		csvOutputWriter.Flush()
	}
}

func GetMinersFromBlockResponse(blockResponse *routes.APIBlockResponse) (_miners []string, _err error) {
	// Get the first transaction in the slice.
	blockRewardTxn := blockResponse.Transactions[0]
	if blockRewardTxn.TransactionType != "BLOCK_REWARD" {
		return []string{}, errors.Errorf("GetMinersFromBlockResponse() block with height %d has non BLOCK_REWARD " +
			"first transaction", blockResponse.Header.Height)
	}

	// Get the miners from the block reward txn.
	var miners []string
	for _, minerOutput := range blockRewardTxn.Outputs {
		miners = append(miners, minerOutput.PublicKeyBase58Check)
	}
	return miners, nil
}

func GetBlockResponseForBlockRequest(blockRequest *routes.APIBlockRequest, node string) (_response *routes.APIBlockResponse, _err error) {
	endpoint := node + routes.RoutePathAPIBlock

	// Package request.
	postBody, err := json.Marshal(*blockRequest)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockResponseForBlockRequest() failed to marshal json")
	}
	postBuffer := bytes.NewBuffer(postBody)

	// Execute request.
	resp, err := http.Post(endpoint, "application/json", postBuffer)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockResponseForBlockRequest() failed to execute request")
	}
	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.Errorf("GetBlockResponseForBlockRequest(): Received non 200 response code: " +
			"Status Code: %v Body: %v", resp.StatusCode, string(bodyBytes))
	}

	// Process response.
	blockResponse := routes.APIBlockResponse{}
	err = json.NewDecoder(resp.Body).Decode(&blockResponse)
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		outputError := fmt.Sprintf("GetBlockResponseForBlockRequest(): failed decoding body: %s", string(bodyBytes))
		return nil, errors.Wrap(err, outputError)
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockResponseForBlockRequest(): failed closing body")
	}
	return &blockResponse, nil
}

func GetBlockResponseForTip(node string) (_response *routes.APIBlockResponse, _err error) {
	endpoint := node + routes.RoutePathAPIBase

	// Execute request.
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockResponseForTip() failed to execute request")
	}
	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.Errorf("GetBlockResponseForBlockRequest(): Received non 200 response code: " +
			"Status Code: %v Body: %v", resp.StatusCode, string(bodyBytes))
	}

	// Process response.
	blockResponse := routes.APIBlockResponse{}
	err = json.NewDecoder(resp.Body).Decode(&blockResponse)
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		outputError := fmt.Sprintf("GetBlockResponseForBlockRequest(): failed decoding body: %s", string(bodyBytes))
		return nil, errors.Wrap(err, outputError)
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "GetBlockResponseForBlockRequest(): failed closing body")
	}
	return &blockResponse, nil
}

