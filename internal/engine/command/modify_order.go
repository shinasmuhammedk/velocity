package command

// in the command package
type ModifyOrderCommand struct {
	OrderID     int64
	NewPrice    int64
	NewQuantity int64
	Result      chan error
}

func (ModifyOrderCommand) isCommand() {}
