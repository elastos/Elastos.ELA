// Copyright (c) 2017-2019 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/dpos/state"
	"github.com/elastos/Elastos.ELA/utils/signal"
)

const (
	// end block height of compensation
	H2 = uint32(439000)

	// count of CRC
	CRCCount = 12

	// count of producer in a round for dpos
	DPOSProducerCount = 36
)

func main() {
	var opk string
	flag.StringVar(&opk, "opk", "", "owner public key")
	flag.Parse()
	if opk == "" {
		fmt.Println("Invaild parameter, use '-opk <ownerpublickey>' to " +
			"specify a not nil producer owner public key.")
		os.Exit(1)
	}

	ownerPublicKey, err := common.HexStringToBytes(opk)
	if err != nil {
		fmt.Println("invalid node public key")
		os.Exit(1)
	}
	if err := generateCompensation(ownerPublicKey); err != nil {
		fmt.Println("generate illegal compensation failed")
		os.Exit(1)
	}
}

func generateCompensation(ownerPublicKey []byte) error {
	chain, err := blockchain.NewChainStore(
		"elastos/data", config.DefaultParams.GenesisBlock)
	if err != nil {
		fmt.Println("init chain store failed")
		return err
	}
	defer chain.Close()

	arbiters, err := state.NewArbitrators(&config.DefaultParams, nil,
		chain.GetHeight, func() (*types.Block, error) {
			hash := chain.GetCurrentBlockHash()
			block, err := chain.GetBlock(hash)
			if err != nil {
				return nil, err
			}
			blockchain.CalculateTxsFee(block)
			return block, nil
		}, func(height uint32) (*types.Block, error) {
			hash, err := chain.GetBlockHash(height)
			if err != nil {
				return nil, err
			}
			block, err := chain.GetBlock(hash)
			if err != nil {
				return nil, err
			}
			blockchain.CalculateTxsFee(block)
			return block, nil
		})
	if err != nil {
		fmt.Println("init arbiters failed")
		return err
	}
	ledger := blockchain.Ledger{}
	ledger.Store = chain
	ledger.Arbitrators = arbiters
	blockchain.DefaultLedger = &ledger

	blockchain.DefaultLedger.Arbitrators = arbiters
	blockChain, err := blockchain.New(chain,
		&config.DefaultParams, arbiters.State)
	if err != nil {
		fmt.Println("init blockchain failed")
		return err
	}
	ledger.Blockchain = blockChain

	var interrupt = signal.NewInterrupt()
	return processBlockForCompensation(blockChain, chain, arbiters,
		chain.GetHeight(), interrupt.C, ownerPublicKey)
}

func processBlockForCompensation(b *blockchain.BlockChain,
	db blockchain.IChainStore, arbiters state.Arbitrators, bestHeight uint32,
	interrupt <-chan struct{}, ownerPublicKey []byte) (err error) {
	done := make(chan struct{})
	go func() {
		ownerHash, e := contract.PublicKeyToStandardProgramHash(ownerPublicKey)
		if e != nil {
			err = e
			return
		}

		foundIllegal := false
		foundIllegalReward := false
		illegalHeight := uint32(0)
		currentReward := state.RewardData{}
		myVotes := common.Fixed64(0)
		totalVotes := common.Fixed64(0)
		totalCompensation := common.Fixed64(0)
		totalRoundDPOSReward := common.Fixed64(0)

		// for deal with force change reward
		lastBlockReward := common.Fixed64(0)
		dposIndex := uint8(0)

		startHeight := config.DefaultParams.VoteStartHeight
		for i := startHeight; i <= bestHeight; i++ {
			hash, e := db.GetBlockHash(i)
			if e != nil {
				err = e
				break
			}
			block, e := db.GetBlock(hash)
			if e != nil {
				err = e
				break
			}

			if block.Height >= bestHeight-uint32(DPOSProducerCount) {
				blockchain.CalculateTxsFee(block)
			}

			// print process block information
			if block.Height%10000 == 0 && !foundIllegal {
				fmt.Println("process block height:", block.Height)
			}

			if e = blockchain.PreProcessSpecialTx(block); e != nil {
				err = e
				break
			}
			confirm, _ := db.GetConfirm(block.Hash())
			arbiters.ProcessBlock(block, confirm)

			producer := b.GetState().GetProducer(ownerPublicKey)

			// found illegal start height
			if !foundIllegal && producer != nil && producer.State() == state.Illegal {
				illegalHeight = block.Height
				currentReward = arbiters.GetCurrentRewardData()
				fmt.Println("##################################")
				fmt.Println("changed to illegal at height:", block.Height)
				fmt.Println("##################################")
				foundIllegal = true
			}

			// found last reward after illegal
			if foundIllegal && !foundIllegalReward {
				for _, o := range block.Transactions[0].Outputs {
					if !ownerHash.IsEqual(o.ProgramHash) {
						continue
					}
					lastRewardHeight := uint32(0)
					if votes, ok := currentReward.OwnerVotesInRound[*ownerHash]; ok {
						myVotes = votes
						totalVotes = currentReward.TotalVotesInRound
						lastRewardHeight = block.Height
					}
					fmt.Println("##################################")
					fmt.Println("producer last reward height:", lastRewardHeight)
					fmt.Println("producer last votes:", myVotes)
					fmt.Println("producer last totalVotes:", totalVotes)
					fmt.Println("producer last votesPercent:",
						float64(myVotes)/float64(totalVotes))
					fmt.Println("##################################")
					foundIllegalReward = true
					err = checkFirstRoundReward(b, db, lastRewardHeight-36,
						lastRewardHeight-1, myVotes, totalVotes, o.Value)
					if err != nil {
						return
					}
					break
				}
			}

			// calculate the reward, if total reward is not 0 and coin
			// base has reward to CRCAddress then should calculate the reward
			for _, out := range block.Transactions[0].Outputs {
				if totalRoundDPOSReward == common.Fixed64(0) {
					break
				}
				if out.ProgramHash.IsEqual(config.DefaultParams.CRCAddress) {
					individualBlockConfirmReward, individualProducerReward :=
						calculateReward(totalRoundDPOSReward, myVotes, totalVotes)
					totalCompensation += individualBlockConfirmReward
					totalCompensation += individualProducerReward
					fmt.Println(
						"height:", block.Height,
						"rewardVotes:", individualProducerReward,
						"rewardDPOS:", individualBlockConfirmReward,
						"totalCompensation:", totalCompensation)

					// check confirm reward
					if individualBlockConfirmReward == out.Value/12 {
						fmt.Println("✔️ check pass, dpos confirm reward:",
							individualBlockConfirmReward)
					} else {
						fmt.Println("× check not pass, dpos confirm reward:",
							individualBlockConfirmReward,
							"CRC confirm reward:", out.Value/CRCCount)
						panic("reward check failed")
					}

					// deal with force change, dposIndex less than
					// DPOSProducerCount means have force changed
					if dposIndex < DPOSProducerCount {
						totalRoundDPOSReward = lastBlockReward
					} else {
						totalRoundDPOSReward = 0
					}
					dposIndex = 0
				}
			}

			// calculate DPOS reward of block
			if foundIllegalReward && block.Height <= H2 {
				blockchain.CalculateTxsFee(block)
				lastBlockReward = getBlockDPOSReward(block)
				totalRoundDPOSReward += lastBlockReward
				dposIndex++
			}

			// print the result and exit
			if block.Height == H2 {
				fmt.Println("##################################")
				fmt.Println("A:", myVotes)
				fmt.Println("B:", totalVotes)
				fmt.Println("P:", myVotes/totalVotes)
				fmt.Println("H1:", illegalHeight)
				fmt.Println("H2:", H2)
				fmt.Println("R:", totalCompensation, " (totalCompensation)")
				fmt.Println("##################################")
				break
			}
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		fmt.Println("process finished")

	case <-interrupt:
		fmt.Println("process interrupted")
	}
	return err
}

func getBlockDPOSReward(block *types.Block) common.Fixed64 {
	totalTxFx := common.Fixed64(0)
	for _, tx := range block.Transactions {
		totalTxFx += tx.Fee
	}

	return common.Fixed64(math.Ceil(float64(totalTxFx+
		config.DefaultParams.RewardPerBlock) * 0.35))
}

func checkFirstRoundReward(b *blockchain.BlockChain, db blockchain.IChainStore,
	startHeight uint32, endHeight uint32, myVotes common.Fixed64,
	totalVotesInRound common.Fixed64, realReward common.Fixed64) error {

	totalDPOSReward := common.Fixed64(0)
	for i := startHeight; i <= endHeight; i++ {
		hash, e := db.GetBlockHash(i)
		if e != nil {
			return e
		}
		block, e := db.GetBlock(hash)
		if e != nil {
			return e
		}

		blockchain.CalculateTxsFee(block)
		reward := getBlockDPOSReward(block)
		totalDPOSReward += reward
	}

	individualBlockConfirmReward, individualProducerReward :=
		calculateReward(totalDPOSReward, myVotes, totalVotesInRound)

	totalCompensation := common.Fixed64(0)
	totalCompensation += individualBlockConfirmReward
	totalCompensation += individualProducerReward

	fmt.Println("##################################")
	if totalCompensation != realReward {
		fmt.Println("× check first turn from", startHeight,
			"to", endHeight, "failed, totalCompensation:",
			totalCompensation, "need to be:", realReward)
		return errors.New("check illegal round reward failed")
	} else {
		fmt.Println("✔️check first turn from", startHeight,
			"to", endHeight, "succeed, totalCompensation:",
			totalCompensation, "need to be:", realReward)
	}
	fmt.Println("##################################")

	return nil
}

func calculateReward(totalDPOSReward common.Fixed64, myVotes common.Fixed64,
	totalVotesInRound common.Fixed64) (common.Fixed64, common.Fixed64) {

	totalBlockConfirmReward := float64(totalDPOSReward) * 0.25
	totalTopProducersReward := float64(totalDPOSReward) - totalBlockConfirmReward
	individualBlockConfirmReward := common.Fixed64(
		math.Floor(totalBlockConfirmReward / float64(DPOSProducerCount)))
	rewardPerVote := totalTopProducersReward / float64(totalVotesInRound)
	individualProducerReward := common.Fixed64(math.Floor(float64(
		myVotes) * rewardPerVote))

	return individualBlockConfirmReward, individualProducerReward
}
