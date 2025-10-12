package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

//
// --- INDEXATION DANS ELASTICSEARCH ---
//

// Indexe un produit MongoDB dans Elasticsearch
func IndexProduct(p models.Product) {
	if database.ElasticClient == nil {
		log.Println("⚠️ Elastic non initialisé, impossible d’indexer:", p.Name)
		return
	}

	data, _ := json.Marshal(p)
	req := esapi.IndexRequest{
		Index:      "products",                  // nom de ton index
		DocumentID: p.ID.Hex(),                  // identifiant unique du produit
		Body:       bytes.NewReader(data),
		Refresh:    "true",                      // rend la donnée immédiatement visible
	}

	res, err := req.Do(context.Background(), database.ElasticClient)
	if err != nil {
		log.Println("❌ Erreur envoi Elastic:", err)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("⚠️ Elastic a renvoyé une erreur pour %s: %s", p.Name, res.String())
	} else {
		log.Printf("✅ Produit indexé dans Elasticsearch: %s", p.Name)
	}
}

//
// --- RECHERCHE DANS ELASTICSEARCH ---
//

// Recherche des produits dans Elasticsearch par nom, description ou tags
func SearchProducts(query string) ([]map[string]interface{}, error) {
	if database.ElasticClient == nil {
		return nil, errors.New("client Elasticsearch non initialisé")
	}

	var buf bytes.Buffer
	q := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"name", "description", "tags"},
			},
		},
	}

	if err := json.NewEncoder(&buf).Encode(q); err != nil {
		return nil, fmt.Errorf("erreur encodage requête: %v", err)
	}

	req := esapi.SearchRequest{
		Index: []string{"products"},
		Body:  &buf,
	}
	res, err := req.Do(context.Background(), database.ElasticClient)
	if err != nil {
		return nil, fmt.Errorf("erreur requête Elastic: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		json.NewDecoder(res.Body).Decode(&e)
		log.Printf("❌ Elasticsearch erreur: %+v", e)
		return nil, errors.New("index non trouvé ou vide")
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("erreur décodage JSON: %v", err)
	}

	hitsData, ok := r["hits"].(map[string]interface{})
	if !ok {
		return nil, errors.New("réponse Elastic invalide (pas de hits)")
	}

	hitsArray, ok := hitsData["hits"].([]interface{})
	if !ok {
		return nil, errors.New("aucun résultat trouvé")
	}

	results := make([]map[string]interface{}, 0, len(hitsArray))
	for _, hit := range hitsArray {
		hitMap, _ := hit.(map[string]interface{})
		if source, ok := hitMap["_source"].(map[string]interface{}); ok {
			results = append(results, source)
		}
	}

	return results, nil
}
