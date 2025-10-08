package database

import (
	"context"
	"log"
	"os"
	"time"

	// Mongo
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	// Redis
	"github.com/redis/go-redis/v9"

	// Elasticsearch (v8)
	"github.com/elastic/go-elasticsearch/v8"
)

var (
	// --- MongoDB ---
	MongoAuthDB         *mongo.Database
	MongoOrdersDB       *mongo.Database
	MongoAddressesDB    *mongo.Database
	MongoCompanyDB      *mongo.Database
	MongoCompanyUsersDB *mongo.Database

	// --- Nouvelles bases séparées ---
	MongoProductsDB     *mongo.Database
	MongoCategoriesDB   *mongo.Database

	// --- Redis & Elastic ---
	RedisClient   *redis.Client
	ElasticClient *elasticsearch.Client
)

// ✅ Initialise toutes les connexions (Mongo, Redis, Elastic)
func ConnectDatabases() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connectMongo(ctx)
	connectRedis(ctx)
	connectElastic()

	log.Println("✅ Toutes les bases (Mongo + Redis + Elastic) sont connectées")
}

//
// --- MONGO ---
//
func connectMongo(ctx context.Context) {
	// --- AUTH ---
	clientAuth, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_AUTH_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo AUTH:", err)
	}
	MongoAuthDB = clientAuth.Database("db_auth")

	// --- ORDERS ---
	clientOrders, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_ORDERS_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo ORDERS:", err)
	}
	MongoOrdersDB = clientOrders.Database("db_orders")

	// --- ADDRESSES ---
	clientAddresses, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_ADDRESSES_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo ADDRESSES:", err)
	}
	MongoAddressesDB = clientAddresses.Database("db_addresses")

	// --- COMPANY ---
	clientCompany, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_COMPANY_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo COMPANY:", err)
	}
	MongoCompanyDB = clientCompany.Database("db_company")

	// --- PRODUITS ---
	clientProducts, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_PRODUCTS_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo PRODUCTS:", err)
	}
	MongoProductsDB = clientProducts.Database("db_products")

	// --- CATEGORIES ---
	clientCategories, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_CATEGORIES_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo CATEGORIES:", err)
	}
	MongoCategoriesDB = clientCategories.Database("db_categories")

	log.Println("✅ Connecté à toutes les bases MongoDB (Auth, Orders, Company, Products, Categories)")
}

//
// --- REDIS ---
//
func connectRedis(ctx context.Context) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"), // ex: "localhost:6379"
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("❌ Erreur connexion Redis:", err)
	}
	log.Println("✅ Connecté à Redis")
}

//
// --- ELASTICSEARCH ---
//
func connectElastic() {
	cfg := elasticsearch.Config{
		Addresses: []string{
			os.Getenv("ELASTIC_URL"), // ex: "http://localhost:9200"
		},
		Username: os.Getenv("ELASTIC_USER"),
		Password: os.Getenv("ELASTIC_PASSWORD"),
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatal("❌ Erreur création client Elasticsearch:", err)
	}

	res, err := client.Info()
	if err != nil {
		log.Fatal("❌ Erreur connexion Elasticsearch:", err)
	}
	defer res.Body.Close()

	ElasticClient = client
	log.Println("✅ Connecté à Elasticsearch")
}
