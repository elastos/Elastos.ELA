package transaction

// TransactionPool provides storage for transactions in the pending
// transaction pool.
type TransactionPool interface {

	//  add a transaction to the pool.
	Add(*NodeTransaction) error

	//returns all transactions that were in the pool.
	Dump() ([]*NodeTransaction, error)
}
