/*
 * Copyright 2018 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */
package polkadot

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/tidwall/gjson"
)

const BATCH_CHARGE_TO_TAG = "batch_charge"

type Block struct {
	Hash          string         `json:"block"`         // actually block signature in XRP chain
	PrevBlockHash string         `json:"previousBlock"` // actually block signature in DOT chain
	Timestamp     uint64         `json:"timestamp"`
	Height        uint64         `json:"height"`
	Transactions  []*Transaction `json:"transactions"`
}

type Transaction struct {
	TxID        string
	Fee         uint64
	TimeStamp   uint64
	From        string
	To          string
	Amount      uint64
	BlockHeight uint64
	BlockHash   string
	Status      string
	ToArr       []string //@required 格式："地址":"数量"
	ToDecArr    []string //@required 格式："地址":"数量(带小数)"
}

func GetTransactionInBlock(json *gjson.Result) []*Transaction {
	var (
		blockHash    = gjson.Get(json.Raw, "hash").String()
		blockHeight  = gjson.Get(json.Raw, "number").Uint()
		transactions = []*Transaction{}
	)

	getMethod := func(result gjson.Result) string {
		return fmt.Sprintf("%s.%s", result.Get("method.pallet").String(), result.Get("method.method").String())
	}

	blockTime := uint64(time.Now().Unix())

	for _, extrinsic := range gjson.Get(json.Raw, "extrinsics").Array() {
		method := getMethod(extrinsic)
		success := gjson.Get(extrinsic.Raw, "success").Bool()

		// 过滤失败交易
		if !success {
			continue
		}

		// 获取这个区块的时间
		if method == "timestamp.set" {
			blockTime = extrinsic.Get("args.now").Uint()
		}

		// 解析批量转账
		if method == "utility.batch" {
			batchTxs := []*Transaction{}
			txHash := extrinsic.Get("hash").String()
			txFrom := extrinsic.Get("signature.signer").String()
			calls := extrinsic.Get("args.calls").Array()
			toAmount := uint64(0)
			toArray := []string{}
			for _, call := range calls {

				method := getMethod(call)
				to := call.Get("args.dest").String()
				value := call.Get("args.value").Uint()
				if method == "balances.transfer" {
					batchTxs = append(batchTxs, &Transaction{
						TxID:        txHash,
						TimeStamp:   blockTime,
						From:        txFrom,
						To:          to,
						Amount:      value,
						BlockHash:   blockHash,
						BlockHeight: blockHeight,
						Status:      "-1",
					})
				}
			}

			// 比对交易事件是否匹配的上， 确定交易状态
			for _, event := range extrinsic.Get("events").Array() {
				eventMethod := getMethod(event)
				if eventMethod == "balances.Transfer" {
					datas := event.Get("data").Array()
					from := datas[0].String()
					to := datas[1].String()
					value := datas[2].Uint()
					for i, tx := range batchTxs {
						if tx.Status == "-1" {
							if tx.From != from {
								continue
							}
							if tx.To != to {
								continue
							}
							if tx.Amount != value {
								continue
							}
							batchTxs[i].Status = "1"
							toAmount, _ = math.SafeAdd(toAmount, value)
							toArray = append(toArray, to)
						}
					}
				}
			}

			fee := uint64(0)

			tip := uint64(gjson.Get(extrinsic.Raw, "tip").Uint())

			info := gjson.Get(extrinsic.Raw, "info")
			if info.Exists() {
				partialFee := uint64(gjson.Get(info.Raw, "partialFee").Uint())

				fee, _ = math.SafeAdd(tip, partialFee)
			}

			for _, batchTransactionItem := range batchTxs {
				transactions = append(transactions, &Transaction{
					TxID:        batchTransactionItem.TxID,
					Fee:         fee,
					TimeStamp:   batchTransactionItem.TimeStamp,
					From:        batchTransactionItem.From,
					To:          batchTransactionItem.To,
					Amount:      toAmount,
					BlockHeight: batchTransactionItem.BlockHeight,
					BlockHash:   batchTransactionItem.BlockHash,
					Status:      batchTransactionItem.Status,
					ToArr:       toArray,
				})
				break
			}

			continue
		}

		if method != "balances.transfer" && method != "claims.attest" && method != "balances.transferKeepAlive" { //不是这个method的全部不要
			continue
		}

		argsTo := ""        //检测到的接收地址
		argsAmountStr := "" //检测到的接收金额
		from := ""          //来源地址
		to := ""            //目标地址
		amountStr := ""     //金额
		args := gjson.Get(extrinsic.Raw, "args")
		if len(args.Raw) > 0 {
			argsTo = gjson.Get(args.Raw, "dest").String()
			argsAmountStr = gjson.Get(args.Raw, "value").String()
		}

		for _, event := range gjson.Get(extrinsic.Raw, "events").Array() {
			method := getMethod(event)
			if method == "balances.Transfer" {
				data := gjson.Get(event.Raw, "data").Array()
				if len(data) == 3 {
					from = data[0].String()
					to = data[1].String()
					amountStr = data[2].String()
				}
			}
			if method == "claims.Claimed" {
				data := gjson.Get(event.Raw, "data").Array()
				if len(data) == 3 {
					//from = data[1].String()
					to = data[0].String()
					amountStr = data[2].String()
				}
			}
		}

		if argsTo == "" && to == "" { //没有取到值
			continue
		}
		if argsAmountStr == "" && amountStr == "" { //没有取到值
			continue
		}
		if method == "balances.transfer" && argsTo != to { //值不一样
			continue
		}
		if method == "balances.transfer" && argsAmountStr != amountStr { //值不一样
			continue
		}

		txid := gjson.Get(extrinsic.Raw, "hash").String()

		fee := uint64(0)

		tip := uint64(gjson.Get(extrinsic.Raw, "tip").Uint())

		info := gjson.Get(extrinsic.Raw, "info")
		if info.Exists() {
			partialFee := uint64(gjson.Get(info.Raw, "partialFee").Uint())

			fee, _ = math.SafeAdd(tip, partialFee)
		}

		amountInt, err := strconv.ParseInt(amountStr, 10, 64)
		if err == nil {
			amount := uint64(amountInt)

			transactions = append(transactions, &Transaction{
				TxID:        txid,
				Fee:         fee,
				TimeStamp:   blockTime,
				From:        from,
				To:          to,
				Amount:      amount,
				BlockHeight: blockHeight,
				BlockHash:   blockHash,
				Status:      "1",
			})
		}
	}

	return transactions
}

func NewBlock(json *gjson.Result) *Block {
	obj := &Block{}
	// 解析
	obj.Hash = gjson.Get(json.Raw, "hash").String()
	obj.PrevBlockHash = gjson.Get(json.Raw, "parentHash").String()
	obj.Height = gjson.Get(json.Raw, "number").Uint()
	obj.Transactions = GetTransactionInBlock(json)

	if obj.Hash == "" {
		time.Sleep(5 * time.Second)
	}
	return obj
}

//BlockHeader 区块链头
func (b *Block) BlockHeader() *openwallet.BlockHeader {

	obj := openwallet.BlockHeader{}
	//解析json
	obj.Hash = b.Hash
	//obj.Confirmations = b.Confirmations
	obj.Previousblockhash = b.PrevBlockHash
	obj.Height = b.Height
	//obj.Symbol = Symbol

	return &obj
}

type AddrBalance struct {
	Address string
	Balance *big.Int
	Free    *big.Int
	Freeze  *big.Int
	Nonce   uint64
	index   int
	Actived bool
}

type TxArtifacts struct {
	Hash        string
	Height      int64
	GenesisHash string
	SpecVersion uint32
	Metadata    string
	TxVersion   uint32
	ChainName   string
}

func GetTxArtifacts(json *gjson.Result) *TxArtifacts {
	obj := &TxArtifacts{}

	obj.Hash = gjson.Get(json.Raw, "at").Get("hash").String()
	obj.Height = gjson.Get(json.Raw, "at").Get("height").Int()
	obj.GenesisHash = gjson.Get(json.Raw, "genesisHash").String()
	obj.SpecVersion = uint32(gjson.Get(json.Raw, "specVersion").Uint())
	obj.Metadata = gjson.Get(json.Raw, "metadata").String()
	obj.TxVersion = uint32(gjson.Get(json.Raw, "txVersion").Uint())
	obj.ChainName = gjson.Get(json.Raw, "chainName").String()

	return obj
}
