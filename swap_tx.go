package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

func createTransaction() {
	// ETH/USDC pool address
	poolAddress := "0x8ad599c3A0ff1De082011EFDDc58f1908eb6e6D8"
	toAddress := common.HexToAddress(poolAddress)

	publicKey := PRIVATE_KEY.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Error().Msg("Error casting public key to ECDSA")
		return
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := CLIENT.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	gasPrice, err := CLIENT.SuggestGasPrice(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	value := new(big.Int).SetInt64(0)
	gasLimit := uint64(0)

	txData := []byte{}

	functionSig := "swap(address,bool,int256,int160,bytes)"
	functionSigEncoded := crypto.Keccak256([]byte(functionSig))[:4] // first 4 bytes
	txData = append(txData, functionSigEncoded...)
	fmt.Printf("functionSigEncoded: 0x%x\n", functionSigEncoded)

	recipient := "0x379c7A0BCa9B985E5b980D917Ea71B83F1C656ca"
	recipientEncoded := common.LeftPadBytes([]byte(common.Hex2Bytes(removeHexPrefix(recipient))), 32)
	txData = append(txData, recipientEncoded...)
	fmt.Printf("recipientEncoded: 0x%x\n", recipientEncoded) // hex.EncodeToString(recipient))

	zeroForOneEncoded := math.PaddedBigBytes(common.Big1, 32)
	txData = append(txData, zeroForOneEncoded...)
	fmt.Printf("zeroForOneEncoded: 0x%x\n", zeroForOneEncoded)

	// USDC amount to swap
	amountSpecified, ok := new(big.Int).SetString("100000000", 10)
	if !ok {
		log.Error().Msg("Can't set amountSpecified string")
		return
	}
	amountSpecifiedEncoded := math.U256Bytes(amountSpecified)
	txData = append(txData, amountSpecifiedEncoded...)
	fmt.Println("amountSpecifiedEncoded:", hex.EncodeToString(amountSpecifiedEncoded))

	sqrtPriceLimitX96, ok := new(big.Int).SetString("100000000", 10)
	if !ok {
		log.Error().Msg("Can't set sqrtPriceLimitX96 string")
		return
	}
	sqrtPriceLimitX96Encoded := math.U256Bytes(sqrtPriceLimitX96)
	fmt.Println("sqrtPriceLimitX96Encoded:", hex.EncodeToString(sqrtPriceLimitX96Encoded))

	dataEncoded := common.LeftPadBytes([]byte{}, 32)
	fmt.Println("dataEncoded", hex.EncodeToString(dataEncoded))
	txData = append(recipientEncoded, dataEncoded...)

	fmt.Println("txData", hex.EncodeToString(txData))

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, txData)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(new(big.Int).SetInt64(CHAIN_ID)), PRIVATE_KEY)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	fmt.Println("signedTx:", signedTx)

	// err = CLIENT.SendTransaction(context.Background(), signedTx)
	// if err != nil {
	// 	log.Error().Err(err).Msg("")
	// 	return
	// }
}
