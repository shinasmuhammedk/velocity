package command

type CancelOrderCommand struct {
    OrderID string
    Result  chan error   // the background goroutine sends the outcome here
}

func (CancelOrderCommand) isCommand (){}