package main

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var PRIVATE_KEY *ecdsa.PrivateKey
var CLIENT *ethclient.Client
var CHAIN_ID int64 = 1
var WG sync.WaitGroup

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Error().Msg("Can't load env")
		os.Exit(1)
	}

	// ConfigRuntime()

	// init logger
	initLogger()

	log.Info().Msg("Log process started")

	PRIVATE_KEY, err = crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		log.Error().Msg("Can't get private key")
	}

	// create eth client
	CLIENT = connectEthereum()

	time.Sleep(time.Second * 2)

	// create swap tx
	go createTransaction()

	time.Sleep(time.Second * 2)

	fmt.Println("start")

	time.Sleep(time.Second * 2)

	WG.Add(1)

	// detect new blocks
	go detectBlocks()

	WG.Wait()

	fmt.Println("end")
}

func connectEthereum() *ethclient.Client {
	alchemyURL := os.Getenv("ALCHEMY_URL")
	client, err := ethclient.Dial(alchemyURL)
	if err != nil {
		log.Error().Err(err).Msg("")
		log.Error().Msg("Retrying connectEthereum")

		time.Sleep(time.Millisecond * 500)

		return connectEthereum()
	}

	return client
}

func initLogger() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Caller().Stack().Timestamp().Str("service", "detector").Logger()
}

func ConfigRuntime() {
	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
}
