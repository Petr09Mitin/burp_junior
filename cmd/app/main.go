package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/burp_junior/internal/rest/routers"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	DotenvPath       = "../.env"
	MongoHostEnv     = "MONGO_HOST"
	MongoPortEnv     = "MONGO_PORT"
	MongoUsernameEnv = "MONGO_INITDB_ROOT_USERNAME"
	MongoPasswordEnv = "MONGO_INITDB_ROOT_PASSWORD"
)

func mountRouters() {
	if err := godotenv.Load(); err != nil {
		log.Println("unable to read .env file")
		return
	}

	connString := fmt.Sprintf(
		"mongodb://%s:%s@%s:%s",
		os.Getenv(MongoUsernameEnv),
		os.Getenv(MongoPasswordEnv),
		os.Getenv(MongoHostEnv),
		os.Getenv(MongoPortEnv),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connString))
	if err != nil {
		log.Println("err connecting to mongo: ", err)
		return
	}

	collection := client.Database("burp_junior").Collection("request")

	go func() {
		routers.MountProxyRouter(collection)
	}()

	routers.MountAPIRouter(collection)
}

func main() {
	mountRouters()
}
