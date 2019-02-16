package api

import (
        "fmt"
        "net/http"
        "time"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

        "github.com/omnom-nom/apiserver"
)

const (
        // APIServerStartupTimeout ...
        APIServerStartupTimeout = 5 * time.Second
        // APIServerStartupWaitPause ...
        APIServerStartupWaitPause = 500 * time.Millisecond

	DbIP = "192.168.1.101"
	DbPort = 8000
	DbZone = "us-west-2"
)

var (
	env  *EnvSingleton
	once sync.Once
)

func handleCrash(w http.ResponseWriter) {
        crash := recover()
        if crash == nil {
                return
        }
        log.Error(crash)
}

func initDb() *ApiDb {

	dbUrl := fmt.Sprintf("http://%s:%s", DbIP, DbPort)

	config := &aws.Config{
		Region:   aws.String(DbZone),
		Endpoint: aws.String(dbUrl),
	}

	sess := session.Must(session.NewSession(config))

	return &ApiDb{dynamodb.New(sess)}
}


func GetEnvInstance() *EnvSingleton {

	once.Do(func() {
		env.db = initDb()
	})

	return env
}

func Init() error {

        factory, err := apiserver.FactoryForGorillaMux()
        if err != nil {
                log.Errorf("failed to create mux: %v",err)
                return fmt.Errorf("failed to create mux: %v", err)
        }

        // register middleware objects with factory
        factory.Default(apiserver.MiddlewareLogger, apiserver.Logger())
        factory.Always("crash-handler", apiserver.NewCrashHandler(handleCrash))

        secureMux, err := factory.Make(routes)
        if err != nil {
                log.Errorf("failed to do factory make: %v",err)
                return fmt.Errorf("failed to do factory make: %v", err)
        }

        httpServer, err := apiserver.New(secureMux, apiserver.ServerAddress(fmt.Sprintf("%s:%d", "0.0.0.0", 8080)))
        if err != nil {
                log.Errorf("failed to create HTTP API server: %v",err)
                return fmt.Errorf("failed to create HTTP API server: %s", err)
        }

        if err = httpServer.StartHTTP(); err != nil {
                log.Errorf("failed to start HTTPS API server: %s", err)
                return fmt.Errorf("failed to start HTTPS API server: %v", err)
        }
        defer func() {
                if !httpServer.IsStopped() {
                        if err := httpServer.Stop(); err != nil {
                                errMsg := fmt.Sprintf("failed to stop HTTP server: %s", err)
                                fmt.Println(errMsg)
                        }
                }
        }()

	waitUntil := time.Now().Add(APIServerStartupTimeout)
        for waitUntil.After(time.Now()) {
                if httpServer.IsRunning() {
                        break
                }
                if httpServer.IsStopped() {
                        log.Error("http server has stopped, can not continue")
                        return fmt.Errorf("http server has stopped, can not continue")
                }

                fmt.Println("waiting for api servers to start...")
                time.Sleep(APIServerStartupWaitPause)
        }

        if !httpServer.IsRunning() {
                log.Error("http server is not running")
                return fmt.Errorf("http server is not running after %s", APIServerStartupTimeout)
        }

        log.Infof("http server is running: %s", httpServer.Endpoint())

	for {
                time.Sleep(120 * time.Second)
                continue
        }

	return nil
}
