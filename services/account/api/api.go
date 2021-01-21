package api

import (
	"cloudservices/account/config"
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"fmt"

	"github.com/go-redis/redis"

	"google.golang.org/grpc"
)

// ObjectModelAPI captures all object model APIs
type APIServer interface {
	Register(gServer *grpc.Server)
	Close() error
	GetKeyService() crypto.KeyService
	gapi.AccountServiceServer
}

type apiServer struct {
	*base.DBObjectModelAPI
	keyService crypto.KeyService
}

// NewObjectModelAPI creates a ObjectModelAPI based on sql DB
// Note: api with the same config should be shared
// to minimize resource usage
func NewAPIServer() (APIServer, error) {
	dbURL, err := base.GetDBURL(*config.Cfg.SQL_Dialect, *config.Cfg.SQL_DB, *config.Cfg.SQL_User, *config.Cfg.SQL_Password, *config.Cfg.SQL_Host, *config.Cfg.SQL_Port, *config.Cfg.DisableDBSSL)
	if err != nil {
		return nil, err
	}
	roDbURL := dbURL
	if config.Cfg.SQL_ReadOnlyHost != nil && len(*config.Cfg.SQL_ReadOnlyHost) > 0 {
		roDbURL, err = base.GetDBURL(*config.Cfg.SQL_Dialect, *config.Cfg.SQL_DB, *config.Cfg.SQL_User, *config.Cfg.SQL_Password, *config.Cfg.SQL_ReadOnlyHost, *config.Cfg.SQL_Port, *config.Cfg.DisableDBSSL)
		if err != nil {
			return nil, err
		}
	}
	var redisClient *redis.Client
	if !*config.Cfg.DisableScaleOut {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:6379", *config.Cfg.RedisHost),
			Password: "", // no password set
			DB:       0,  // use default DB
		})
	}
	dbAPI, err := base.NewDBObjectModelAPI(*config.Cfg.SQL_Dialect, dbURL, roDbURL, redisClient)
	if err != nil {
		return nil, err
	}
	// custom DB configurations
	db := dbAPI.GetDB()
	roDB := dbAPI.GetReadOnlyDB()
	haveReadonlyDB := db != roDB

	maxCnx := *config.Cfg.SQL_MaxCnx
	if maxCnx != 0 {
		db.SetMaxOpenConns(maxCnx)
		if haveReadonlyDB {
			roDB.SetMaxOpenConns(maxCnx)
		}
	}

	maxIdleCnx := *config.Cfg.SQL_MaxIdleCnx
	if maxIdleCnx != 0 {
		db.SetMaxIdleConns(maxIdleCnx)
		if haveReadonlyDB {
			roDB.SetMaxIdleConns(maxIdleCnx)
		}
	}

	maxCnxLife := *config.Cfg.SQL_MaxCnxLife
	if maxCnxLife != 0 {
		db.SetConnMaxLifetime(maxCnxLife)
		if haveReadonlyDB {
			roDB.SetConnMaxLifetime(maxCnxLife)
		}
	}

	keyService := crypto.NewKeyService(*config.Cfg.AWSRegion, *config.Cfg.JWTSecret, *config.Cfg.AWSKMSKey, *config.Cfg.UseKMS)
	apiServer := &apiServer{DBObjectModelAPI: dbAPI, keyService: keyService}
	return apiServer, err
}

func (server *apiServer) Register(gServer *grpc.Server) {
	gapi.RegisterAccountServiceServer(gServer, server)
}

func (server *apiServer) GetKeyService() crypto.KeyService {
	return server.keyService
}

func getPagingParams(paging *gapi.Paging) (base.PageToken, int) {
	startToken := base.StartPageToken
	if len(paging.GetStartToken()) > 0 {
		startToken = base.PageToken(paging.GetStartToken())
	}
	rowSize := base.MaxRowsLimit
	if paging.GetSize() > 0 {
		rowSize = int(paging.GetSize())
	}
	return startToken, rowSize
}
