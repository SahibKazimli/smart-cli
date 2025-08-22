package main

import (
	""
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"smart-cli/go-backend/chunk_retriever"
)

func main() {
	// Load env variables
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found.")
	}

	rdb := chunk_retriever.Connect()
	// Prepare query
	query := chunk_retriever.ChunkQuery{
		Query:     "example query text",
		IndexName: "smart-cli_index",
		TopK:      5,
	}

}
