package api

import "github.com/idena-network/idena-contract-runner/chain"

type ChainApi struct {
	baseApi *BaseApi
	bc      *chain.MemBlockchain
}

func NewChainApi(baseApi *BaseApi, chain *chain.MemBlockchain) *ChainApi {
	return &ChainApi{
		baseApi: baseApi,
		bc:      chain,
	}
}

func (api *ChainApi) GenerateBlocks(cnt int) {
	api.bc.GenerateBlocks(cnt)
}
