package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"cedra_back_end/internal/database"
	"cedra_back_end/internal/models"
)

func IndexProduct(p models.Product) {
	body, _ := json.Marshal(p)
	res, err := database.ElasticClient.Index(
		"products",
		bytes.NewReader(body),
		database.ElasticClient.Index.WithDocumentID(p.ID.Hex()),
	)
	if err != nil {
		fmt.Println("❌ Erreur indexation Elasticsearch:", err)
		return
	}
	defer res.Body.Close()
}

func SearchProducts(query string) ([]models.Product, error) {
	var buf bytes.Buffer
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"name", "description"},
			},
		},
	}
	json.NewEncoder(&buf).Encode(searchQuery)

	res, err := database.ElasticClient.Search(
		database.ElasticClient.Search.WithContext(context.Background()),
		database.ElasticClient.Search.WithIndex("products"),
		database.ElasticClient.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	var results []models.Product
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		source := hit.(map[string]interface{})["_source"]
		b, _ := json.Marshal(source)
		var p models.Product
		json.Unmarshal(b, &p)
		results = append(results, p)
	}
	return results, nil
}
