package command

type CancelOrderCommand struct {
    OrderID int64
    Result  chan error   // the background goroutine sends the outcome here
}

func (CancelOrderCommand) isCommand (){}