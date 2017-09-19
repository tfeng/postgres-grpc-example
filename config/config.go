package config

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"os"
)

var (
	Db         = connect()
	Logger, _  = zap.NewDevelopment()
	PrivateKey = generateKey()
	Sugar      = Logger.Sugar()
)

func generateKey() *rsa.PrivateKey {
	if key, err := rsa.GenerateKey(rand.Reader, 1024); err != nil {
		Logger.Fatal("Unable to generate RSA key", zap.Error(err))
		return nil
	} else {
		return key
	}
}

func connect() *pg.DB {
	var addr = os.Getenv("POSTGRESQL_ADDRESS")
	if addr == "" {
		addr = "127.0.0.1:5432"
	}
	return pg.Connect(&pg.Options{
		Addr:     addr,
		User:     "postgres",
		Password: "password",
	})
}
