package component

import (
	"context"
	"log"
	"testing"

	"github.com/serialx/hashring"
)

func TestConsistentHash_onSetNodes(t *testing.T) {
	nodes := make(map[string]struct{})
	nodes["pod1"] = struct{}{}
	c := &ConsistentHash{
		serviceName:       "offlinepush",
		podName:           "pod1",
		types:             []string{"type1", "type2", "type3", "type4", "type5", "type6"},
		responsibleKeysCh: make(chan []string),
		ring:              hashring.New([]string{"pod1"}),
		nodes:             nodes,
	}
	ch := c.Subscribe(context.Background())
	tests := []struct {
		name  string
		pod   string
		nodes []string
	}{
		{
			name:  "ok",
			pod:   "pod1",
			nodes: []string{"pod1", "pod2", "pod3"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log.Println(c.GetResponsibleKeys())
			c.OnSetNodes(tc.nodes)
			keys := <-ch
			log.Println(keys)
		})
	}
}

func TestConsistentHash_onAddNodes(t *testing.T) {
	nodes := make(map[string]struct{})
	nodes["pod1"] = struct{}{}
	c := &ConsistentHash{
		serviceName:       "offlinepush",
		podName:           "pod1",
		types:             []string{"type1", "type2", "type3", "type4", "type5", "type6"},
		responsibleKeysCh: make(chan []string),
		ring:              hashring.New([]string{"pod1"}),
		nodes:             nodes,
	}
	ch := c.Subscribe(context.Background())
	tests := []struct {
		name    string
		pod     string
		newnode string
	}{
		{
			name:    "ok",
			pod:     "pod1",
			newnode: "pod2",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log.Println(c.GetResponsibleKeys())
			c.OnAddNode(tc.newnode)
			keys := <-ch
			log.Println(keys)
		})
	}
}
