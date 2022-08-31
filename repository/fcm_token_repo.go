package repository

import (
	"time"

	"github.com/appditto/natrium-wallet-server/models/dbmodels"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

// Repository for SQL operations
type FcmTokenRepo struct {
	DB *gorm.DB
}

func (repo *FcmTokenRepo) CreateMockTokens() error {
	token1 := &dbmodels.FcmToken{
		FcmToken: "token1",
		Account:  "account1",
	}

	token2 := &dbmodels.FcmToken{
		FcmToken: "token2",
		Account:  "account2",
	}

	token22 := &dbmodels.FcmToken{
		FcmToken: "token3",
		Account:  "account2",
	}

	err := repo.DB.Create(&token1).Error

	if err != nil {
		return err
	}

	err = repo.DB.Create(&token2).Error

	if err != nil {
		return err
	}

	err = repo.DB.Create(&token22).Error

	if err != nil {
		return err
	}

	return nil
}

func (repo *FcmTokenRepo) GetTokensForAccount(account string) ([]dbmodels.FcmToken, error) {
	var tokens []dbmodels.FcmToken
	if err := repo.DB.Where("account = ?", account).Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (repo *FcmTokenRepo) DeleteFcmToken(token string) error {
	return repo.DB.Delete(&dbmodels.FcmToken{}, "fcm_token = ?", token).Error
}

func (repo *FcmTokenRepo) AddOrUpdateToken(token string, account string) error {
	// Add token to db if not exists
	var count int64
	err := repo.DB.Model(&dbmodels.FcmToken{}).Where("fcm_token = ?", token).Where("account = ?", account).Count(&count).Error
	if err != nil || count == 0 {
		fcmToken := &dbmodels.FcmToken{
			FcmToken: token,
			Account:  account,
		}
		if err = repo.DB.Create(fcmToken).Error; err != nil {
			return err
		}
	} else if count > 0 {
		// Already exists so we will update updated_at
		if err = repo.DB.Model(&dbmodels.FcmToken{}).Where("fcm_token = ?", token).Where("account = ?", account).Update("updated_at", time.Now()).Error; err != nil {
			klog.Errorf("Error updating fcm token updated_at %v", err)
			return err
		}
	}
	return nil
}
