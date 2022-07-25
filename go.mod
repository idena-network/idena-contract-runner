module github.com/idena-network/idena-contract-runner

require (
	github.com/idena-network/idena-go v0.29.1
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v0.0.0-20200227202807-02e2044944cc
	github.com/tendermint/tm-db v0.6.7
	github.com/urfave/cli/v2 v2.0.0
)

replace github.com/cosmos/iavl => github.com/idena-network/iavl v0.12.3-0.20211223100228-a33b117aa31e

replace github.com/idena-network/idena-go => ..\idena-go

replace github.com/idena-network/idena-wasm-binding => ..\idena-wasm\idena-wasm-binding
replace github.com/ipfs/fs-repo-migrations/fs-repo-11-to-12 => github.com/idena-network/fs-repo-migrations/fs-repo-11-to-12 v0.0.0-20220601101433-9ce72c125fd3
go 1.16
