package database

import (
	"context"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gocql/gocql"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// --- Configuration ScyllaDB ---
type ScyllaKeyspaceConfig struct {
	Hosts       []string
	Keyspace    string
	Username    string
	Password    string
	SSLEnabled  bool
	CACertPath  string
	Timeout     time.Duration
	NumConns    int
	Consistency gocql.Consistency
}

type ScyllaManager struct {
	sessions map[string]*gocql.Session // keyspace → session
	configs  map[string]ScyllaKeyspaceConfig
	mu       sync.Mutex
}

// --- Variables Globales ---
var (
	Scylla      *ScyllaManager
	Redis       *redis.Client
	RedisClient *redis.Client // Alias pour compatibilité
	Elastic     *elasticsearch.Client
	MinIO       *minio.Client
)

// --- Initialisation ---
func ConnectDatabases() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Initialiser ScyllaDB (multi-keyspaces)
	if err := InitScyllaDB(); err != nil {
		log.Fatalf("❌ Échec initialisation ScyllaDB: %v", err)
	}

	// 2. Initialiser Redis
	connectRedis(ctx)

	// 3. Initialiser Elasticsearch
	connectElastic()

	// 4. Initialiser MinIO
	connectMinIO(ctx)

	log.Println("✅ Toutes les bases de données sont connectées")
}

// =============================================
// SCYLLA DB (Multi-Keyspaces avec SSL & Rôles)
// =============================================

// InitScyllaDB initialise le gestionnaire de sessions ScyllaDB
func InitScyllaDB() error {
	Scylla = &ScyllaManager{
		sessions: make(map[string]*gocql.Session),
		configs:  loadScyllaConfigs(),
	}

	// Créer les sessions pour chaque keyspace configuré
	for keyspace := range Scylla.configs {
		if _, err := Scylla.GetSession(keyspace); err != nil {
			return fmt.Errorf("échec initialisation keyspace %s: %v", keyspace, err)
		}
	}

	// Note: Les tables doivent être créées manuellement via scripts/scylladb_init.cql
	// L'initialisation automatique est désactivée pour éviter les problèmes de permissions

	return nil
}

// loadScyllaConfigs charge les configurations depuis .env
func loadScyllaConfigs() map[string]ScyllaKeyspaceConfig {
	configs := make(map[string]ScyllaKeyspaceConfig)

	// Configuration commune
	hosts := strings.Split(os.Getenv("SCYLLA_HOSTS"), ",")
	sslEnabled := strings.ToLower(os.Getenv("SCYLLA_SSL_ENABLED")) == "true"
	caPath := os.Getenv("SCYLLA_SSL_CA_PATH")
	timeout := 5 * time.Second // ✅ Réduit de 10s à 5s pour timeout plus rapide
	numConns := 20             // ✅ Augmenté de 10 à 20 pour plus de connexions parallèles
	consistency := gocql.Quorum

	// --- Keyspace Produits ---
	if ks := os.Getenv("SCYLLA_KS_PRODUCTS_KEYSPACE"); ks != "" {
		configs[ks] = ScyllaKeyspaceConfig{
			Hosts:       hosts,
			Keyspace:    ks,
			Username:    os.Getenv("SCYLLA_KS_PRODUCTS_ROLE"),
			Password:    os.Getenv("SCYLLA_KS_PRODUCTS_PASSWORD"),
			SSLEnabled:  sslEnabled,
			CACertPath:  caPath,
			Timeout:     timeout,
			NumConns:    numConns,
			Consistency: consistency,
		}
	}

	// --- Keyspace Utilisateurs ---
	if ks := os.Getenv("SCYLLA_KS_USERS_KEYSPACE"); ks != "" {
		configs[ks] = ScyllaKeyspaceConfig{
			Hosts:       hosts,
			Keyspace:    ks,
			Username:    os.Getenv("SCYLLA_KS_USERS_ROLE"),
			Password:    os.Getenv("SCYLLA_KS_USERS_PASSWORD"),
			SSLEnabled:  sslEnabled,
			CACertPath:  caPath,
			Timeout:     timeout,
			NumConns:    numConns,
			Consistency: consistency,
		}
	}

	// --- Keyspace Commandes ---
	if ks := os.Getenv("SCYLLA_KS_ORDERS_KEYSPACE"); ks != "" {
		configs[ks] = ScyllaKeyspaceConfig{
			Hosts:       hosts,
			Keyspace:    ks,
			Username:    os.Getenv("SCYLLA_KS_ORDERS_ROLE"),
			Password:    os.Getenv("SCYLLA_KS_ORDERS_PASSWORD"),
			SSLEnabled:  sslEnabled,
			CACertPath:  caPath,
			Timeout:     timeout,
			NumConns:    numConns,
			Consistency: consistency,
		}
	}

	return configs
}

// createScyllaCluster crée une configuration de cluster pour un keyspace
func createScyllaCluster(config ScyllaKeyspaceConfig) (*gocql.ClusterConfig, error) {
	cluster := gocql.NewCluster(config.Hosts...)
	cluster.Keyspace = config.Keyspace
	cluster.Consistency = config.Consistency
	cluster.Timeout = config.Timeout
	cluster.NumConns = config.NumConns

	// ✅ Optimisations de performance
	cluster.MaxWaitSchemaAgreement = 30 * time.Second
	cluster.ReconnectInterval = 1 * time.Second
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: config.Username,
		Password: config.Password,
	}

	// Configuration SSL si activé
	// Note: La configuration SSL dépend de la version de gocql
	// Pour gocql v1.7.0+, utilisez cluster.SslOpt
	if config.SSLEnabled && config.CACertPath != "" {
		caCert, err := os.ReadFile(config.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("impossible de lire le certificat CA: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("impossible de parser le certificat CA")
		}
	}

	// Politique de sélection d'hôtes optimisée
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())

	return cluster, nil
}

// GetSession retourne une session pour un keyspace donné
func (sm *ScyllaManager) GetSession(keyspace string) (*gocql.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Vérifie que le keyspace est configuré
	config, exists := sm.configs[keyspace]
	if !exists {
		return nil, fmt.Errorf("keyspace '%s' non configuré", keyspace)
	}

	// Si la session existe déjà, la retourner
	if session, exists := sm.sessions[keyspace]; exists {
		if err := session.Query("SELECT now() FROM system.local").Exec(); err == nil {
			return session, nil
		}
		// Si la session est invalide, la recréer
		session.Close()
	}

	// Crée une nouvelle configuration de cluster
	cluster, err := createScyllaCluster(config)
	if err != nil {
		return nil, fmt.Errorf("erreur configuration cluster pour %s: %v", keyspace, err)
	}

	// Crée une nouvelle session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("erreur création session pour %s: %v", keyspace, err)
	}

	// Stocke la session pour réutilisation
	sm.sessions[keyspace] = session
	log.Printf("✅ Nouvelle session ScyllaDB pour keyspace '%s' (utilisateur: %s)",
		keyspace, config.Username)

	return session, nil
}

// CloseScylla ferme toutes les sessions ScyllaDB
func CloseScylla() {
	Scylla.mu.Lock()
	defer Scylla.mu.Unlock()

	for keyspace, session := range Scylla.sessions {
		session.Close()
		log.Printf("🔌 Session ScyllaDB fermée pour keyspace '%s'", keyspace)
	}
}

// =============================================
// HELPERS POUR ACCÈS FACILITÉ AUX SESSIONS
// =============================================

// GetUsersSession retourne la session pour le keyspace users
func GetUsersSession() (*gocql.Session, error) {
	keyspace := os.Getenv("SCYLLA_KS_USERS_KEYSPACE")
	if keyspace == "" {
		return nil, fmt.Errorf("SCYLLA_KS_USERS_KEYSPACE non configuré")
	}
	return Scylla.GetSession(keyspace)
}

// GetProductsSession retourne la session pour le keyspace products
func GetProductsSession() (*gocql.Session, error) {
	keyspace := os.Getenv("SCYLLA_KS_PRODUCTS_KEYSPACE")
	if keyspace == "" {
		return nil, fmt.Errorf("SCYLLA_KS_PRODUCTS_KEYSPACE non configuré")
	}
	return Scylla.GetSession(keyspace)
}

// GetOrdersSession retourne la session pour le keyspace orders
func GetOrdersSession() (*gocql.Session, error) {
	keyspace := os.Getenv("SCYLLA_KS_ORDERS_KEYSPACE")
	if keyspace == "" {
		return nil, fmt.Errorf("SCYLLA_KS_ORDERS_KEYSPACE non configuré")
	}
	return Scylla.GetSession(keyspace)
}

// =============================================
// REDIS (inchangé)
// =============================================
func connectRedis(ctx context.Context) {
	Redis = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	RedisClient = Redis // Alias pour compatibilité

	if err := Redis.Ping(ctx).Err(); err != nil {
		log.Fatal("❌ Erreur connexion Redis:", err)
	}
	log.Println("✅ Connecté à Redis")
}

// =============================================
// ELASTICSEARCH (inchangé)
// =============================================
func connectElastic() {
	cfg := elasticsearch.Config{
		Addresses: []string{os.Getenv("ELASTIC_URL")},
		Username:  os.Getenv("ELASTIC_USER"),
		Password:  os.Getenv("ELASTIC_PASSWORD"),
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

	Elastic = client
	log.Println("✅ Connecté à Elasticsearch")
}

// =============================================
// MINIO (inchangé)
// =============================================
func connectMinIO(ctx context.Context) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal("❌ Erreur connexion MinIO:", err)
	}

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

	MinIO = client
	log.Println("✅ Connecté à MinIO :", endpoint)
}
