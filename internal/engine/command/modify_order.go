package command

// in the command package
type ModifyOrderCommand struct {
	OrderID     string
	NewPrice    int64
	NewQuantity int64
	Result      chan error
}
func (ModifyOrderCommand) isCommand() {}