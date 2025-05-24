package services

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/meta-node-blockchain/meta-node/pkg/logger"
	pb "github.com/meta-node-blockchain/meta-node/pkg/proto"
	"github.com/meta-node-blockchain/meta-node/pkg/transaction"

	"github.com/meta-node-blockchain/meta-node/cmd/client"
	"github.com/meta-node-blockchain/mining-service/internal/model"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	e_common "github.com/ethereum/go-ethereum/common"
)

type SendTransactionService interface {
	UpdateOtpStatus(
		parent common.Address,
		status bool,
	)(interface{}, error) 	
	ActiveUserByBe(
		parent common.Address,
		otp [32]byte,
	) (interface{}, error)
		AddNoti(
		title string,
		body string,
		to common.Address,
	) (interface{}, error)
	CheckUserRegistered(
		to common.Address,
	) (interface{}, error)
}
type sendTransactionService struct {
	chainClientAdminNoti        *client.Client
	notiStorageAbi     *abi.ABI
	notiStorageAddress e_common.Address
	adminNoti          e_common.Address
	chainClientBeMinningUser        *client.Client
	miningUserAbi      *abi.ABI
	miningUserAddress  e_common.Address
	beMinningUser      e_common.Address
}

func NewSendTransactionService(
	chainClientAdminNoti        *client.Client,
	notiStorageAbi *abi.ABI,
	notiStorageAddress e_common.Address,
	adminNoti e_common.Address,
	chainClientBeMinningUser        *client.Client,
	miningUserAbi *abi.ABI,
	miningUserAddress e_common.Address,
	beMinningUser e_common.Address,
) SendTransactionService {
	return &sendTransactionService{
		chainClientAdminNoti:        chainClientAdminNoti,
		notiStorageAbi:     notiStorageAbi,
		notiStorageAddress: notiStorageAddress,
		adminNoti:          adminNoti,
		chainClientBeMinningUser:chainClientBeMinningUser,
		miningUserAbi:      miningUserAbi,
		miningUserAddress:  miningUserAddress,
		beMinningUser:      beMinningUser,
	}
}
func (h *sendTransactionService) sendTransactionAndGetResult(
	chainClient       *client.Client,
	from common.Address,
	to common.Address,
	abi *abi.ABI,
	methodName string,
	input []byte,
	unpackTo string,
	attempts int,
) (interface{}, error) {
	callData := transaction.NewCallData(input)

	bData, err := callData.Marshal()
	if err != nil {
		logger.Error(fmt.Sprintf("Marshal calldata for %s failed", methodName), err)
		return nil, err
	}

	relatedAddress := []e_common.Address{}
	maxGas := uint64(5_000_000)
	maxGasPrice := uint64(1_000_000_000)
	timeUse := uint64(0)

	for attempt := 1; attempt <= attempts; attempt++ {
		ch := make(chan model.ResultData, 1)
		go func() {
			receipt, err := chainClient.SendTransactionWithDeviceKey(
				from,
				to,
				big.NewInt(0),
				bData,
				relatedAddress,
				maxGas,
				maxGasPrice,
				timeUse,
			)
			ch <- model.ResultData{
				Receipt: receipt,
				Err:     err,
			}
		}()

		select {
		case res := <-ch:
			if res.Err != nil {
				logger.Error(fmt.Sprintf("SendTransactionWithDeviceKey error in %s", methodName), res.Err)
				return nil, res.Err
			}

			fmt.Printf("rc %s: %v\n", methodName, res.Receipt)

			if res.Receipt.Status() == pb.RECEIPT_STATUS_RETURNED {
				if unpackTo != "" {
					kq := make(map[string]interface{})
					err := abi.UnpackIntoMap(kq, unpackTo, res.Receipt.Return())
					if err != nil {
						logger.Error(fmt.Sprintf("UnpackIntoMap error for %s", methodName), err)
						return nil, err
					}
					return kq, nil
				}
				return true, nil
			}
			return hex.EncodeToString(res.Receipt.Return()), nil

		case <-time.After(10 * time.Second):
			logger.Error(fmt.Sprintf("Timeout in %s", methodName))
			if attempt < attempts {
				time.Sleep(1 * time.Second)
				continue
			}
			return nil, fmt.Errorf("timeout after %d attempts in %s", attempts, methodName)
		}
	}

	return nil, fmt.Errorf("unexpected error in %s", methodName)
}
func (h *sendTransactionService) AddNoti(
	title string,
	body string,
	to common.Address,
) (interface{}, error) {
	fmt.Println("AddNoti")
	params := struct {
		Title string
		Body  string
	}{
		Title: title,
		Body:  body,
	}
	input, err := h.notiStorageAbi.Pack(
		"AddNoti",
		params,
		to,
	)
	if err != nil {
		logger.Error("error when pack call data AddNoti", err)
		return nil, err
	}
	return h.sendTransactionAndGetResult(
		h.chainClientAdminNoti,
		h.adminNoti,
		h.notiStorageAddress,
		h.notiStorageAbi,
		"AddNoti", 
		input, 
		"", 
		3,
	)
}
func (h *sendTransactionService) CheckUserRegistered(
	to common.Address,
) (interface{}, error) {
	input, err := h.notiStorageAbi.Pack(
		"checkUserRegistered",
		to,
	)
	if err != nil {
		logger.Error("error when pack call data checkUserRegistered", err)
		return nil, err
	}
	return h.sendTransactionAndGetResult(
		h.chainClientAdminNoti,
		h.adminNoti,
		h.notiStorageAddress,	
		h.notiStorageAbi,	
		"AddNoti", 
		input, 
		"", 
		3,
	)
}
func (h *sendTransactionService) UpdateOtpStatus(
	parent common.Address,
	status bool,
) (interface{}, error) {
	fmt.Println("updateOtpStatus")
	input, err := h.miningUserAbi.Pack(
		"updateOtpStatus",
		parent,
		status,
	)
	if err != nil {
		logger.Error("error when pack call data updateOtpStatus", err)
		return nil, err
	}
	return h.sendTransactionAndGetResult(
		h.chainClientBeMinningUser,
		h.beMinningUser,
		h.miningUserAddress,
		h.miningUserAbi,
		"updateOtpStatus", 
		input, 
		"", 
		3,
	)

}

func (h *sendTransactionService) ActiveUserByBe(
	parent common.Address,
	otp [32]byte,
) (interface{}, error) {
	fmt.Println("ActiveUserByBe")
	input, err := h.miningUserAbi.Pack(
		"activeUserByBe",
		parent,
		otp,
	)
	fmt.Println("input:",hex.EncodeToString(input))
	if err != nil {
		logger.Error("error when pack call data activeUserByBe", err)
		return nil, err
	}
	return h.sendTransactionAndGetResult(
		h.chainClientBeMinningUser,
		h.beMinningUser,
		h.miningUserAddress,
		h.miningUserAbi,
		"activeUserByBe", 
		input, 
		"", 
		3,
	)
}

// func (h *sendTransactionService) UpdateOtpStatus(
// 	parent common.Address,
// 	status bool,
// ) (interface{}, error) {
// 	fmt.Println("updateOtpStatus")
// 	var result interface{}
// 	input, err := h.miningUserAbi.Pack(
// 		"updateOtpStatus",
// 		parent,
// 		status,
// 	)
// 	if err != nil {
// 		logger.Error("error when pack call data updateOtpStatus", err)
// 		return nil, err
// 	}
// 	callData := transaction.NewCallData(input)

// 	bData, err := callData.Marshal()
// 	if err != nil {
// 		logger.Error("error when marshal call data updateOtpStatus", err)
// 		return nil, err
// 	}
// 	fmt.Println("input: ", hex.EncodeToString(bData))
// 	relatedAddress := []e_common.Address{}
// 	maxGas := uint64(5_000_000)
// 	maxGasPrice := uint64(1_000_000_000)
// 	timeUse := uint64(0)
// 	receipt, err := h.chainClient.SendTransactionWithDeviceKey(
// 		h.beMinningUser,
// 		h.miningUserAddress,
// 		big.NewInt(0),
// 		// 4,
// 		bData,
// 		relatedAddress,
// 		maxGas,
// 		maxGasPrice,
// 		timeUse,
// 	)
// 	fmt.Println("rc updateOtpStatus:", receipt)
// 	if err != nil {
// 		return result, err
// 	}
// 	if receipt.Status() == pb.RECEIPT_STATUS_RETURNED {
// 		logger.Info("updateOtpStatus - Result - Success")
// 	} else {
// 		result = hex.EncodeToString(receipt.Return())
// 		logger.Info("updateOtpStatus - Result - ", result)

// 	}
// 	return result, nil
// }

// func (h *sendTransactionService) AddNoti(
// 	title string,
// 	body string,
// 	to common.Address,
// ) (interface{}, error) {
// 	fmt.Println("AddNoti")
// 	var result interface{}
// 	params := struct {
// 		Title string
// 		Body  string
// 	}{
// 		Title: title,
// 		Body:  body,
// 	}
// 	input, err := h.notiStorageAbi.Pack(
// 		"AddNoti",
// 		params,
// 		to,
// 	)
// 	if err != nil {
// 		logger.Error("error when pack call data AddNoti", err)
// 		return nil, err
// 	}
// 	callData := transaction.NewCallData(input)

// 	bData, err := callData.Marshal()
// 	if err != nil {
// 		logger.Error("error when marshal call data AddNoti", err)
// 		return nil, err
// 	}
// 	fmt.Println("input: ", hex.EncodeToString(bData))
// 	relatedAddress := []e_common.Address{}
// 	maxGas := uint64(5_000_000)
// 	maxGasPrice := uint64(1_000_000_000)
// 	timeUse := uint64(0)
// 	receipt, err := h.chainClient.SendTransactionWithDeviceKey(
// 		h.adminNoti,
// 		h.notiStorageAddress,
// 		big.NewInt(0),
// 		// 4,
// 		bData,
// 		relatedAddress,
// 		maxGas,
// 		maxGasPrice,
// 		timeUse,
// 	)
// 	fmt.Println("rc AddNoti:", receipt)
// 	if receipt.Status() == pb.RECEIPT_STATUS_RETURNED {
// 		logger.Info("AddNoti - Result - Success")
// 	} else {
// 		result = hex.EncodeToString(receipt.Return())
// 		logger.Info("AddNoti - Result - ", result)

// 	}
// 	return result, nil
// }
// func (h *sendTransactionService) CheckUserRegistered(
// 	to common.Address,
// ) (interface{}, error) {
// 	var result interface{}
// 	input, err := h.notiStorageAbi.Pack(
// 		"checkUserRegistered",
// 		to,
// 	)
// 	if err != nil {
// 		logger.Error("error when pack call data checkUserRegistered", err)
// 		return nil, err
// 	}
// 	callData := transaction.NewCallData(input)

// 	bData, err := callData.Marshal()
// 	if err != nil {
// 		logger.Error("error when marshal call data checkUserRegistered", err)
// 		return nil, err
// 	}
// 	fmt.Println("input: ", hex.EncodeToString(bData))
// 	fmt.Println("h.adminNoti:", h.adminNoti)
// 	fmt.Println("h.notiStorageAddress:", h.notiStorageAddress)
// 	relatedAddress := []e_common.Address{}
// 	maxGas := uint64(5_000_000)
// 	maxGasPrice := uint64(1_000_000_000)
// 	timeUse := uint64(0)
// 	receipt, err := h.chainClient.SendTransactionWithDeviceKey(
// 		h.adminNoti,
// 		h.notiStorageAddress,
// 		big.NewInt(0),
// 		// 4,
// 		bData,
// 		relatedAddress,
// 		maxGas,
// 		maxGasPrice,
// 		timeUse,
// 	)
// 	fmt.Println("rc checkUserRegistered:", receipt)
// 	if receipt.Status() == pb.RECEIPT_STATUS_RETURNED {
// 		kq := make(map[string]interface{})
// 		err = h.notiStorageAbi.UnpackIntoMap(kq, "checkUserRegistered", receipt.Return())
// 		if err != nil {
// 			logger.Error("UnpackIntoMap")
// 			return nil, err
// 		}
// 		result = kq[""]
// 		logger.Info("checkUserRegistered - Result - Success", result)
// 		return result, nil

// 	} else {
// 		result = hex.EncodeToString(receipt.Return())
// 		logger.Info("checkUserRegistered - Result - ", result)
// 		return result, nil
// 	}
// }
