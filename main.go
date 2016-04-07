package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// test 123

const (
	API_URL      = "https://api.blockcypher.com/v1/ltc/main/txs"
	RPC_PORT     = 9332
	RPC_USERNAME = "user"
	RPC_PASSWORD = "pass"
	RPC_HOST     = "127.0.0.1"
)

type TXInfo struct {
	Addresses     []string `json:"addresses"`
	BlockHeight   int      `json:"block_height"`
	BlockIndex    int      `json:"block_index"`
	Confidence    float64  `json:"confidence"`
	Confirmations int      `json:"confirmations"`
	DoubleSpend   bool     `json:"double_spend"`
	Fees          int      `json:"fees"`
	Hash          string   `json:"hash"`
	Hex           string   `json:"hex"`
	Inputs        []struct {
		Addresses   []string `json:"addresses"`
		Age         int      `json:"age"`
		OutputIndex int      `json:"output_index"`
		OutputValue int      `json:"output_value"`
		PrevHash    string   `json:"prev_hash"`
		Script      string   `json:"script"`
		ScriptType  string   `json:"script_type"`
		Sequence    int      `json:"sequence"`
	} `json:"inputs"`
	LockTime int `json:"lock_time"`
	Outputs  []struct {
		Addresses  []string `json:"addresses"`
		Script     string   `json:"script"`
		ScriptType string   `json:"script_type"`
		Value      int      `json:"value"`
	} `json:"outputs"`
	Preference   string `json:"preference"`
	ReceiveCount int    `json:"receive_count"`
	Received     string `json:"received"`
	RelayedBy    string `json:"relayed_by"`
	Size         int    `json:"size"`
	Total        int    `json:"total"`
	Ver          int    `json:"ver"`
	VinSz        int    `json:"vin_sz"`
	VoutSz       int    `json:"vout_sz"`
}

func SendHTTPGetRequest(url string, jsonDecode bool, result interface{}) (err error) {
	res, err := http.Get(url)

	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		log.Printf("HTTP status code: %d\n", res.StatusCode)
		return errors.New("Status code was not 200.")
	}

	contents, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if jsonDecode {
		err := JSONDecode(contents, &result)

		if err != nil {
			return err
		}
	} else {
		result = &contents
	}

	return nil
}

func BuildURL() string {
	return fmt.Sprintf("http://%s:%s@%s:%d", RPC_USERNAME, RPC_PASSWORD, RPC_HOST, RPC_PORT)
}

func JSONDecode(data []byte, to interface{}) error {
	err := json.Unmarshal(data, &to)

	if err != nil {
		return err
	}

	return nil
}

func EncodeURLValues(url string, values url.Values) string {
	path := url
	if len(values) > 0 {
		path += "?" + values.Encode()
	}
	return path
}

func SendRPCRequest(method, req interface{}) (map[string]interface{}, error) {
	var params []interface{}
	if req != nil {
		params = append(params, req)
	} else {
		params = nil
	}

	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"id":     1,
		"params": params,
	})

	if err != nil {
		return nil, err
	}

	resp, err := http.Post(BuildURL(), "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	if result["error"] != nil {
		errorMsg := result["error"].(map[string]interface{})
		return nil, fmt.Errorf("Error code: %v, message: %v\n", errorMsg["code"], errorMsg["message"])
	}
	return result, nil
}

func GetBlockHeight() (float64, error) {
	result, err := SendRPCRequest("getinfo", nil)
	if err != nil {
		return 0, err
	}
	result = result["result"].(map[string]interface{})
	block := result["blocks"]
	return block.(float64), nil
}

func main() {
	vals := url.Values{}
	vals.Set("limit", "1000")
	vals.Set("includeHex", "true")
	url := EncodeURLValues(API_URL, vals)
	txs := []TXInfo{}
	err := SendHTTPGetRequest(url, true, &txs)

	if err != nil {
		log.Fatal(err)
		return
	}

	currentHeight, err := GetBlockHeight()
	if err != nil {
		log.Fatal(err)
	}

	for {
		blockHeight, err := GetBlockHeight()
		if err != nil {
			log.Fatal(err)
		}

		// wait until a block is found
		if currentHeight != blockHeight {
			log.Printf("New block: %g Prev height: %g\n", blockHeight, currentHeight)
			log.Printf("Got %d transactions.\n", len(txs))
			currentHeight = blockHeight
			counter, errCounter := 0, 0
			for _, x := range txs {
				if x.Confirmations == 0 {
					counter++
					fmt.Printf("Unconfirmed TX: %s Fee: %d Time: %v.\n", x.Hash, x.Fees, x.Received)
					txid, err := SendRPCRequest("sendrawtransaction", x.Hex)
					if err != nil {
						errCounter++
						log.Println(err)
					} else {
						log.Printf("TX %s relayed to network.\n", txid["result"].(string))
					}
				}
			}
			log.Printf("Processed %d unconfirmed transactions. %d transactions were invalid.\n", counter, errCounter)
		} else {
			time.Sleep(1000)
		}
	}
}
