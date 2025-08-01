package store

import (
	"sort"
	"testing"
)

func Test_Server_IsReadOnly(t *testing.T) {
	testCases := []struct {
		name          string
		servers       Servers
		nodeID        string
		expectedRO    bool
		expectedFound bool
	}{
		{
			name:          "EmptyServers",
			servers:       nil,
			nodeID:        "1",
			expectedRO:    false,
			expectedFound: false,
		},
		{
			name:          "EmptyNodeID",
			servers:       Servers(make([]*Server, 1)),
			nodeID:        "",
			expectedRO:    false,
			expectedFound: false,
		},
		{
			name: "NonExistentNode",
			servers: Servers([]*Server{
				{ID: "node1", Addr: "localhost:4002", Suffrage: "Voter"},
			}),
			nodeID:        "node2",
			expectedRO:    false,
			expectedFound: false,
		},
		{
			name: "ExistingVoterNode",
			servers: Servers([]*Server{
				{ID: "node1", Addr: "localhost:4002", Suffrage: "Voter"},
			}),
			nodeID:        "node1",
			expectedRO:    false,
			expectedFound: true,
		},
		{
			name: "ExistingNonvoterNode",
			servers: Servers([]*Server{
				{ID: "node1", Addr: "localhost:4002", Suffrage: "Nonvoter"},
			}),
			nodeID:        "node1",
			expectedRO:    true,
			expectedFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro, found := tc.servers.IsReadOnly(tc.nodeID)
			if ro != tc.expectedRO || found != tc.expectedFound {
				t.Fatalf("IsReadOnly for %s returned ro: %t, found: %t, expected ro: %t, expected found: %t", tc.name, ro, found, tc.expectedRO, tc.expectedFound)
			}
		})
	}
}

func Test_Server_Contains(t *testing.T) {
	testCases := []struct {
		name     string
		servers  Servers
		nodeID   string
		expected bool
	}{
		{
			name:     "EmptyServers",
			servers:  nil,
			nodeID:   "1",
			expected: false,
		},
		{
			name:     "EmptyNodeID",
			servers:  Servers(make([]*Server, 1)),
			nodeID:   "",
			expected: false,
		},
		{
			name: "NonExistentNode",
			servers: Servers([]*Server{
				{ID: "node1", Addr: "localhost:4002", Suffrage: "Voter"},
			}),
			nodeID:   "node2",
			expected: false,
		},
		{
			name: "ExistingNode",
			servers: Servers([]*Server{
				{ID: "node1", Addr: "localhost:4002", Suffrage: "Voter"},
			}),
			nodeID:   "node1",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.servers.Contains(tc.nodeID)
			if actual != tc.expected {
				t.Fatalf("Contains for %s returned %t, expected %t", tc.name, actual, tc.expected)
			}
		})
	}
}

func Test_Server_Sort(t *testing.T) {
	servers := Servers{
		{ID: "3", Addr: "localhost:4003", Suffrage: "Voter"},
		{ID: "1", Addr: "localhost:4001", Suffrage: "Voter"},
		{ID: "2", Addr: "localhost:4002", Suffrage: "Nonvoter"},
	}
	expectedOrder := []string{"1", "2", "3"}

	sort.Sort(Servers(servers))

	for i, server := range servers {
		if server.ID != expectedOrder[i] {
			t.Fatalf("Expected server ID %s at index %d, got %s", expectedOrder[i], i, server.ID)
		}
	}
}
