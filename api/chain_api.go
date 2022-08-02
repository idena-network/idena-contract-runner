package api

import (
	"fmt"
	"github.com/idena-network/idena-contract-runner/chain"
	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/core/mempool"
	"math/big"
)

type ChainApi struct {
	baseApi *BaseApi
	pool    *mempool.TxPool
	bc      *chain.MemBlockchain
}

func NewChainApi(baseApi *BaseApi, chain *chain.MemBlockchain, pool *mempool.TxPool) *ChainApi {
	return &ChainApi{
		baseApi: baseApi,
		bc:      chain,
		pool:    pool,
	}
}

func (api *ChainApi) GenerateBlocks(cnt int) {
	fmt.Println(fmt.Sprintf("start generating blocks: %v", cnt))
	api.bc.GenerateBlocks(cnt)
}

func (api *ChainApi) TxReceipt(hash common.Hash) *TxReceipt {
	tx := api.pool.GetTx(hash)
	var idx *types.TransactionIndex

	if tx == nil {
		tx, idx = api.bc.GetTx(hash)
	}

	if tx == nil {
		return nil
	}

	if idx == nil {
		idx = api.bc.GetTxIndex(hash)
	}

	var blockHash common.Hash
	var feePerGas *big.Int
	if idx != nil {
		blockHash = idx.BlockHash
		block := api.bc.GetBlock(blockHash)
		if block != nil {
			feePerGas = block.Header.FeePerGas()
		}
	}

	receipt := api.bc.GetReceipt(hash)

	return convertReceipt(tx, receipt, feePerGas)
}
