package polkadot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const (
	testNodeAPI = "http://127.0.0.1:8080"
)

func PrintJsonLog(t *testing.T, logCont string) {
	if strings.HasPrefix(logCont, "{") {
		var str bytes.Buffer
		_ = json.Indent(&str, []byte(logCont), "", "    ")
		t.Logf("Get Call Result return: \n\t%+v\n", str.String())
	} else {
		t.Logf("Get Call Result return: \n\t%+v\n", logCont)
	}
}

func TestGetCall(t *testing.T) {
	tw := NewClient(testNodeAPI, true)

	if r, err := tw.GetCall("/metadata/"); err != nil {
		t.Errorf("Get Call Result failed: %v\n", err)
	} else {
		PrintJsonLog(t, r.String())
	}
}

func Test_getBlockHeight(t *testing.T) {

	c := NewClient(testNodeAPI, true)

	r, err := c.getBlockHeight()

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("height:", r)
	}

}

func Test_getBlockByHeight(t *testing.T) {
	c := NewClient(testNodeAPI, true)
	r, err := c.getBlockByHeight(1830393)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(r)
	}
}

func Test_getBalance(t *testing.T) {

	c := NewClient(testNodeAPI, true)

	address := "CyVGZAGjfD9bbQcN7Ja7cFrazM4yAzawRhKcWYN1ZmMrpwf"

	r, err := c.getBalance(address, true, 20000000)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(r)
	}

	address = "J5cTdWxoMRZyQHyvvDMxB6dp7YitNpzEkj3ZrJFsGmARcC2"

	r, err = c.getBalance(address, true, 20000000)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(r)
	}
}

func Test_sendTransaction(t *testing.T) {
	c := NewClient(testNodeAPI, true)
	r, err := c.sendTransaction("0x39028453538d40098561df3a2fe577c2995d7f6a5bd45f0f5708e9b9e11cc4896e1db800ba6d5" +
		"4c43a5c07e6d1b89ff51d356ac686b0bcd592d00a5140c4842a054c567830aad78e0bd03c86792e794756a516507b1911f824267d51b75" +
		"b3676d0a9b40295030000050009d0dbc83629dcd90f7e1cc989cd2cd713205adff4a3b0f2699e31b7e35981340700dc5c2402")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(r)
	}
}
