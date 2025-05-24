package app

import (
	"fmt"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/meta-node-blockchain/meta-node/cmd/client"
	c_config "github.com/meta-node-blockchain/meta-node/cmd/client/pkg/config"
	"github.com/meta-node-blockchain/meta-node/pkg/logger"
	"github.com/meta-node-blockchain/mining-service/internal/config"
	"github.com/meta-node-blockchain/mining-service/internal/database"
	"github.com/meta-node-blockchain/mining-service/internal/model"
	"github.com/meta-node-blockchain/mining-service/internal/network"
	"github.com/meta-node-blockchain/mining-service/internal/repositories"
	"github.com/meta-node-blockchain/mining-service/internal/services"
	"github.com/meta-node-blockchain/mining-service/internal/usecase"
)

type App struct {
	Config      *config.AppConfig
	ClientNoti *client.Client
	ClientBeMiningUser *client.Client
	EventChan   chan model.EventLog
	StopChan    chan bool
	MiningUser  *network.MiningUser
}

func NewApp(
	configPath string,
	loglevel int,
) (*App, error) {
	loggerConfig := &logger.LoggerConfig{
		Flag:    loglevel,
		Outputs: []*os.File{os.Stdout},
	}
	logger.SetConfig(loggerConfig)

	config, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal("invalid configuration", err)
		return nil, err
	}
	app := &App{}
	app.ClientNoti, err = client.NewClient(
		&c_config.ClientConfig{
			Version_:                config.MetaNodeVersion,
			PrivateKey_:             config.PrivateKeyAdminNoti,
			ParentAddress:           config.NotiOwnerAddress,
			ParentConnectionAddress: config.ParentConnectionAddress,
			DnsLink_:                config.DnsLink(),
			ConnectionAddress_:      config.ConnectionAddress_,
			ParentConnectionType:    config.ParentConnectionType,
			ChainId:                 config.ChainId,
		},
	)
	if err != nil {
		logger.Error(fmt.Sprintf("error when create chain client %v", err))
		return nil, err
	}
	app.ClientBeMiningUser, err = client.NewClient(
		&c_config.ClientConfig{
			Version_:                config.MetaNodeVersion,
			PrivateKey_:             config.PrivateKeyBeMiningUser,
			ParentAddress:           config.BeMiningUserAddress,
			ParentConnectionAddress: config.ParentConnectionAddress,
			DnsLink_:                config.DnsLink(),
			ConnectionAddress_:      config.ConnectionAddress_,
			ParentConnectionType:    config.ParentConnectionType,
			ChainId:                 config.ChainId,
		},
	)
	if err != nil {
		logger.Error(fmt.Sprintf("error when create chain client %v", err))
		return nil, err
	}
	app.EventChan = make(chan model.EventLog, 1000) // buffer 100 để tránh nghẽn
	leveldb, err := database.Open(config.PathLevelDB)
	readerMiningUser, err := os.Open(config.MiningUserABIPath)
	if err != nil {
		logger.Error("Error occured while read create miningUser smart contract abi")
		return nil, err
	}
	defer readerMiningUser.Close()

	miningUserAbi, err := abi.JSON(readerMiningUser)
	if err != nil {
		logger.Error("Error occured while parse create miningUser smart contract abi")
		return nil, err
	}
	readerNotiStorage, err := os.Open(config.NotiStorageABIPath)
	if err != nil {
		logger.Error("Error occured while read create miningUser smart contract abi")
		return nil, err
	}
	defer readerNotiStorage.Close()

	notiStorageAbi, err := abi.JSON(readerNotiStorage)
	if err != nil {
		logger.Error("Error occured while parse create miningUser smart contract abi")
		return nil, err
	}

	bserverPrivateKey, err := os.ReadFile(config.ServerPrivateKeyPath)
	if err != nil {
		logger.Error("Can not read private key pem file")
		return nil, err
	}
	bserverPublicKey, err := os.ReadFile(config.StoredPubKey)
	if err != nil {
		logger.Error("Can not read private key pem file")
		return nil, err
	}
	servs := services.NewSendTransactionService(
		app.ClientNoti,
		&notiStorageAbi,
		common.HexToAddress(config.NotiStorageAddress),
		common.HexToAddress(config.NotiOwnerAddress),
		app.ClientBeMiningUser,
		&miningUserAbi,
		common.HexToAddress(config.MiningUserAddress),
		common.HexToAddress(config.BeMiningUserAddress),
	)
	database.StartMySQL(config)
	db := database.GetMySqlConn()
	otpRepo := repositories.NewOtpRepository(db)
	otpUsecase := usecase.NewOtpUsecase(otpRepo)
	app.MiningUser = network.NewMiningUserEventHandler(
		config,
		servs,
		&miningUserAbi,
		string(bserverPrivateKey),
		leveldb,
		string(bserverPublicKey),
		app.EventChan,
		otpUsecase,
	)

	app.Config = config
	return app, nil
}

func (app *App) Run() {
	app.StopChan = make(chan bool)
	go app.MiningUser.ListenEvents() // BẮT ĐẦU LẮNG NGHE EVENT
	for {
		select {
		case <-app.StopChan:
			return
		case eventLogs := <-app.EventChan:
			fmt.Println("📩 Event Received:", eventLogs)
			app.MiningUser.HandleConnectSmartContract(eventLogs)

		}
	}
}

func (app *App) Stop() error {
	app.ClientNoti.Close()
	app.ClientBeMiningUser.Close()
	logger.Warn("App Stopped")
	return nil
}
