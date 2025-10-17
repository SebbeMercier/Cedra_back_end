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

	// MinIO
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	// --- MongoDB ---
	MongoAuthDB         *mongo.Database
	MongoOrdersDB       *mongo.Database
	MongoAddressesDB    *mongo.Database
	MongoCompanyDB      *mongo.Database
	MongoCompanyUsersDB *mongo.Database

	MongoProductsDB   *mongo.Database
	MongoCategoriesDB *mongo.Database

	// --- Redis, Elastic, MinIO ---
	RedisClient   *redis.Client
	ElasticClient *elasticsearch.Client
	MinioClient   *minio.Client
)

//
// === INITIALISATION GLOBALE ===
//
func ConnectDatabases() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connectMongo(ctx)
	connectRedis(ctx)
	connectElastic()
	connectMinIO(ctx)

	log.Println("✅ Toutes les bases (Mongo + Redis + Elastic + MinIO) sont connectées")
}

//
// --- MONGO ---
//
func connectMongo(ctx context.Context) {
	clientAuth, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_AUTH_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo AUTH:", err)
	}
	MongoAuthDB = clientAuth.Database("db_auth")

	clientOrders, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_ORDERS_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo ORDERS:", err)
	}
	MongoOrdersDB = clientOrders.Database("db_orders")

	clientAddresses, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_ADDRESSES_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo ADDRESSES:", err)
	}
	MongoAddressesDB = clientAddresses.Database("addresses_db")

	clientCompany, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_COMPANY_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo COMPANY:", err)
	}
	MongoCompanyDB = clientCompany.Database("company_db")

	clientProducts, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_PRODUCTS_URL")))
	if err != nil {
		log.Fatal("❌ Erreur connexion Mongo PRODUCTS:", err)
	}
	MongoProductsDB = clientProducts.Database("db_products")

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
		Addr:     os.Getenv("REDIS_HOST"),
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
			os.Getenv("ELASTIC_URL"),
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

//
// --- MINIO ---
//
func connectMinIO(ctx context.Context) {
	endpoint := os.Getenv("MINIO_ENDPOINT")       // ex: "localhost:9000"
	accessKey := os.Getenv("MINIO_ACCESS_KEY")    // ex: "admin"
	secretKey := os.Getenv("MINIO_SECRET_KEY")    // ex: "password"
	useSSL := os.Getenv("MINIO_USE_SSL") == "true" // "true" ou "false"

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal("❌ Erreur connexion MinIO:", err)
	}

	MinioClient = client
	log.Println("✅ Connecté à MinIO :", endpoint)

	// ✅ Crée automatiquement le bucket s’il n’existe pas
	bucketName := os.Getenv("MINIO_BUCKET")
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		log.Fatal("❌ Erreur vérification bucket MinIO:", err)
	}
	if !exists {
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatal("❌ Erreur création bucket MinIO:", err)
		}
		log.Println("🪣 Bucket créé :", bucketName)
	} else {
		log.Println("🪣 Bucket MinIO déjà présent :", bucketName)
	}
}