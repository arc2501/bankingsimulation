package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransfer(t *testing.T) {
	store := NewStore(testDB)
	// creating 2 random accounts for transaction
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	// Now we can easily test a single tx but it will not give good result
	// so for stressing it out lets use the Go routines
	// we want to run n concurrent tx
	n := 5
	// each tx will transfer Rs10 from acc1 to acc2
	amount := int64(10)

	// we need this channel to recieve the errors in go routines
	errs := make(chan error)
	// need this channel to recieve the Tx Results
	results := make(chan TransferTxResult)

	for i := 0; i < n; i++ {
		// giving a dynamic tx name tx1,tx2
		txName := fmt.Sprintf("tx %d", i+1)
		// Remember we cannot use testify require as it is running in different scope
		go func() {
			// New context will accept the parent context and the key value pair (seperately)
			// as a rule key of this context should be a seperate struct (as suggested)
			// so we declare an empty context
			// and value is txname
			ctx := context.WithValue(context.Background(), txKey, txName)
			result, err := store.TransferTx(ctx, TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})
			// Sending err back to the calling function
			errs <- err
			// Sending result back to the calling function
			results <- result

		}() // this bracket ensures its running
	}
	fmt.Println(">>before: ", account1.Balance, account2.Balance)

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
		result := <-results
		require.NotEmpty(t, result)

		//check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)

		// skipping the entry checks for now

		// Coming straight to main juice for update account

		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		// checking ki kahi jo account humne bheja tha and jo
		// from account result ne return kiya wo same hai ya nai
		require.Equal(t, account1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)

		// Printing the balances from both the account returned by the result
		fmt.Println(">> tx:", fromAccount.Balance, toAccount.Balance)

		// diff between the remaining balance of account we created and
		// the account being returned by the result after tx
		// this diff should = the money which we have deducted
		diff1 := account1.Balance - fromAccount.Balance
		diff2 := toAccount.Balance - account2.Balance
		// bhai jitna nikla hai utna hi deposit hona chahiye
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		require.True(t, diff2 > 0)
		// utna hi nikalna chahiye jitna amount bataya hai
		// and uske multiples mein as 5 go routines nikal
		// rahe honge na
		require.True(t, diff1%amount == 0)

		k := int(diff1 / amount)
		// kayde se wo multiple 1 se bada and no.of times
		// operated (which is 5 in this case) se chhota hona
		require.True(t, k >= 1 && k <= n)

	}
	// Now checking for the updated balance
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after tx:", updatedAccount1.Balance, updatedAccount2.Balance)
	// account1 se ghatane ke baad ka balance
	require.Equal(t, account1.Balance-int64(n)*amount, updatedAccount1.Balance)
	// account2 mein badhaane ke baad ka balance
	require.Equal(t, account2.Balance+int64(n)*amount, updatedAccount2.Balance)

}
