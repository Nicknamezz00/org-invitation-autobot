package store

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	viper.SetConfigFile("../../config/config.yaml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("env")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}
}

func New(v *viper.Viper) *gorm.DB {
	if v == nil {
		panic("nil viper")
	}
	return connectPostgreSQL(v.Sub("postgres"))
}

func connectPostgreSQL(v *viper.Viper) *gorm.DB {
	var kv []string
	for _, key := range []string{"host", "user", "password", "dbname", "port"} {
		kv = append(kv, fmt.Sprintf("%s=%s", key, v.GetString(key)))
	}

	dsn := fmt.Sprintf("%s sslmode=disable TimeZone=Asia/Shanghai", strings.Join(kv, " "))
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}
