package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MongoAuthDB    *mongo.Database
	MongoCatalogDB *mongo.Database
	MongoOrdersDB  *mongo.Database
)

func ConnectMongo() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// DB Auth
	clientAuth, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_AUTH_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo AUTH:", err)
	}
	MongoAuthDB = clientAuth.Database("db_auth")

	// DB Catalog
	clientCatalog, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_CATALOG_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo CATALOG:", err)
	}
	MongoCatalogDB = clientCatalog.Database("db_catalog")

	// DB Orders
	clientOrders, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_ORDERS_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo ORDERS:", err)
	}
	MongoOrdersDB = clientOrders.Database("db_orders")

	log.Println("✅ Connecté à MongoDB")
}