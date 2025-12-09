package user

import (
	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Autoriser toutes les origines (à ajuster en production)
		return true
	},
}

// CartWebSocket gère la synchronisation temps réel du panier
func CartWebSocket(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(401, gin.H{"error": "Non authentifié"})
		return
	}

	// Upgrade vers WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Erreur upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	ctx := context.Background()

	// S'abonner au canal Redis pour ce user
	pubsub := database.Redis.Subscribe(ctx, "cart:"+userID)
	defer pubsub.Close()

	// Canal pour recevoir les messages Redis
	ch := pubsub.Channel()

	// Envoyer un message de connexion
	conn.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"message": "Synchronisation panier activée",
	})

	// Boucle d'écoute
	for {
		select {
		case msg := <-ch:
			// Recevoir une notification de changement de panier
			if msg.Payload == "updated" || msg.Payload == "cleared" {
				// Récupérer le panier actuel
				key := "cart:" + userID
				data, err := database.Redis.Get(ctx, key).Result()

				var response map[string]interface{}
				if err != nil || data == "" {
					response = map[string]interface{}{
						"type":  "cart_updated",
						"items": []interface{}{},
						"total": 0,
						"count": 0,
					}
				} else {
					var cart []models.CartItem
					json.Unmarshal([]byte(data), &cart)

					total := 0.0
					for _, item := range cart {
						total += item.Price * float64(item.Quantity)
					}

					response = map[string]interface{}{
						"type":  "cart_updated",
						"items": cart,
						"total": total,
						"count": len(cart),
					}
				}

				// Envoyer au client
				if err := conn.WriteJSON(response); err != nil {
					log.Printf("❌ Erreur envoi WebSocket: %v", err)
					return
				}
			}
		case <-time.After(30 * time.Second):
			// Ping pour garder la connexion active
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
