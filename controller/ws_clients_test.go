package controller

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWsClientPut(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	assert.Equal(t, id, wsClients.Get(id).id)
	assert.Equal(t, 1, wsClients.Len())
}

func TestWsClientPutOnlyOnce(t *testing.T) {
	// Ensure it behaves like a map
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})
	wsClients.Put(WSClient{
		id: id,
	})

	assert.Equal(t, id, wsClients.Get(id).id)
	assert.Equal(t, 1, wsClients.Len())
}

func TestWsClientAddAccount(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	wsClients.AddAccount(id, "account_1")
	wsClients.AddAccount(id, "account_2")
	wsClients.AddAccount(id, "account_2")
	assert.Equal(t, "account_1", wsClients.Get(id).accounts[0])
	assert.Equal(t, "account_2", wsClients.Get(id).accounts[1])
	assert.Equal(t, 2, len(wsClients.Get(id).accounts))
}

func TestWsClientUpdateCurrency(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	wsClients.UpdateCurrency(id, "TRY")
	assert.Equal(t, "TRY", wsClients.Get(id).currency)
	wsClients.UpdateCurrency(id, "USD")
	assert.Equal(t, "USD", wsClients.Get(id).currency)
}

func TestWsClientDelete(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	assert.Equal(t, id, wsClients.Get(id).id)
	assert.Equal(t, 1, wsClients.Len())

	wsClients.Delete(id)
	assert.Equal(t, 0, wsClients.Len())
	assert.Equal(t, (*WSClient)(nil), wsClients.Get(id))
}

func TestWsClientindexOf(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	assert.Equal(t, 0, wsClients.indexOf(id))
	assert.Equal(t, 1, wsClients.Len())
}
