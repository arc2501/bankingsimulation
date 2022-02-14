package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	// this is composition in go
	*Queries
	// Required to create a new Transaction Object
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// Local function taking context and a call back function as input
// then it will start a new DB Transaction
// create a new query object with that Tx
// and call the callback function with the created queries
// and finally commit or rollback transaction based on the error
// returned by the above formed function

func (store *Store) execTx(ctx context.Context, fn func(*Queries) error) error {
	// begining a transaction
	// second argument is for Isolation Level in that DB
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// creating a new query object with that tx object
	// as we know New func accepts DBTX interface
	q := New(tx)

	// Now we have queries which run within transaction
	// We can call input function with that queries
	err = fn(q)

	if err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil {
			return fmt.Errorf("tx err: %v,rb Err: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// creating txKey value for the txName in test file
var txKey = struct{}{}

// New Transfer Tx Global func to perform money tran eg
// It creates a transfer record, add account entries, and update
// accounts' balance within a single database transaction
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		// as you can see that this callback func is using result & arg which
		// are declared outside the function scope, this is what makes this
		// callback func a CLOSURE
		// Since Golang lacks the support of Generics type, Closure is often used
		// when we want to get a result from a callback function as callbacks themselves
		// dont know the exact type which they will be returning

		// yaha pe calling function ki value recieve kar rahe hai
		// by asking the calling function key
		txName := ctx.Value(txKey)

		// Creating a Transfer
		fmt.Println(txName, "create transfer")
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		fmt.Println(txName, "create Entry1")
		// Creating From Account Entry
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		fmt.Println(txName, "create Entry2")
		// Creating To Account Entry
		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		// TODO:Update Account
		// Now after writing the test lets write this
		// get the account & update its balance

		// this if measure is used to ensure that our tx accquires
		// lock in consistent manner
		if arg.FromAccountID < arg.ToAccountID {

			// getting the account and saving to account1
			fmt.Println(txName, "Get Account1 for Update")
			account1, err := q.GetAccountForUpdate(ctx, arg.FromAccountID)
			if err != nil {
				return err
			}
			// result ki FromAccount field bhari with the return of the
			// Update Account jo ki we did on account1 (From account)
			fmt.Println(txName, "Update Account1")
			result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID:      arg.FromAccountID,
				Balance: account1.Balance - arg.Amount,
			})
			if err != nil {
				return err
			}
			fmt.Println(txName, "Get Account2 for Update")
			account2, err := q.GetAccountForUpdate(ctx, arg.ToAccountID)
			if err != nil {
				return err
			}
			fmt.Println(txName, "Update Account2")
			result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID:      arg.ToAccountID,
				Balance: account2.Balance + arg.Amount,
			})

			if err != nil {
				return err
			}

		} else {
			fmt.Println(txName, "Get Account2 for Update")
			account2, err := q.GetAccountForUpdate(ctx, arg.ToAccountID)
			if err != nil {
				return err
			}
			fmt.Println(txName, "Update Account2")
			result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID:      arg.ToAccountID,
				Balance: account2.Balance + arg.Amount,
			})
			if err != nil {
				return err
			}

			fmt.Println(txName, "Get Account1 for Update")
			account1, err := q.GetAccountForUpdate(ctx, arg.FromAccountID)
			if err != nil {
				return err
			}
			// result ki FromAccount field bhari with the return of the
			// Update Account jo ki we did on account1 (From account)
			fmt.Println(txName, "Update Account1")
			result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
				ID:      arg.FromAccountID,
				Balance: account1.Balance - arg.Amount,
			})
			if err != nil {
				return err
			}

		}

		return nil
	})
	return result, err
}
