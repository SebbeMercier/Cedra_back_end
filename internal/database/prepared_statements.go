package database

import (
	"log"
	"sync"

	"github.com/gocql/gocql"
)

var (
	// Prepared statements pour les requêtes fréquentes
	stmtGetUserByEmail    *gocql.Query
	stmtGetUserByID       *gocql.Query
	stmtInsertUser        *gocql.Query
	stmtInsertUserByEmail *gocql.Query
	stmtUpdateUser        *gocql.Query

	preparedOnce sync.Once
)

// InitPreparedStatements initialise les prepared statements
func InitPreparedStatements() {
	preparedOnce.Do(func() {
		session, err := GetUsersSession()
		if err != nil {
			log.Printf("⚠️ Impossible d'initialiser les prepared statements: %v", err)
			return
		}

		// Requête pour récupérer user_id par email
		stmtGetUserByEmail = session.Query("SELECT user_id FROM users_by_email WHERE email = ?")

		// Requête pour récupérer un utilisateur par ID
		stmtGetUserByID = session.Query(`SELECT email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin 
			FROM users WHERE user_id = ?`)

		// Requête pour insérer un utilisateur
		stmtInsertUser = session.Query(`INSERT INTO users (user_id, email, password, name, role, provider, provider_id, company_id, company_name, is_company_admin, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)

		// Requête pour insérer dans users_by_email
		stmtInsertUserByEmail = session.Query("INSERT INTO users_by_email (email, user_id) VALUES (?, ?)")

		// Requête pour mettre à jour un utilisateur
		stmtUpdateUser = session.Query("UPDATE users SET name = ?, is_company_admin = ?, company_name = ?, role = ?, company_id = ?, updated_at = ? WHERE user_id = ?")

		log.Println("✅ Prepared statements initialisés")
	})
}

// GetPreparedStatements retourne les prepared statements
func GetPreparedGetUserByEmail() *gocql.Query {
	return stmtGetUserByEmail
}

func GetPreparedGetUserByID() *gocql.Query {
	return stmtGetUserByID
}

func GetPreparedInsertUser() *gocql.Query {
	return stmtInsertUser
}

func GetPreparedInsertUserByEmail() *gocql.Query {
	return stmtInsertUserByEmail
}

func GetPreparedUpdateUser() *gocql.Query {
	return stmtUpdateUser
}
