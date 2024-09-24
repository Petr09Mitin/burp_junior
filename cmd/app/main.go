package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	mongo_repo "github.com/burp_junior/internal/repository/mongo"
	"github.com/burp_junior/internal/rest/routers"
	"github.com/burp_junior/usecase/request"
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

	repo := mongo_repo.NewRequestsRepo(collection)
	rs, err := request.NewRequestService(repo)
	if err != nil {
		log.Println("err creating request service: ", err)
		return
	}

	go func() {
		routers.MountProxyRouter(rs)
	}()

	routers.MountAPIRouter(rs)
}

func main() {
	mountRouters()
}
