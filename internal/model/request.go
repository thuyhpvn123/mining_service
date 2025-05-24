package model

type OtpVerificationRequest struct {
	Referal       string    
}
type OtpAuthenticationRequest struct {
	Referal       string    
	Otp   string  
}