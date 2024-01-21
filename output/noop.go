package output

import "log"

type NoOp struct{}

func NewNoOp() NoOp {
	return NoOp{}
}

func (_ NoOp) Start() error {
	log.Println("new NoOp output started.")
	return nil
}

func (_ NoOp) Stop() error {
	log.Println("NoOp output stopped.")
	return nil
}
