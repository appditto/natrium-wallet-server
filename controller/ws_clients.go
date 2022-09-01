package controller

import (
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type WSClient struct {
	ID       uuid.UUID
	Accounts []string // Subscribed accounts
	Currency string
	Conn     *websocket.Conn
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
		if v.ID == id {
			return true
		}
	}
	return false
}

func (r *WSClientMap) accountExists(id uuid.UUID, account string) bool {
	for _, v := range r.subscriptions {
		if v.ID == id {
			for _, a := range v.Accounts {
				if a == account {
					return true
				}
			}
		}
	}
	return false
}

// Get accounts
func (r *WSClientMap) GetConnsForAccount(account string) []*websocket.Conn {
	r.mu.Lock()
	defer r.mu.Unlock()
	var conns []*websocket.Conn
	for _, v := range r.subscriptions {
		for _, a := range v.Accounts {
			if a == account {
				conns = append(conns, v.Conn)
			}
		}
	}
	return conns
}

// Get accounts
func (r *WSClientMap) GetAllConns() []WSClient {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.subscriptions
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
	if !r.exists(value.ID) {
		r.subscriptions = append(r.subscriptions, value)
	}
}

// Add account
func (r *WSClientMap) AddAccount(id uuid.UUID, account string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.accountExists(id, account) {
		for i, v := range r.subscriptions {
			if v.ID == id {
				r.subscriptions[i].Accounts = append(r.subscriptions[i].Accounts, account)
			}
		}
	}
}

func (r *WSClientMap) UpdateCurrency(id uuid.UUID, currency string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exists(id) {
		r.subscriptions[r.indexOf(id)].Currency = currency
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
		if v.ID == id {
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
