package main

import (
	"testing"
)

func TestEventCounts(t *testing.T) {
	c := new(eventCounts)
	c.log()
	if c.messageReceive.Load() != 0 || c.messageRead.Load() != 0 {
		t.Fatalf("initial counts should be 0")
	}
	c.messageReceive.Add(2)
	c.messageRead.Add(1)
	c.reactionCreated.Add(1)
	if c.messageReceive.Load() != 2 || c.messageRead.Load() != 1 || c.reactionCreated.Load() != 1 {
		t.Fatalf("counts after add: receive=2 read=1 reaction_created=1")
	}
	c.log()
}
