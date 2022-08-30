package controller

import (
	"sync"

	"github.com/google/uuid"
)

type WSClient struct {
	id       uuid.UUID
	accounts []string // Subscribed accounts
	currency string
}

type WSClientMap struct {
	mu            sync.Mutex
	subscriptions []WSClient
}

func NewWSSubscriptions() *WSClientMap {
	return &WSClientMap{
		subscriptions: []WSClient{},
	}
}

// See if element exists
func (r *WSClientMap) exists(id uuid.UUID) bool {
	for _, v := range r.subscriptions {
		if v.id == id {
			return true
		}
	}
	return false
}

func (r *WSClientMap) accountExists(id uuid.UUID, account string) bool {
	for _, v := range r.subscriptions {
		if v.id == id {
			for _, a := range v.accounts {
				if a == account {
					return true
				}
			}
		}
	}
	return false
}

// Get length - synchronized
func (r *WSClientMap) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.subscriptions)
}

// Put value into map - synchronized
func (r *WSClientMap) Put(value WSClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.exists(value.id) {
		r.subscriptions = append(r.subscriptions, value)
	}
}

// Add account
func (r *WSClientMap) AddAccount(id uuid.UUID, account string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.accountExists(id, account) {
		for i, v := range r.subscriptions {
			if v.id == id {
				r.subscriptions[i].accounts = append(r.subscriptions[i].accounts, account)
			}
		}
	}
}

func (r *WSClientMap) UpdateCurrency(id uuid.UUID, currency string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exists(id) {
		r.subscriptions[r.indexOf(id)].currency = currency
	}
}

// Gets a value from the map - synchronized
func (r *WSClientMap) Get(id uuid.UUID) *WSClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exists(id) {
		return &r.subscriptions[r.indexOf(id)]
	}

	return nil
}

// Removes specified id - synchronized
func (r *WSClientMap) Delete(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	index := r.indexOf(id)
	if index > -1 {
		r.subscriptions = remove(r.subscriptions, r.indexOf(id))
	}
}

func (r *WSClientMap) indexOf(id uuid.UUID) int {
	for i, v := range r.subscriptions {
		if v.id == id {
			return i
		}
	}
	return -1
}

// NOT thread safe, must be called from within a locked section
func remove(s []WSClient, i int) []WSClient {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
