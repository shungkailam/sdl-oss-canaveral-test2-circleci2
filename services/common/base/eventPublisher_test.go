/*
 * Copyright (c) 2018 Nutanix Inc. All rights reserved.
 */

package base_test

import (
	"cloudservices/common/base"
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// Event definition
type TestEvent struct {
	TenantID string
	EdgeID   string
	Status   bool
	ID       string
}

func (event *TestEvent) IsAsync() bool {
	return true
}

func (event *TestEvent) EventName() string {
	return "TestEvent"
}

func (event *TestEvent) GetID() string {
	return event.ID
}

// Event listener
type TestEventListener struct {
}

func (listener *TestEventListener) OnEvent(ctx context.Context, event base.Event) error {
	edgeEvent := event.(*TestEvent)
	fmt.Println("Got event!", edgeEvent)
	return nil
}

func (listener *TestEventListener) EventName() string {
	return "TestEvent"
}

func TestEventPublisher(t *testing.T) {
	ctx := context.Background()
	listener := &TestEventListener{}
	base.Publisher.Subscribe(listener)
	base.Publisher.Publish(ctx, &TestEvent{Status: true, ID: uuid.New().String()})
	base.Publisher.Publish(ctx, &TestEvent{Status: false, ID: uuid.New().String()})
	base.Publisher.Unsubscribe(listener)
	base.Publisher.Publish(ctx, &TestEvent{Status: true, ID: uuid.New().String()})
	base.Publisher.Publish(ctx, &TestEvent{Status: false, ID: uuid.New().String()})
}
