package api

import (
	"context"
	"github.com/idena-network/idena-contract-runner/chain"
	"github.com/idena-network/idena-go/blockchain"
	"github.com/idena-network/idena-go/blockchain/attachments"
	"github.com/idena-network/idena-go/blockchain/fee"
	"github.com/idena-network/idena-go/blockchain/types"
	"github.com/idena-network/idena-go/blockchain/validation"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/hexutil"
	"github.com/idena-network/idena-go/vm"
	"github.com/idena-network/idena-go/vm/env"
	"github.com/idena-network/idena-go/vm/helpers"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"math/big"
	"strconv"
)

type ContractApi struct {
	baseApi *BaseApi
	bc      *chain.MemBlockchain
}

// NewContractApi creates a new NetApi instance
func NewContractApi(baseApi *BaseApi, bc *chain.MemBlockchain) *ContractApi {
	return &ContractApi{baseApi: baseApi, bc: bc}
}

type DeployArgs struct {
	From     common.Address  `json:"from"`
	CodeHash hexutil.Bytes   `json:"codeHash"`
	Amount   decimal.Decimal `json:"amount"`
	Args     DynamicArgs     `json:"args"`
	MaxFee   decimal.Decimal `json:"maxFee"`
	Code     hexutil.Bytes   `json:"code"`
}

type CallArgs struct {
	From           common.Address  `json:"from"`
	Contract       common.Address  `json:"contract"`
	Method         string          `json:"method"`
	Amount         decimal.Decimal `json:"amount"`
	Args           DynamicArgs     `json:"args"`
	MaxFee         decimal.Decimal `json:"maxFee"`
	BroadcastBlock uint64          `json:"broadcastBlock"`
}

type TerminateArgs struct {
	From     common.Address  `json:"from"`
	Contract common.Address  `json:"contract"`
	Args     DynamicArgs     `json:"args"`
	MaxFee   decimal.Decimal `json:"maxFee"`
}

type DynamicArgs []*DynamicArg

type DynamicArg struct {
	Index  int    `json:"index"`
	Format string `json:"format"`
	Value  string `json:"value"`
}

type ReadonlyCallArgs struct {
	Contract common.Address `json:"contract"`
	Method   string         `json:"method"`
	Format   string         `json:"format"`
	Args     DynamicArgs    `json:"args"`
}

type EventsArgs struct {
	Contract common.Address `json:"contract"`
}

func (a DynamicArg) ToBytes() ([]byte, error) {
	switch a.Format {
	case "byte":
		i, err := strconv.ParseUint(a.Value, 10, 8)
		if err != nil {
			return nil, errors.Errorf("cannot parse byte: \"%v\"", a.Value)
		}
		return []byte{byte(i)}, nil
	case "int8":
		i, err := strconv.ParseInt(a.Value, 10, 8)
		if err != nil {
			return nil, errors.Errorf("cannot parse int8: \"%v\"", a.Value)
		}
		return common.ToBytes(i), nil
	case "uint64":
		i, err := strconv.ParseUint(a.Value, 10, 64)
		if err != nil {
			return nil, errors.Errorf("cannot parse uint64: \"%v\"", a.Value)
		}
		return common.ToBytes(i), nil
	case "int64":
		i, err := strconv.ParseInt(a.Value, 10, 64)
		if err != nil {
			return nil, errors.Errorf("cannot parse int64: \"%v\"", a.Value)
		}
		return common.ToBytes(i), nil
	case "string":
		return []byte(a.Value), nil
	case "bigint":
		v := new(big.Int)
		_, ok := v.SetString(a.Value, 10)
		if !ok {
			return nil, errors.Errorf("cannot parse bigint: \"%v\"", a.Value)
		}
		return v.Bytes(), nil
	case "hex":
		data, err := hexutil.Decode(a.Value)
		if err != nil {
			return nil, errors.Errorf("cannot parse hex: \"%v\"", a.Value)
		}
		return data, nil
	case "dna":
		d, err := decimal.NewFromString(a.Value)
		if err != nil {
			return nil, errors.Errorf("cannot parse dna: \"%v\"", a.Value)
		}
		return blockchain.ConvertToInt(d).Bytes(), nil
	default:
		data, err := hexutil.Decode(a.Value)
		if err != nil {
			return nil, errors.Errorf("cannot parse hex: \"%v\"", a.Value)
		}
		return data, nil
	}
}

func (d DynamicArgs) ToSlice() ([][]byte, error) {

	m := make(map[int]*DynamicArg)
	maxIndex := -1
	for _, a := range d {
		m[a.Index] = a
		if a.Index > maxIndex {
			maxIndex = a.Index
		}
	}
	var data [][]byte

	for i := 0; i <= maxIndex; i++ {
		if a, ok := m[i]; ok {
			bytes, err := a.ToBytes()
			if err != nil {
				return nil, err
			}
			data = append(data, bytes)
		} else {
			data = append(data, nil)
		}
	}
	return data, nil
}

type TxReceipt struct {
	Contract common.Address  `json:"contract"`
	Method   string          `json:"method"`
	Success  bool            `json:"success"`
	GasUsed  uint64          `json:"gasUsed"`
	TxHash   *common.Hash    `json:"txHash"`
	Error    string          `json:"error"`
	GasCost  decimal.Decimal `json:"gasCost"`
	TxFee    decimal.Decimal `json:"txFee"`
}

type Event struct {
	Contract common.Address  `json:"contract"`
	Event    string          `json:"event"`
	Args     []hexutil.Bytes `json:"args"`
}

type MapItem struct {
	Key   interface{} `json:"key"`
	Value interface{} `json:"value"`
}

type IterateMapResponse struct {
	Items             []*MapItem     `json:"items"`
	ContinuationToken *hexutil.Bytes `json:"continuationToken"`
}

func (api *ContractApi) buildDeployContractTx(args DeployArgs, estimate bool) (*types.Transaction, error) {
	var codeHash common.Hash
	codeHash.SetBytes(args.CodeHash)

	from := args.From
	if from == (common.Address{}) {
		from = api.baseApi.getCurrentCoinbase()
	}
	convertedArgs, err := args.Args.ToSlice()
	if err != nil {
		return nil, err
	}
	payload, _ := attachments.CreateDeployContractAttachment(codeHash, args.Code, convertedArgs...).ToBytes()
	tx := api.baseApi.getTx(from, nil, types.DeployContractTx, args.Amount, args.MaxFee, decimal.Zero, 0, 0, payload)
	return api.signIfNeeded(from, tx, estimate)
}

func (api *ContractApi) buildCallContractTx(args CallArgs, estimate bool) (*types.Transaction, error) {

	from := args.From
	if from == (common.Address{}) {
		from = api.baseApi.getCurrentCoinbase()
	}
	convertedArgs, err := args.Args.ToSlice()
	if err != nil {
		return nil, err
	}
	payload, _ := attachments.CreateCallContractAttachment(args.Method, convertedArgs...).ToBytes()
	tx := api.baseApi.getTx(from, &args.Contract, types.CallContractTx, args.Amount, args.MaxFee, decimal.Zero, 0, 0,
		payload)
	return api.signIfNeeded(from, tx, estimate)
}

func (api *ContractApi) buildTerminateContractTx(args TerminateArgs, estimate bool) (*types.Transaction, error) {

	from := args.From
	if from == (common.Address{}) {
		from = api.baseApi.getCurrentCoinbase()
	}
	convertedArgs, err := args.Args.ToSlice()
	if err != nil {
		return nil, err
	}
	payload, _ := attachments.CreateTerminateContractAttachment(convertedArgs...).ToBytes()
	tx := api.baseApi.getTx(from, &args.Contract, types.TerminateContractTx, decimal.Zero, args.MaxFee, decimal.Zero, 0,
		0, payload)
	return api.signIfNeeded(from, tx, estimate)
}

func (api *ContractApi) signIfNeeded(from common.Address, tx *types.Transaction, estimate bool) (*types.Transaction, error) {
	sign := !estimate || api.baseApi.canSign(from)
	if !sign {
		return tx, nil
	}
	return api.baseApi.signTransaction(from, tx, nil)
}

func (api *ContractApi) EstimateDeploy(args DeployArgs) (*TxReceipt, error) {
	appState := api.baseApi.getAppStateForCheck()
	vm := vm.NewVmImpl(appState, api.bc.Head, nil, api.bc.Config())
	tx, err := api.buildDeployContractTx(args, true)
	if err != nil {
		return nil, err
	}
	var from *common.Address
	if tx.Signed() {
		if err := validation.ValidateTx(appState, tx, appState.State.FeePerGas(), validation.MempoolTx); err != nil {
			return nil, err
		}
	} else {
		from = &args.From
	}
	r := vm.Run(tx, from, -1)
	r.GasCost = api.bc.GetGasCost(appState, r.GasUsed)
	return convertEstimatedReceipt(tx, r, appState.State.FeePerGas()), nil
}

func (api *ContractApi) EstimateCall(args CallArgs) (*TxReceipt, error) {
	appState := api.baseApi.getAppStateForCheck()
	vm := vm.NewVmImpl(appState, api.bc.Head, nil, api.bc.Config())
	tx, err := api.buildCallContractTx(args, true)
	if err != nil {
		return nil, err
	}
	var from *common.Address
	if tx.Signed() {
		if err := validation.ValidateTx(appState, tx, appState.State.FeePerGas(), validation.MempoolTx); err != nil {
			return nil, err
		}
	} else {
		from = &args.From
	}
	if !common.ZeroOrNil(tx.Amount) {
		var sender common.Address
		if tx.Signed() {
			sender, _ = types.Sender(tx)
		} else {
			sender = args.From
		}
		appState.State.SubBalance(sender, tx.Amount)
		appState.State.AddBalance(*tx.To, tx.Amount)
	}

	r := vm.Run(tx, from, -1)
	r.GasCost = api.bc.GetGasCost(appState, r.GasUsed)
	return convertEstimatedReceipt(tx, r, appState.State.FeePerGas()), nil
}

func (api *ContractApi) EstimateTerminate(args TerminateArgs) (*TxReceipt, error) {
	appState := api.baseApi.getAppStateForCheck()
	vm := vm.NewVmImpl(appState, api.bc.Head, nil, api.bc.Config())
	tx, err := api.buildTerminateContractTx(args, true)
	if err != nil {
		return nil, err
	}
	var from *common.Address
	if tx.Signed() {
		if err := validation.ValidateTx(appState, tx, appState.State.FeePerGas(), validation.MempoolTx); err != nil {
			return nil, err
		}
	} else {
		from = &args.From
	}
	r := vm.Run(tx, from, -1)
	r.GasCost = api.bc.GetGasCost(appState, r.GasUsed)
	return convertEstimatedReceipt(tx, r, appState.State.FeePerGas()), nil
}

func convertReceipt(tx *types.Transaction, receipt *types.TxReceipt, feePerGas *big.Int) *TxReceipt {
	fee := fee.CalculateFee(1, feePerGas, tx)
	var err string
	if receipt.Error != nil {
		err = receipt.Error.Error()
	}
	txHash := receipt.TxHash
	return &TxReceipt{
		Success:  receipt.Success,
		Error:    err,
		Method:   receipt.Method,
		Contract: receipt.ContractAddress,
		TxHash:   &txHash,
		GasUsed:  receipt.GasUsed,
		GasCost:  blockchain.ConvertToFloat(receipt.GasCost),
		TxFee:    blockchain.ConvertToFloat(fee),
	}
}

func convertEstimatedReceipt(tx *types.Transaction, receipt *types.TxReceipt, feePerGas *big.Int) *TxReceipt {
	res := convertReceipt(tx, receipt, feePerGas)
	if !tx.Signed() {
		res.TxHash = nil
	}
	return res
}

func (api *ContractApi) Deploy(ctx context.Context, args DeployArgs) (common.Hash, error) {
	tx, err := api.buildDeployContractTx(args, false)
	if err != nil {
		return common.Hash{}, err
	}
	return api.baseApi.sendInternalTx(ctx, tx)
}

func (api *ContractApi) Call(ctx context.Context, args CallArgs) (common.Hash, error) {
	tx, err := api.buildCallContractTx(args, false)
	if err != nil {
		return common.Hash{}, err
	}
	return api.baseApi.sendInternalTx(ctx, tx)
}
func (api *ContractApi) Terminate(ctx context.Context, args TerminateArgs) (common.Hash, error) {
	tx, err := api.buildTerminateContractTx(args, false)
	if err != nil {
		return common.Hash{}, err
	}
	return api.baseApi.sendInternalTx(ctx, tx)
}

func (api *ContractApi) ReadData(contract common.Address, key string, format string) (interface{}, error) {
	data := api.baseApi.getReadonlyAppState().State.GetContractValue(contract, []byte(key))
	if data == nil {
		return nil, errors.New("data is nil")
	}
	return conversion(format, data)
}

func (api *ContractApi) ReadonlyCall(args ReadonlyCallArgs) (interface{}, error) {
	vm := vm.NewVmImpl(api.baseApi.getReadonlyAppState(), api.bc.Head, nil, api.bc.Config())
	convertedArgs, err := args.Args.ToSlice()
	if err != nil {
		return nil, err
	}
	data, err := vm.Read(args.Contract, args.Method, convertedArgs...)
	if err != nil {
		return nil, err
	}
	return conversion(args.Format, data)
}

func (api *ContractApi) GetStake(contract common.Address) interface{} {
	hash := api.baseApi.getReadonlyAppState().State.GetCodeHash(contract)
	stake := api.baseApi.getReadonlyAppState().State.GetContractStake(contract)
	return struct {
		Hash  *common.Hash
		Stake decimal.Decimal
	}{
		hash,
		blockchain.ConvertToFloat(stake),
	}
}

func (api *ContractApi) Events(args EventsArgs) interface{} {

	events := api.bc.ReadEvents(args.Contract)

	var list []*Event
	for idx := range events {
		e := &Event{
			Contract: events[idx].Contract,
			Event:    events[idx].Event,
		}
		list = append(list, e)
		for i := range events[idx].Args {
			e.Args = append(e.Args, events[idx].Args[i])
		}
	}
	return list
}

func (api *ContractApi) ReadMap(contract common.Address, mapName string, key hexutil.Bytes, format string) (interface{}, error) {
	data := api.baseApi.getReadonlyAppState().State.GetContractValue(contract, env.FormatMapKey([]byte(mapName), key))
	if data == nil {
		return nil, errors.New("data is nil")
	}
	return conversion(format, data)
}

func (api *ContractApi) IterateMap(contract common.Address, mapName string, continuationToken *hexutil.Bytes, keyFormat, valueFormat string, limit int) (*IterateMapResponse, error) {
	state := api.baseApi.getReadonlyAppState().State

	minKey := []byte(mapName)
	maxKey := []byte(mapName)
	for i := len([]byte(mapName)); i < common.MaxContractStoreKeyLength; i++ {
		maxKey = append(maxKey, 0xFF)
	}

	if continuationToken != nil && len(*continuationToken) > 0 {
		minKey = *continuationToken
	}

	var items []*MapItem
	var err error
	var token hexutil.Bytes
	prefixLen := len([]byte(mapName))
	state.IterateContractStore(contract, minKey, maxKey, func(key []byte, value []byte) bool {

		if len(items) >= limit {
			token = key
			return true
		}

		item := new(MapItem)
		item.Key, err = conversion(keyFormat, key[prefixLen:])
		if err != nil {
			return true
		}
		item.Value, err = conversion(valueFormat, value)
		if err != nil {
			return true
		}
		items = append(items, item)
		return false
	})
	if err != nil {
		return nil, err
	}
	return &IterateMapResponse{
		Items:             items,
		ContinuationToken: &token,
	}, nil
}

func conversion(convertTo string, data []byte) (interface{}, error) {
	switch convertTo {
	case "byte":
		return helpers.ExtractByte(0, data)
	case "uint64":
		return helpers.ExtractUInt64(0, data)
	case "string":
		return string(data), nil
	case "bigint":
		v := new(big.Int)
		v.SetBytes(data)
		return v.String(), nil
	case "hex":
		return hexutil.Encode(data), nil
	case "dna":
		v := new(big.Int)
		v.SetBytes(data)
		return blockchain.ConvertToFloat(v), nil
	default:
		return hexutil.Encode(data), nil
	}
}