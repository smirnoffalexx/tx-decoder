package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

type DetectorStatus int

const (
	CatchingUpToNewBlocks DetectorStatus = iota
	ListeningForNewHeads
)

type DetectorState struct {
	StartBlockTime    time.Time      `json:"start_block_time"`
	StartBlockNumber  uint64         `json:"start_block_number"`
	PrevBlockTime     time.Time      `json:"prev_block_time"`
	PrevBlockNumber   uint64         `json:"prev_block_number"`
	LastBlockTime     time.Time      `json:"last_block_time"`
	LastBlockNumber   uint64         `json:"last_block_number"`
	Status            DetectorStatus `json:"status"`
	CatchupBlocksLeft int            `json:"catchup_blocks_left"`
}

type DetectorSettings struct {
	LastProcessedBlock  int64  `json:"last_processed_block"`
	LastProcessedTxHash string `json:"last_processed_tx_hash"`
}

var DETECTOR_STATE DetectorState
var DETECTOR_SETTINGS DetectorSettings
var UNISWAP_POOLS []string
var POOL_TOPICS map[string]string

func pools() []string {
	ethUSDCPool := "0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640"
	ethUSDTPool := "0x11b815efB8f581194ae79006d24E0d814B7697F6"

	return []string{ethUSDCPool, ethUSDTPool}
}

func topics() {
	swapEvent := "Swap(address,address,int256,int256,uint160,uint128,int24)"
	mintEvent := "Mint(address,address,int24,int24,uint128,uint256,uint256)"
	burnEvent := "Burn(address,int24,int24,uint128,uint256,uint256)"

	swapTopic := crypto.Keccak256([]byte(swapEvent))
	mintTopic := crypto.Keccak256([]byte(mintEvent))
	burnTopic := crypto.Keccak256([]byte(burnEvent))

	fmt.Printf("swapTopic: 0x%x\n", swapTopic)
	fmt.Printf("mintTopic: 0x%x\n", mintTopic)
	fmt.Printf("burnTopic: 0x%x\n", burnTopic)

	swapTopicString := "0x" + hex.EncodeToString(swapTopic)
	mintTopicString := "0x" + hex.EncodeToString(mintTopic)
	burnTopicString := "0x" + hex.EncodeToString(burnTopic)

	POOL_TOPICS = make(map[string]string)
	POOL_TOPICS[swapTopicString] = swapEvent
	POOL_TOPICS[mintTopicString] = mintEvent
	POOL_TOPICS[burnTopicString] = burnEvent
}

func detectBlocks() {
	UNISWAP_POOLS = pools()
	topics()
	defer WG.Done()

	log.Info().Msg("Enter detectBlocks")

	currentBlock, err := CLIENT.BlockNumber(context.Background())
	if err != nil {
		return
	}

	DETECTOR_SETTINGS = DetectorSettings{
		LastProcessedBlock: int64(currentBlock), // 17013950,
	}

	detectorError := make(chan bool)

	if err := catchupToNewBlocks(); err != nil {
		log.Error().Err(err).Msg("")
		os.Exit(1)
	}

	go subscribeNewHead(detectorError)

	for {
		select {
		case <-detectorError:
			log.Error().Msg("Closing CLIENT and resubscribing")
			CLIENT.Close()
			time.Sleep(time.Millisecond * 200)
			CLIENT = connectEthereum()

			go subscribeNewHead(detectorError)
		}
	}
}

func subscribeNewHead(detectorError chan bool) {
	DETECTOR_STATE.Status = ListeningForNewHeads

	log.Info().Msg("Subscribed to new block heads")

	headers := make(chan *types.Header)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sub, err := CLIENT.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Error().Err(err).Msg("")
		sub.Unsubscribe()
		close(headers)
		detectorError <- true
		return
	}

	for {
		select {
		case err := <-sub.Err():
			log.Error().Err(err).Msg("")
			sub.Unsubscribe()
			close(headers)
			detectorError <- true
			return
		case header := <-headers:
			if err := processBlockManager(header.Number); err != nil {
				log.Error().Err(err).Msg("")
			}
		}
	}
}

func catchupToNewBlocks() error {
	const hourWorthOfBlocks = 60 * 60 / 11
	DETECTOR_STATE.Status = CatchingUpToNewBlocks

	DETECTOR_STATE.LastBlockNumber = uint64(DETECTOR_SETTINGS.LastProcessedBlock)

	isDone := false

	for !isDone {
		log.Info().Msg("Entering catching up cycle")

		currentBlock, err := CLIENT.BlockNumber(context.Background())
		if err != nil {
			return err
		}

		oldestBlock := currentBlock - hourWorthOfBlocks
		if DETECTOR_STATE.LastBlockNumber+1 > oldestBlock {
			oldestBlock = DETECTOR_STATE.LastBlockNumber + 1
		}

		for i := oldestBlock; i < currentBlock+1; i++ {
			if err := processBlockManager(new(big.Int).SetUint64(i)); err != nil {
				return err
			}

			DETECTOR_STATE.CatchupBlocksLeft = int(currentBlock - i)

			log.Info().Msg("Catching up: " + strconv.Itoa(int(i)) + "/" + strconv.Itoa(int(currentBlock)) +
				". Left: " + strconv.Itoa(int(currentBlock-i)))
		}

		currentBlock, err = CLIENT.BlockNumber(context.Background())
		if err != nil {
			return err
		}

		if currentBlock == DETECTOR_STATE.LastBlockNumber {
			isDone = true
		}
	}

	return nil
}

func processBlockManager(number *big.Int) error {
	log.Info().Msg("processBlockManager:" + number.String())
	// start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	block, err := CLIENT.BlockByNumber(ctx, number)
	if err != nil {
		log.Error().Err(err).Msg("")
		return err
	}
	// fmt.Println("BlockNumber:", block.Number().Int64(), "Txs:", block.Transactions(), "blockHash:", block.Hash())
	DETECTOR_SETTINGS.LastProcessedBlock = number.Int64()
	DETECTOR_STATE.LastBlockNumber = number.Uint64()

	if err := processTxs(block.Transactions()); err != nil {
		return err
	}

	return nil
}

func processTxs(txs types.Transactions) error {
	for _, tx := range txs {
		receipt, err := CLIENT.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			return err
		}

		logs := receipt.Logs

		for _, txLog := range logs {
			if !Contains(UNISWAP_POOLS, txLog.Address.String()) {
				continue
			}

			fmt.Println("topics:", txLog.Topics)
			fmt.Println("logData:", hex.EncodeToString(txLog.Data))

			zeroTopic := txLog.Topics[0].String()

			if !Contains(mapKeysToSlice(POOL_TOPICS), zeroTopic) {
				continue
			}

			eventName, _ := parseEvent(POOL_TOPICS[zeroTopic])
			// params := []string{}
			params := make(map[string]string)

			if eventName == "Swap" {
				params["sender"] = "0x" + txLog.Topics[1].String()[26:]
				params["recepient"] = "0x" + txLog.Topics[2].String()[26:]
				// params = append(params, "0x"+txLog.Topics[1].String()[26:]) // sender
				// params = append(params, "0x"+txLog.Topics[2].String()[26:]) // recepient
				// decode int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick
			}

			if eventName == "Mint" {
				params["owner"] = "0x" + txLog.Topics[1].String()[26:]
				// params = append(params, "0x"+txLog.Topics[1].String()[26:]) // owner
				// params = append(params, "0x"+txLog.Topics[2].String())      // tickLower int24
				// params = append(params, "0x"+txLog.Topics[3].String())      // tickUpper int24
				// decode address sender, uint128 amount, uint256 amount0, uint256 amount1
			}

			if eventName == "Burn" {
				params["owner"] = "0x" + txLog.Topics[1].String()[26:]
				// params = append(params, "0x"+txLog.Topics[1].String()[24:]) // owner
				// params = append(params, "0x"+txLog.Topics[2].String())      // tickLower int24
				// params = append(params, "0x"+txLog.Topics[3].String())      // tickUpper int24
				// decode uint128 amount, uint256 amount0, uint256 amount1
			}

			paramsJSON, err := json.Marshal(params)
			if err != nil {
				log.Error().Err(err)
				return err
			}

			log.Info().Msg(eventName + ", txHash: " + tx.Hash().String() + ", contarctAddress: " +
				txLog.Address.String() + ", params: " + string(paramsJSON)) // strings.Join(params, ", "))
		}
	}

	return nil
}

func decodeLogData(args []string, hexData string) []string {
	fmt.Println("logData:", hexData)
	startIndex := 0
	data := []string{}
	for _, arg := range args {
		param := hexData[startIndex : startIndex+64]
		fmt.Println(param)

		if arg == "address" {
			param = "0x" + param[20:]
		} else {

		}

		data = append(data, param)
		startIndex = startIndex + 64
	}

	return data
}
