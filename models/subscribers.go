package models

import "sync"

// Singleton to keep client info loaded in memory
// This should contain users currently connected via websocket in-memory
type SubscriberInfo struct {
	Clients []string
}

var singleton *SubscriberInfo
var once sync.Once

func GetSubInfo() *SubscriberInfo {
	once.Do(func() {
		// Create object
		singleton = &SubscriberInfo{
			Clients: []string{},
		}
	})
	return singleton
}
