package repositories

import (
	"fmt"
	"time"

	"github.com/meta-node-blockchain/mining-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OtpRepository interface {
	SaveOtpVerification(referal, otp string, expiredTime time.Time) error
	CheckOtpVerification(referal, otp string, nowTime time.Time) (bool, error)
	DeleteOtpVerification(referal, otp string) error
}

type otpRepository struct {
	db *gorm.DB
}

func NewOtpRepository(db *gorm.DB) OtpRepository {
	return &otpRepository{db}
}

func (repo *otpRepository) SaveOtpVerification(
	referal, otp string, expiredTime time.Time) error {
		fmt.Println("referal:",referal)
		fmt.Println("otp:",otp)
		fmt.Println("expiredTime:",expiredTime)
	return repo.db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&model.OtpVerification{
		Referal:       referal,
		Otp:   otp,
		ExpiredTime: expiredTime}).Error
}

func (repo *otpRepository) CheckOtpVerification(
	referal, otp string, nowTime time.Time) (bool, error) {
	var history *model.OtpVerification
	result := repo.db.Model(&model.OtpVerification{}).
		Where("referal = ? AND otp = ? AND expiredTime > ?", referal, otp, nowTime).
		First(&history)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, result.Error    
	}
	updateResult := repo.db.Model(&history).Update("verified", true)
	if updateResult.Error != nil {
		return false, updateResult.Error    
	}
	
	return updateResult.RowsAffected > 0, nil
}

func (repo *otpRepository) DeleteOtpVerification(referal, otp string) error {
	return repo.db.Unscoped().Where("encrypted_token = ? AND otp = ?", referal, otp).
		Delete(&model.OtpVerification{}).Error
}

