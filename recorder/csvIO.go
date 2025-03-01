package recorder

import (
	"encoding/csv"
	"log"
	"os"
)

func initializeCSV(filename string, header []string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %s", err)
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Failed to close file: %s", err)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write(header); err != nil {
		log.Fatalf("Failed to write header to file: %s", err)
		panic(err)
	}
}

func appendToCSV(filename string, data [][]string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
		panic(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("Failed to close file: %s", err)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.WriteAll(data); err != nil {
		log.Fatalf("Failed to write data to file: %s", err)
		panic(err)
	}
}
