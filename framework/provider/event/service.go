package trace

import (
    "github.com/gohade/hade/framework"
    "github.com/gohade/hade/framework/contract"
)

type EventEngine interface {
    QueueType() string
    Push(topic string, content []byte) error
    Pop(topic string) ([]byte, error)
}

type HadeEventService struct {
    container framework.Container
    events    map[string]contract.Event
    topics    map[string]string //key为event的name，val为topic
    engine    EventEngine
}

func (h *HadeEventService) Register(event contract.Event) error {
    return h.RegisterWithTopic("default", event)
}

func (h *HadeEventService) RegisterWithTopic(topic string, event contract.Event) error {
    name := event.GetName()
    h.events[name] = event
    h.topics[name] = topic
    return nil
}

func (h *HadeEventService) getTopic(event contract.Event) string {
    name := event.GetName()
    if val, ok := h.topics[name]; ok {
        return val
    }
    return "default"
}

func (h *HadeEventService) Fire(event contract.Event) error {
    bt, err := event.MarshalText()
    if err != nil {
        return err
    }

    if err := h.engine.Push(h.getTopic(event), bt); err != nil {
        return err
    }

    return nil
}

func NewHadeEventService(params ...interface{}) (interface{}, error) {
    c := params[0].(framework.Container)

    // TODO: load event file

    // TODO: new event engine

    // TODO: go start handler

    return &HadeEventService{container: c}, nil
}
