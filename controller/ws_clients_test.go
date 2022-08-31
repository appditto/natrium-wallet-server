package controller

import (
	"testing"

	"github.com/appditto/natrium-wallet-server/utils"
	"github.com/google/uuid"
)

func TestWsClientPut(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	utils.AssertEqual(t, id, wsClients.Get(id).id)
	utils.AssertEqual(t, 1, wsClients.Len())
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

	utils.AssertEqual(t, id, wsClients.Get(id).id)
	utils.AssertEqual(t, 1, wsClients.Len())
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
	utils.AssertEqual(t, "account_1", wsClients.Get(id).accounts[0])
	utils.AssertEqual(t, "account_2", wsClients.Get(id).accounts[1])
	utils.AssertEqual(t, 2, len(wsClients.Get(id).accounts))
}

func TestWsClientUpdateCurrency(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	wsClients.UpdateCurrency(id, "TRY")
	utils.AssertEqual(t, "TRY", wsClients.Get(id).currency)
	wsClients.UpdateCurrency(id, "USD")
	utils.AssertEqual(t, "USD", wsClients.Get(id).currency)
}

func TestWsClientDelete(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-1234567890ab")
	wsClients := NewWSSubscriptions()
	wsClients.Put(WSClient{
		id: id,
	})

	utils.AssertEqual(t, id, wsClients.Get(id).id)
	utils.AssertEqual(t, 1, wsClients.Len())

	wsClients.Delete(id)
	utils.AssertEqual(t, 0, wsClients.Len())
	utils.AssertEqual(t, (*WSClient)(nil), wsClients.Get(id))
}
