package engine

func (e *Engine) Start(){
    e.run()
}

func (e *Engine) run() {
	for order := range e.orderQueue {

		trades, err := e.matcher.Match(order)
		if err != nil {
			continue
		}

		for _, t := range trades {
			e.tradeQueue <- t
		}
	}
}