package go_ora

import "context"

type Transaction struct {
	conn *Connection
	ctx  context.Context
}

func (tx *Transaction) Commit() error {
	return errStuHelperSelectOnly
}

func (tx *Transaction) Rollback() error {
	return errStuHelperSelectOnly
}
