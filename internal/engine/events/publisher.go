package events

type Publisher interface{
    Publish(Event)
}

type Subscriber interface{
    Handle(Event)
}