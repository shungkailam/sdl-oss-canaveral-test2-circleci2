package base

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

var (
	AsyncEventListenerTimeoutSecs = time.Second * 60
)

// Event interface is to be implemented.
type Event interface {
	IsAsync() bool
	EventName() string
	GetID() string
}

// EventListener is to be implemented.
type EventListener interface {
	OnEvent(context.Context, Event) error
	// Event name must match the name of the event to which this listener wants to subscribe.
	EventName() string
}

// EventPublisher interface is implemented by EventPublisherImpl.
type EventPublisher interface {
	Subscribe(EventListener) error
	Unsubscribe(EventListener) error
	Publish(context.Context, Event) error
}

// eventPublisherImpl is the implementation of EventPublisher
type eventPublisherImpl struct {
	rwMutex        *sync.RWMutex
	eventListeners map[string]*map[EventListener]bool
}

// NewEventPublisher returns the pointer to an EventPublisherImpl instance.
func NewEventPublisher() EventPublisher {
	return &eventPublisherImpl{rwMutex: &sync.RWMutex{}, eventListeners: map[string]*map[EventListener]bool{}}
}

func validateEventSubscriber(eventListener EventListener) (string, error) {
	var eventName string
	listenerType := reflect.TypeOf(eventListener)
	if listenerType == nil {
		return eventName, errors.New("Invalid event listener")
	}
	if listenerType.Kind() != reflect.Ptr {
		return eventName, errors.New("Event listener must be a pointer")
	}
	eventName = strings.TrimSpace(eventListener.EventName())
	if len(eventName) == 0 {
		return eventName, errors.New("Invalid event name")
	}
	return eventName, nil
}

func (manager *eventPublisherImpl) doInWriteLock(op func() error) error {
	manager.rwMutex.Lock()
	defer manager.rwMutex.Unlock()
	return op()
}

func (manager *eventPublisherImpl) doInReadLock(op func() error) error {
	manager.rwMutex.RLock()
	defer manager.rwMutex.RUnlock()
	return op()
}

// Subscribe is to subscribe to an event
func (manager *eventPublisherImpl) Subscribe(eventListener EventListener) error {
	eventName, err := validateEventSubscriber(eventListener)
	if err != nil {
		return err
	}
	return manager.doInWriteLock(func() error {
		listeners, ok := manager.eventListeners[eventName]
		if !ok {
			listeners = &map[EventListener]bool{}
			manager.eventListeners[eventName] = listeners
		}
		(*listeners)[eventListener] = true
		return nil
	})
}

// Unsubscribe is to unsubscribe from an event
func (manager *eventPublisherImpl) Unsubscribe(eventListener EventListener) error {
	eventName, err := validateEventSubscriber(eventListener)
	if err != nil {
		return err
	}
	return manager.doInWriteLock(func() error {
		listeners, ok := manager.eventListeners[eventName]
		if !ok {
			return fmt.Errorf("No listener found for event name %s", eventName)
		}
		delete(*listeners, eventListener)
		if len(*listeners) == 0 {
			delete(manager.eventListeners, eventName)
		}
		return nil
	})
}

// Publish is to publish an event
func (manager *eventPublisherImpl) Publish(ctx context.Context, event Event) error {
	reqID := GetRequestID(ctx)
	eventName := strings.TrimSpace(event.EventName())
	if len(eventName) == 0 {
		return errors.New("Invalid event")
	}
	eventID := strings.TrimSpace(event.GetID())
	if len(eventID) == 0 {
		return errors.New("Invalid event. Missing ID")
	}
	glog.Infof("Request %s: Publishing event %s", reqID, eventName)
	return manager.doInReadLock(func() error {
		listeners, ok := manager.eventListeners[eventName]
		if !ok {
			glog.Warningf("No listener found for event name %s", eventName)
			return nil
		}
		// Range copies the address to the same location
		// A new location is created before being overwritten.
		fn := func(ctx context.Context, listener EventListener) error {
			start := time.Now()
			defer func() {
				elapsed := time.Since(start)
				glog.Infof("Request %s: Completed in %.2f ms for event %s", reqID, float32(elapsed/time.Millisecond), eventName)
			}()
			err := listener.OnEvent(ctx, event)
			if err != nil {
				// Error is swallowed for async listeners
				glog.Warningf("Request %s: Error occurred in async publish. Error: %s", reqID, err.Error())
			}
			return err
		}
		errMsgs := []string{}
		for listener := range *listeners {
			if event.IsAsync() {
				// Create a new context because the other old ctx can be cancelled by the caller for async
				calleeCtx := context.WithValue(context.Background(), RequestIDKey, GetRequestID(ctx))
				authContext, err := GetAuthContext(ctx)
				if err == nil {
					calleeCtx = context.WithValue(calleeCtx, AuthContextKey, authContext)
				}
				calleeCtx, _ = context.WithTimeout(calleeCtx, AsyncEventListenerTimeoutSecs)
				go fn(calleeCtx, listener)
			} else {
				err := fn(ctx, listener)
				if err != nil {
					errMsgs = append(errMsgs, err.Error())
				}
			}
		}
		if len(errMsgs) > 0 {
			errMsg := strings.Join(errMsgs, "\n")
			glog.Warningf("Request %s: Error occurred in processing event %+v. Error: %s", reqID, event, errMsg)
			return fmt.Errorf("Error: %s", errMsg)
		}
		return nil
	})
}

// Publisher is the singleton instance to be used
var Publisher EventPublisher

func init() {
	Publisher = NewEventPublisher()
}
