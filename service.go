package kiplestar

import (
	"context"
	redisv8 "github.com/go-redis/redis/v8"
	"github.com/m2c/kiplestar/config"
	"github.com/m2c/kiplestar/iris"
	"github.com/m2c/kiplestar/kafka"
	"github.com/m2c/kiplestar/kipledb"
	"github.com/m2c/kiplestar/redis"
	"sync"
)

//we need create the single object but thread safe
var kipleInstance *kipleSever

var once sync.Once

type kipleSever struct {
	app   iris.App
	redis redis.Redis
	db    kipledb.KipleDB
	kafka kafka.Kafka
}
type Server_Option int

const (
	Mysql_service = iota + 1
	Redis_service
)

//create the single object
func GetKipleServerInstance() *kipleSever {
	once.Do(func() {
		kipleInstance = new(kipleSever)
	})
	return kipleInstance
}
func (slf *kipleSever) Default() {
	slf.app.Default()
}
func (slf *kipleSever) New() {
	slf.app.New()
}

//return app
func (slf *kipleSever) App() *iris.App {
	return &slf.app
}
func (slf *kipleSever) DB() *kipledb.KipleDB {
	return &slf.db
}
func (slf *kipleSever) Redis() *redisv8.Client {
	return slf.redis.Redis()
}
func (slf *kipleSever) LoadCustomizeConfig(slfConfig interface{}) error {
	return config.LoadCustomizeConfig(slfConfig)
}

//need call this function after Option
func (slf *kipleSever) StartServer(opt ...Server_Option) error {

	for _, v := range opt {
		var err error
		switch v {
		case Mysql_service:
			err = slf.db.StartDb()
		case Redis_service:
			err = slf.redis.StartRedis()
		}
		if err != nil {
			return err
		}
	}

	return slf.app.Start()
}

func (slf *kipleSever) KafkaService(ctx context.Context, topic string, callBackChan chan []byte) {
	slf.kafka.KafkaReceiver(ctx, topic, callBackChan)
}
