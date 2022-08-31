package repository

import (
	"os"
	"testing"

	"github.com/appditto/natrium-wallet-server/database"
	"github.com/stretchr/testify/assert"
)

func TestGetTokensForAccount(t *testing.T) {
	os.Setenv("MOCK_REDIS", "true")
	defer os.Unsetenv("MOCK_REDIS")
	mockDb, err := database.NewConnection(&database.Config{
		Host:     os.Getenv("DB_MOCK_HOST"),
		Port:     os.Getenv("DB_MOCK_PORT"),
		Password: os.Getenv("DB_MOCK_PASS"),
		User:     os.Getenv("DB_MOCK_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   "testing",
	})
	assert.Equal(t, nil, err)
	err = database.DropAndCreateTables(mockDb)
	assert.Equal(t, nil, err)
	fcmRepo := &FcmTokenRepo{
		DB: mockDb,
	}

	// Create mock tokens
	err = fcmRepo.CreateMockTokens()

	// Get tokens for account
	tokens, err := fcmRepo.GetTokensForAccount("account1")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(tokens))
	tokens, err = fcmRepo.GetTokensForAccount("account2")
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(tokens))
	tokens, err = fcmRepo.GetTokensForAccount("account3")
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(tokens))
}

func TestDeleteToken(t *testing.T) {
	os.Setenv("MOCK_REDIS", "true")
	defer os.Unsetenv("MOCK_REDIS")
	mockDb, err := database.NewConnection(&database.Config{
		Host:     os.Getenv("DB_MOCK_HOST"),
		Port:     os.Getenv("DB_MOCK_PORT"),
		Password: os.Getenv("DB_MOCK_PASS"),
		User:     os.Getenv("DB_MOCK_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   "testing",
	})
	assert.Equal(t, nil, err)
	err = database.DropAndCreateTables(mockDb)
	assert.Equal(t, nil, err)
	fcmRepo := &FcmTokenRepo{
		DB: mockDb,
	}

	// Create mock tokens
	err = fcmRepo.CreateMockTokens()

	// Delete tokens for account
	err = fcmRepo.DeleteFcmToken("token1")
	assert.Equal(t, nil, err)
	tokens, err := fcmRepo.GetTokensForAccount("account1")
	assert.Equal(t, nil, err)
	assert.Equal(t, 0, len(tokens))
}

func TestAddOrUpdateToken(t *testing.T) {
	os.Setenv("MOCK_REDIS", "true")
	defer os.Unsetenv("MOCK_REDIS")
	mockDb, err := database.NewConnection(&database.Config{
		Host:     os.Getenv("DB_MOCK_HOST"),
		Port:     os.Getenv("DB_MOCK_PORT"),
		Password: os.Getenv("DB_MOCK_PASS"),
		User:     os.Getenv("DB_MOCK_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   "testing",
	})
	assert.Equal(t, nil, err)
	err = database.DropAndCreateTables(mockDb)
	assert.Equal(t, nil, err)
	fcmRepo := &FcmTokenRepo{
		DB: mockDb,
	}

	// Create mock tokens
	err = fcmRepo.CreateMockTokens()

	// * 2) We want to test adding a new token
	err = fcmRepo.AddOrUpdateToken("token1", "account_new")
	assert.Equal(t, nil, err)

	tokens, err := fcmRepo.GetTokensForAccount("account_new")
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, len(tokens))
	assert.Equal(t, "token1", tokens[0].FcmToken)
}
