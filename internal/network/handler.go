package network

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	// "github.com/ethereum/go-ethereum/common"
	e_common "github.com/ethereum/go-ethereum/common"
	"github.com/meta-node-blockchain/meta-node/pkg/logger"
	"github.com/meta-node-blockchain/mining-service/internal/config"

	// "github.com/meta-node-blockchain/mining-service/internal/database"
	"github.com/meta-node-blockchain/mining-service/internal/model"
	"github.com/meta-node-blockchain/mining-service/internal/services"
	"github.com/meta-node-blockchain/mining-service/internal/usecase"
	"github.com/meta-node-blockchain/mining-service/internal/utils"
	"github.com/syndtr/goleveldb/leveldb"
)

type MiningUser struct {
	config           *config.AppConfig
	service          services.SendTransactionService
	miningUserABI          *abi.ABI
	ServerPrivateKey string
	DB               *leveldb.DB
	storedPubKey     string
	eventChan        chan model.EventLog
	cancelMonitors   map[string]context.CancelFunc
	cancelMu         sync.Mutex
	usecase 		usecase.OtpUsecase
}

func NewMiningUserEventHandler(
	config *config.AppConfig,
	service services.SendTransactionService,
	miningUserABI *abi.ABI,
	ServerPrivateKey string,
	DB *leveldb.DB,
	storedPubKey string,
	eventChan chan model.EventLog,
	usecase 		usecase.OtpUsecase,
) *MiningUser {
	return &MiningUser{
		config:           config,
		service:          service,
		miningUserABI:          miningUserABI,
		ServerPrivateKey: ServerPrivateKey,
		DB:               DB,
		storedPubKey:     storedPubKey,
		eventChan:        eventChan,
		cancelMonitors:   make(map[string]context.CancelFunc),
		usecase:usecase,
	}
}
func (h *MiningUser) ListenEvents() {
	go func() {
		logger.Info("⏳ Start listening for new events...")

		rpcURL := h.config.RpcURL
		contractAddress := h.config.MiningUserAddress

		// Đọc ABI
		abiBytes, err := os.ReadFile(h.config.MiningUserABIPath)
		if err != nil {
			logger.Error("Error reading ABI file:", err)
			return
		}
		abiJSON := string(abiBytes)

		// Lấy topic0 cho UserRef và UserProcessing
		userRefTopic, err := utils.GetTopic0FromABI(abiJSON, "UserRef")
		if err != nil {
			logger.Error("Error getting UserRef topic0:", err)
			return
		}
		userProcessingTopic, err := utils.GetTopic0FromABI(abiJSON, "UserProcessing")
		if err != nil {
			logger.Error("Error getting UserProcessing topic0:", err)
			return
		}
		var lastBlock string

		for {
			select {
			default:
				// Lấy latest block
				block, err := utils.GetLatestBlockNumber(rpcURL)
				if err != nil {
					logger.Error("Failed to get latest block:", err)
					time.Sleep(2 * time.Second)
					continue
				}

				if block == lastBlock {
					time.Sleep(1 * time.Second)
					continue
				}
				lastBlock = block

				// Lặp qua từng topic để lấy log
				topics := []string{userRefTopic, userProcessingTopic}
				for _, topic := range topics {
					logs, err := utils.GetLogs(rpcURL, block, block, contractAddress, topic)
					if err != nil {
						logger.Error("Error fetching logs for topic", topic, ":", err)
						time.Sleep(1 * time.Second)
						continue
					}

					for _, raw := range logs {
						var log model.EventLog
						if err := json.Unmarshal(raw, &log); err != nil {
							logger.Warn("Cannot decode event log:", err)
							continue
						}
						h.eventChan <- log
					}
				}

				time.Sleep(1 * time.Second)
			}
		}
	}()
}

func (h *MiningUser) HandleConnectSmartContract(event model.EventLog) {
	fmt.Println("event la:", event)
	switch event.Topics[0] {
	case h.miningUserABI.Events["UserRef"].ID.String():
		h.handleUserRef(event)
	case h.miningUserABI.Events["UserProcessing"].ID.String():
		h.handleUserProcessing(event)
	}
}
func (h *MiningUser) handleUserRef(event model.EventLog) {
	fmt.Println("handleUserRef")
	if len(event.Topics) < 3 {
		logger.Warn("Not enough topics in UserRef event")
		return
	}
	// Lấy referal và referer từ indexed topics
	referal := common.HexToAddress(event.Topics[1])
	fmt.Println("referal la:",referal)
	referer := common.HexToAddress(event.Topics[2])
	fmt.Println("referer la:",referer)

	//check if referal subcribe to receive noti
	isSubcribed, err := h.service.CheckUserRegistered(referal)
	if err != nil {
        fmt.Println("Error when call CheckUserRegistered:", err)
        return
    }
	fmt.Println("isSubcribed:",isSubcribed)
	subcribed,ok := isSubcribed.(bool)
	if !ok {
		fmt.Println("Error when parse isSubcribed:", err)
        return
	}
	if !subcribed{
		fmt.Println("User havent subsribe to received noti yet")
        return
	}
	result := make(map[string]interface{})
	err = h.miningUserABI.UnpackIntoMap(result, "UserRef", e_common.FromHex(event.Data))
	if err != nil {
		logger.Error("can't unpack to map", err)
		return
	}
	referralEncryptTokenNoti, ok := result["_referralEncryptTokenNoti"].(string)
	if !ok {
		logger.Error("fail in parse _referralEncryptTokenNoti:", err)
		return
	}
	fmt.Println("referralEncryptTokenNoti:",referralEncryptTokenNoti)
	Otp := utils.GenerateVerificationToken()
	request := model.OtpVerificationRequest{
		Referal: referal.Hex(),
	}
	//save otp in db with expireTime
	h.usecase.OtpVerification(request,Otp)
	title := "refUserViaQRCode success"
	kq:= map[string]interface{}{
		"referer" : referer,
		"otp" : Otp,
	}
	body,err := json.Marshal(kq)
	if err != nil {
        fmt.Println("Error when marshal body:", err)
        return
    }
	//if yes, send noti to referal(nguoi duoc gioi thieu)
	_,err = h.service.AddNoti(title,string(body),referal)
	if err != nil {
		logger.Error("fail in AddNoti:", err)
		return
	}

}

func (h *MiningUser) handleUserProcessing(event model.EventLog) {
	fmt.Println("handleUserProcessing")
	result := make(map[string]interface{})
	err := h.miningUserABI.UnpackIntoMap(result, "UserProcessing", e_common.FromHex(event.Data))
	if err != nil {
		logger.Error("can't unpack to map handleUserProcessing", err)
		return
	}
	user := common.HexToAddress(event.Topics[1])
	fmt.Println("referal la:",user)
	otp, ok := result["OTP"].([32]byte)
	if !ok {
		logger.Error("fail in parse OTP handleUserProcessing:", err)
		return
	}
	parent, ok := result["parent"].(common.Address)
	if !ok {
		logger.Error("fail in parse parent handleUserProcessing:", err)
		return
	}
	fmt.Println("user:",user)
	fmt.Println("otp:",hex.EncodeToString(otp[:]))
	request := model.OtpAuthenticationRequest{
		Referal: user.Hex(),
		Otp: hex.EncodeToString(otp[:]),
	}
	err = h.usecase.OtpAuthentication(request)
	if err == nil{
		fmt.Println("OtpAuthentication passed")
		//authentication success
		_,err = h.service.UpdateOtpStatus(user,true)
		if err != nil {
			logger.Error("fail in UpdateOtpStatus:", err)
			return
		}
		h.service.ActiveUserByBe(parent,otp)
	}else{
		logger.Error("fail in OtpAuthentication:", err)
		return
	}

}
