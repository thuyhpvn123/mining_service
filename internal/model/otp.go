package model

import (
	"time"
	"gorm.io/gorm"
)

type OtpVerification struct {
	gorm.Model
	Referal       string    `json:"referal" gorm:"column:referal;unique"`
	Otp   string    `json:"otp" gorm:"column:otp;"`
	ExpiredTime time.Time `json:"expired_time" gorm:"column:expiredTime;"`
	Verified bool      `json:"verified" gorm:"column:verified;"`
}

func (OtpVerification) TableName() string {
	return "otp_verifications"
}
func (e *OtpVerification) SetDefaultValues() {
	e.Verified = false
}