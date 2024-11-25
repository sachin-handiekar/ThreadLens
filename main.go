package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"thread-analyzer/analyzer"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	
	// Define template functions
	router.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	})
	
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatal(err)
	}

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", gin.H{
			"title": "Thread Dump Analyzer",
		})
	})

	router.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			log.Printf("Error getting file: %v", err)
			c.HTML(http.StatusBadRequest, "upload.html", gin.H{
				"title": "Thread Dump Analyzer",
				"error": "No file uploaded",
			})
			return
		}

		log.Printf("Received file: %s", file.Filename)

		// Validate file extension
		ext := filepath.Ext(file.Filename)
		if ext != ".txt" && ext != ".log" {
			log.Printf("Invalid file type: %s", ext)
			c.HTML(http.StatusBadRequest, "upload.html", gin.H{
				"title": "Thread Dump Analyzer",
				"error": "Invalid file type. Only .txt and .log files are allowed",
			})
			return
		}

		// Save the file
		filename := filepath.Join("uploads", file.Filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			log.Printf("Error saving file: %v", err)
			c.HTML(http.StatusInternalServerError, "upload.html", gin.H{
				"title": "Thread Dump Analyzer",
				"error": "Failed to save file",
			})
			return
		}
		log.Printf("File saved successfully: %s", filename)

		// Analyze the thread dump
		analysis, err := analyzer.AnalyzeThreadDump(filename)
		if err != nil {
			log.Printf("Error analyzing thread dump: %v", err)
			c.HTML(http.StatusInternalServerError, "upload.html", gin.H{
				"title": "Thread Dump Analyzer",
				"error": fmt.Sprintf("Failed to analyze thread dump: %v", err),
			})
			return
		}

		if analysis == nil {
			log.Printf("Analysis returned nil")
			c.HTML(http.StatusInternalServerError, "upload.html", gin.H{
				"title": "Thread Dump Analyzer",
				"error": "Analysis failed: no data returned",
			})
			return
		}

		log.Printf("Analysis completed. Rendering results...")
		log.Printf("Total Threads: %d", analysis.TotalThreads)
		log.Printf("Daemon Threads: %d", analysis.DaemonThreadCount)
		log.Printf("Non-Daemon Threads: %d", analysis.NonDaemonThreadCount)
		log.Printf("Thread States: %v", analysis.StateCount)
		log.Printf("Thread Pools: %d", len(analysis.ThreadPools))
		
		// Pass analysis results to template
		c.HTML(http.StatusOK, "result.html", analysis)
	})

	fmt.Println("Server running on http://localhost:8080")
	router.Run(":8080")
}
