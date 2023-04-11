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

	// USDC amount to swap, decimals of USDC???? 6 or 18
	usdcAmount, ok := new(big.Int).SetString("100000000000000000000", 10)
	if !ok {
		log.Error().Msg("Can't set usdcAmount string")
		return
	}

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

	data := []byte{}
	value := new(big.Int).SetInt64(0)
	gasLimit := uint64(0)

	functionSig := "swap(address,bool,int256,int160,bytes)"
	encodedSig := crypto.Keccak256([]byte(functionSig))[:4] // first 4 bytes

	fmt.Printf("SHA256: 0x%x\n", encodedSig)

	recipient := common.LeftPadBytes([]byte("0x379c7A0BCa9B985E5b980D917Ea71B83F1C656ca"), 32)
	fmt.Println("recipient", recipient)
	fmt.Println("recipient", hex.EncodeToString(recipient))
	zeroForOne := math.PaddedBigBytes(common.Big1, 32)
	fmt.Println("zeroForOne", zeroForOne)
	fmt.Println("zeroForOne", hex.EncodeToString(zeroForOne))

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(new(big.Int).SetInt64(CHAIN_ID)), PRIVATE_KEY)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	fmt.Println("usdcAmount", usdcAmount)
	fmt.Println("encodedSig", encodedSig)
	fmt.Println("signedTx", signedTx)

	// err = CLIENT.SendTransaction(context.Background(), signedTx)
	// if err != nil {
	// 	log.Error().Err(err).Msg("")
	// 	return
	// }
}
