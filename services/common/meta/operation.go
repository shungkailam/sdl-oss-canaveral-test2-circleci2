package meta

type Operation int

const (
	OperationRead Operation = iota + 1
	OperationCreate
	OperationUpdate
	OperationDelete
)
