package chain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kunalshah017/myference/server/internal/chain/bindings"
)

type IndexerConfig struct {
	RPCURL, DatabaseURL string
	Contract            common.Address
	StartBlock          uint64
	Confirmations       uint64
}

type Indexer struct {
	eth           *ethclient.Client
	db            *sql.DB
	contract      common.Address
	binding       *bindings.MyferenceMarket
	chainID       *big.Int
	startBlock    uint64
	confirmations uint64
	feeRecipient  common.Address
}

func OpenIndexer(ctx context.Context, config IndexerConfig) (*Indexer, error) {
	eth, err := ethclient.DialContext(ctx, config.RPCURL)
	if err != nil {
		return nil, err
	}
	chainID, err := eth.ChainID(ctx)
	if err != nil {
		eth.Close()
		return nil, err
	}
	db, err := sql.Open("pgx", config.DatabaseURL)
	if err != nil {
		eth.Close()
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		eth.Close()
		db.Close()
		return nil, err
	}
	binding, err := bindings.NewMyferenceMarket(config.Contract, eth)
	if err != nil {
		eth.Close()
		db.Close()
		return nil, err
	}
	feeRecipient, err := binding.FeeRecipient(&bind.CallOpts{Context: ctx})
	if err != nil {
		eth.Close()
		db.Close()
		return nil, err
	}
	return &Indexer{eth: eth, db: db, contract: config.Contract, binding: binding, chainID: chainID, startBlock: config.StartBlock, confirmations: config.Confirmations, feeRecipient: feeRecipient}, nil
}

func (i *Indexer) Close() {
	i.eth.Close()
	i.db.Close()
}

func (i *Indexer) Sync(ctx context.Context) error {
	head, err := i.eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	if head.Number.Uint64() < i.confirmations {
		return nil
	}
	target := head.Number.Uint64() - i.confirmations
	next, last, lastHash, err := i.cursor(ctx)
	if err != nil {
		return err
	}
	if last != nil {
		canonical, err := i.eth.HeaderByNumber(ctx, new(big.Int).SetUint64(*last))
		if err != nil || canonical.Hash().Hex() != lastHash {
			if err := i.rewind(ctx); err != nil {
				return err
			}
			next = i.startBlock
		}
	}
	for block := next; block <= target; block++ {
		if err := i.indexBlock(ctx, block); err != nil {
			return err
		}
	}
	return nil
}

func (i *Indexer) cursor(ctx context.Context) (uint64, *uint64, string, error) {
	var next uint64
	var last sql.NullInt64
	var hash sql.NullString
	err := i.db.QueryRowContext(ctx, `SELECT next_block,last_block,last_block_hash FROM chain_cursors WHERE chain_id=$1 AND contract_address=$2`, i.chainID.String(), i.contract.Hex()).Scan(&next, &last, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return i.startBlock, nil, "", nil
	}
	if err != nil {
		return 0, nil, "", err
	}
	if !last.Valid {
		return next, nil, "", nil
	}
	value := uint64(last.Int64)
	return next, &value, hash.String, nil
}

func (i *Indexer) indexBlock(ctx context.Context, number uint64) error {
	header, err := i.eth.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return err
	}
	logs, err := i.eth.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: new(big.Int).SetUint64(number), ToBlock: new(big.Int).SetUint64(number), Addresses: []common.Address{i.contract}})
	if err != nil {
		return err
	}
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO chain_blocks (chain_id,contract_address,block_number,block_hash,parent_hash) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (chain_id,contract_address,block_number) DO UPDATE SET block_hash=EXCLUDED.block_hash,parent_hash=EXCLUDED.parent_hash`, i.chainID.String(), i.contract.Hex(), number, header.Hash().Hex(), header.ParentHash.Hex()); err != nil {
		return err
	}
	for _, eventLog := range logs {
		inserted, err := i.insertLog(ctx, tx, eventLog)
		if err != nil {
			return err
		}
		if inserted {
			if err := i.applyLog(ctx, tx, eventLog); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO chain_cursors (chain_id,contract_address,next_block,last_block,last_block_hash) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (chain_id,contract_address) DO UPDATE SET next_block=EXCLUDED.next_block,last_block=EXCLUDED.last_block,last_block_hash=EXCLUDED.last_block_hash`, i.chainID.String(), i.contract.Hex(), number+1, number, header.Hash().Hex()); err != nil {
		return err
	}
	return tx.Commit()
}

func (i *Indexer) insertLog(ctx context.Context, tx *sql.Tx, eventLog types.Log) (bool, error) {
	payload, _ := json.Marshal(map[string]any{"address": eventLog.Address.Hex(), "topics": eventLog.Topics, "data": eventLog.Data})
	result, err := tx.ExecContext(ctx, `INSERT INTO chain_logs (chain_id,contract_address,block_number,block_hash,transaction_hash,log_index,payload) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, i.chainID.String(), i.contract.Hex(), eventLog.BlockNumber, eventLog.BlockHash.Hex(), eventLog.TxHash.Hex(), eventLog.Index, payload)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	return rows == 1, nil
}

func (i *Indexer) applyLog(ctx context.Context, tx *sql.Tx, eventLog types.Log) error {
	if len(eventLog.Topics) == 0 {
		return errors.New("contract log has no topic")
	}
	parsedABI, err := bindings.MyferenceMarketMetaData.GetAbi()
	if err != nil {
		return err
	}
	event, err := parsedABI.EventByID(eventLog.Topics[0])
	if err != nil {
		return err
	}
	switch event.Name {
	case "Deposited":
		decoded, err := i.binding.ParseDeposited(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO chain_accounts (chain_id,contract_address,address,customer_balance) VALUES ($1,$2,$3,$4) ON CONFLICT (chain_id,contract_address,address) DO UPDATE SET customer_balance=chain_accounts.customer_balance+EXCLUDED.customer_balance`, i.chainID.String(), i.contract.Hex(), decoded.Customer.Hex(), decoded.Amount.String())
		return err
	case "WithdrawalRequested":
		decoded, err := i.binding.ParseWithdrawalRequested(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE chain_accounts SET customer_balance=customer_balance-$4,claimable=claimable+$4 WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, i.chainID.String(), i.contract.Hex(), decoded.Account.Hex(), decoded.Amount.String())
		return err
	case "Claimed":
		decoded, err := i.binding.ParseClaimed(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE chain_accounts SET claimable=claimable-$4 WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, i.chainID.String(), i.contract.Hex(), decoded.Account.Hex(), decoded.Amount.String())
		return err
	case "BondDeposited":
		decoded, err := i.binding.ParseBondDeposited(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO chain_accounts (chain_id,contract_address,address,provider_bond) VALUES ($1,$2,$3,$4) ON CONFLICT (chain_id,contract_address,address) DO UPDATE SET provider_bond=EXCLUDED.provider_bond`, i.chainID.String(), i.contract.Hex(), decoded.Provider.Hex(), decoded.TotalBond.String())
		return err
	case "ProviderSignerSet":
		decoded, err := i.binding.ParseProviderSignerSet(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO chain_provider_signers(chain_id,contract_address,provider,signer,allowed) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(chain_id,contract_address,provider,signer) DO UPDATE SET allowed=EXCLUDED.allowed`, i.chainID.String(), i.contract.Hex(), decoded.Provider.Hex(), decoded.Signer.Hex(), decoded.Allowed)
		return err
	case "BondExitRequested":
		decoded, err := i.binding.ParseBondExitRequested(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE chain_accounts SET bond_exit_available_at=$4 WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, i.chainID.String(), i.contract.Hex(), decoded.Provider.Hex(), decoded.AvailableAt)
		return err
	case "BondExitFinalized":
		decoded, err := i.binding.ParseBondExitFinalized(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE chain_accounts SET provider_bond=0,bond_exit_available_at=0,claimable=claimable+$4 WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, i.chainID.String(), i.contract.Hex(), decoded.Provider.Hex(), decoded.Amount.String())
		return err
	case "OfferPublished":
		decoded, err := i.binding.ParseOfferPublished(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO chain_offers (chain_id,contract_address,provider,offer_id,version,model_hash,capability_hash,input_per_million,output_per_million,compute_per_second) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`, i.chainID.String(), i.contract.Hex(), decoded.Provider.Hex(), common.Hash(decoded.OfferId).Hex(), decoded.Version, common.Hash(decoded.ModelHash).Hex(), common.Hash(decoded.CapabilityHash).Hex(), decoded.InputPerMillion.String(), decoded.OutputPerMillion.String(), decoded.ComputePerSecond.String())
		return err
	case "SessionOpened":
		decoded, err := i.binding.ParseSessionOpened(eventLog)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO chain_sessions (chain_id,contract_address,session_id,customer,allowance,expires_at) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, i.chainID.String(), i.contract.Hex(), common.Hash(decoded.SessionId).Hex(), decoded.Customer.Hex(), decoded.Allowance.String(), decoded.ExpiresAt)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE chain_accounts SET customer_balance=customer_balance-$4 WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, i.chainID.String(), i.contract.Hex(), decoded.Customer.Hex(), decoded.Allowance.String())
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO sessions(id,account_id,state,confirmed_balance_wei)
			SELECT $1,a.id,'open',$3 FROM accounts a WHERE lower(a.wallet_address)=lower($2)
			ON CONFLICT(id) DO UPDATE SET account_id=EXCLUDED.account_id,state='open',confirmed_balance_wei=EXCLUDED.confirmed_balance_wei`, common.Hash(decoded.SessionId).Hex(), decoded.Customer.Hex(), decoded.Allowance.String())
		return err
	case "SessionCloseRequested":
		decoded, err := i.binding.ParseSessionCloseRequested(eventLog)
		if err != nil {
			return err
		}
		sessionID := common.Hash(decoded.SessionId).Hex()
		if _, err = tx.ExecContext(ctx, `UPDATE chain_sessions SET close_available_at=$4 WHERE chain_id=$1 AND contract_address=$2 AND session_id=$3`, i.chainID.String(), i.contract.Hex(), sessionID, decoded.AvailableAt); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE sessions SET state='closing' WHERE id=$1 AND state='open'`, sessionID)
		return err
	case "SessionClosed":
		decoded, err := i.binding.ParseSessionClosed(eventLog)
		if err != nil {
			return err
		}
		var customer string
		if err := tx.QueryRowContext(ctx, `UPDATE chain_sessions SET finalized=true WHERE chain_id=$1 AND contract_address=$2 AND session_id=$3 RETURNING customer`, i.chainID.String(), i.contract.Hex(), common.Hash(decoded.SessionId).Hex()).Scan(&customer); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE chain_accounts SET customer_balance=customer_balance+$4 WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, i.chainID.String(), i.contract.Hex(), customer, decoded.ReturnedAmount.String()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE sessions SET state='closed',confirmed_balance_wei=0 WHERE id=$1`, common.Hash(decoded.SessionId).Hex())
		return err
	case "ReceiptSettled":
		decoded, err := i.binding.ParseReceiptSettled(eventLog)
		if err != nil {
			return err
		}
		requestID := common.Hash(decoded.RequestId).Hex()
		result, err := tx.ExecContext(ctx, `INSERT INTO chain_settlements (chain_id,contract_address,request_id,session_id,provider,provider_amount,fee_amount,transaction_hash) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, i.chainID.String(), i.contract.Hex(), requestID, common.Hash(decoded.SessionId).Hex(), decoded.Provider.Hex(), decoded.ProviderAmount.String(), decoded.FeeAmount.String(), eventLog.TxHash.Hex())
		if err != nil {
			return err
		}
		inserted, _ := result.RowsAffected()
		if inserted == 0 {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE chain_sessions SET spent=spent+$4+$5 WHERE chain_id=$1 AND contract_address=$2 AND session_id=$3`, i.chainID.String(), i.contract.Hex(), common.Hash(decoded.SessionId).Hex(), decoded.ProviderAmount.String(), decoded.FeeAmount.String()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sessions SET confirmed_balance_wei=GREATEST(0,confirmed_balance_wei-$2-$3) WHERE id=$1`, common.Hash(decoded.SessionId).Hex(), decoded.ProviderAmount.String(), decoded.FeeAmount.String()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,claimable) VALUES ($1,$2,$3,$4) ON CONFLICT (chain_id,contract_address,address) DO UPDATE SET claimable=chain_accounts.claimable+EXCLUDED.claimable`, i.chainID.String(), i.contract.Hex(), decoded.Provider.Hex(), decoded.ProviderAmount.String()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,claimable) VALUES ($1,$2,$3,$4) ON CONFLICT (chain_id,contract_address,address) DO UPDATE SET claimable=chain_accounts.claimable+EXCLUDED.claimable`, i.chainID.String(), i.contract.Hex(), i.feeRecipient.Hex(), decoded.FeeAmount.String()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE settlement_queue SET state='settled',updated_at=now() WHERE request_id=$1 AND state='broadcasting'`, requestID); err != nil {
			return err
		}
		return transitionSettlementRequest(ctx, tx, requestID, "submitted", "settled")
	case "ProviderSlashed":
		decoded, err := i.binding.ParseProviderSlashed(eventLog)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE chain_accounts SET provider_bond=provider_bond-$4,bond_exit_available_at=CASE WHEN provider_bond-$4=0 THEN 0 ELSE bond_exit_available_at END WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, i.chainID.String(), i.contract.Hex(), decoded.Provider.Hex(), decoded.Amount.String()); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,claimable) VALUES ($1,$2,$3,$4) ON CONFLICT (chain_id,contract_address,address) DO UPDATE SET claimable=chain_accounts.claimable+EXCLUDED.claimable`, i.chainID.String(), i.contract.Hex(), i.feeRecipient.Hex(), decoded.Amount.String())
		return err
	default:
		return nil
	}
}

func (i *Indexer) rewind(ctx context.Context) error {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT request_id FROM chain_settlements WHERE chain_id=$1 AND contract_address=$2`, i.chainID.String(), i.contract.Hex())
	if err != nil {
		return err
	}
	var reverted []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		reverted = append(reverted, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range reverted {
		if _, err := tx.ExecContext(ctx, `UPDATE settlement_queue SET state='broadcasting',updated_at=now() WHERE request_id=$1 AND state='settled'`, id); err != nil {
			return err
		}
		if err := transitionSettlementRequest(ctx, tx, id, "settled", "submitted"); err != nil {
			return err
		}
	}
	for _, table := range []string{"chain_settlements", "chain_sessions", "chain_offers", "chain_provider_signers", "chain_accounts", "chain_blocks", "chain_cursors"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE chain_id=$1 AND contract_address=$2", table), i.chainID.String(), i.contract.Hex()); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM chain_logs WHERE chain_id=$1 AND contract_address=$2`, i.chainID.String(), i.contract.Hex()); err != nil {
		return err
	}
	return tx.Commit()
}

func (i *Indexer) LogCount(ctx context.Context) int {
	var count int
	_ = i.db.QueryRowContext(ctx, `SELECT count(*) FROM chain_logs WHERE chain_id=$1 AND contract_address=$2`, i.chainID.String(), i.contract.Hex()).Scan(&count)
	return count
}
