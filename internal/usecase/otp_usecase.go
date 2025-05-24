package usecase

import (
	"errors"
	"fmt"
	"time"

	"github.com/meta-node-blockchain/mining-service/internal/model"
	"github.com/meta-node-blockchain/mining-service/internal/repositories"
	"github.com/meta-node-blockchain/mining-service/internal/utils"
)


type OtpUsecase interface {
	OtpVerification(request model.OtpVerificationRequest,Otp string) error
	OtpAuthentication(request model.OtpAuthenticationRequest) error
}

type otpUsecase struct {
	otpRepo repositories.OtpRepository
}

func NewOtpUsecase(
	otpRepo repositories.OtpRepository,
) OtpUsecase {
	return &otpUsecase{otpRepo}
}

func (svc *otpUsecase) OtpVerification(request model.OtpVerificationRequest,Otp string) error {
	fmt.Println("222222222")
	exipredTime := time.Now().Add(time.Duration(utils.VerificationExpiredTime) * time.Minute)
	err := svc.otpRepo.SaveOtpVerification(request.Referal,Otp, exipredTime)
	if err != nil {
		return err
	}
	return nil
}

func (svc *otpUsecase) OtpAuthentication(request model.OtpAuthenticationRequest) error {
	existsOtp, err := svc.otpRepo.CheckOtpVerification(request.Referal,request.Otp, time.Now())
	if err != nil {
		return err
	}
	if !existsOtp {
		return errors.New("Invalid otp or otp expired")
	}
	
	return nil
}
