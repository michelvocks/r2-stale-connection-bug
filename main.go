package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	currentVideoURL  = "/video.mp4"
	currentVideoFile = ""
	videoURLMutex    sync.RWMutex
)

func main() {
	// Add your Cloudflare R2 credentials here
	endpoint := "<account-id>.r2.cloudflarestorage.com"
	accessKeyID := "<access-key>"
	secretAccessKey := "<secret-key>"
	bucketName := "my-bucket"
	publicURL := "https://your-public-url-for-the-bucket/"

	// Set this to true to fix the issue
	closeBody := false

	// Initialize minio client object with custom transport
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
		Transport: &http.Transport{
			IdleConnTimeout:     0 * time.Second,
			DisableKeepAlives:   true,
			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 0,
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Start HTTP server in the background
	server := &http.Server{Addr: ":8080"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
			return
		}
		http.NotFound(w, r)
	})

	http.HandleFunc("/video.mp4", func(w http.ResponseWriter, r *http.Request) {
		videoURLMutex.RLock()
		videoFile := currentVideoFile
		videoURLMutex.RUnlock()

		if videoFile == "" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, videoFile)
	})

	http.HandleFunc("/api/video-url", func(w http.ResponseWriter, r *http.Request) {
		videoURLMutex.RLock()
		videoURL := currentVideoURL
		videoURLMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"url": videoURL})
	})

	// Start server in background
	go func() {
		fmt.Println("Starting HTTP server on http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Server is running. Open http://localhost:8080 in your browser")
	fmt.Println("\nPress Enter to upload video to R2, or Ctrl+C to exit...")

	// Setup signal handler for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Setup channel for Enter key press
	enterChan := make(chan bool)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			_, _ = reader.ReadString('\n')
			enterChan <- true
		}
	}()

	// Main loop - wait for Enter or Ctrl+C
	for {
		select {
		case <-sigChan:
			fmt.Println("\nExiting...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
			os.Exit(0)
		case <-enterChan:
			selectedVideo := "BunnyShort.mp4"

			// Check if the file exists
			if _, err := os.Stat(selectedVideo); os.IsNotExist(err) {
				log.Printf("File %s not found in the current directory\n", selectedVideo)
				fmt.Println("Press Enter to try again, or Ctrl+C to exit...")
				continue
			}

			fmt.Printf("\nUploading video: %s\n", selectedVideo)
			fmt.Println("Uploading video to R2...")

			// Generate unique name
			uniqueID := uuid.New().String()
			uniqueFileName := fmt.Sprintf("BunnyShort-%s.mp4", uniqueID)

			fmt.Printf("Generated unique name: %s\n", uniqueFileName)

			// Upload the video file
			contentType := "video/mp4"

			fmt.Printf("Uploading %s to bucket %s...\n", selectedVideo, bucketName)

			info, err := minioClient.FPutObject(context.Background(), bucketName, uniqueFileName, selectedVideo, minio.PutObjectOptions{
				ContentType: contentType,
			})
			if err != nil {
				log.Printf("Failed to upload video: %v\n", err)
				fmt.Println("Press Enter to try again, or Ctrl+C to exit...")
				continue
			}

			fmt.Printf("Successfully uploaded %s (size: %d bytes)\n", info.Key, info.Size)

			// Imitate discord proxy requests
			time.Sleep(1 * time.Second)
			fmt.Println("Sending GET request to fetch the video...")
			err = sendFetchRequest(publicURL+uniqueFileName, closeBody)
			if err != nil {
				log.Printf("Failed to send fetch request: %v\n", err)
				fmt.Println("Press Enter to try again, or Ctrl+C to exit...")
				continue
			}
			fmt.Println("Sending fetch request successful.")

			// Update the video URL and current video file
			newURL := publicURL + uniqueFileName
			videoURLMutex.Lock()
			currentVideoURL = newURL
			currentVideoFile = selectedVideo
			videoURLMutex.Unlock()

			fmt.Printf("Video is now available at: %s\n", newURL)
			fmt.Println("\nPress Enter to upload BunnyShort.mp4 again, or Ctrl+C to exit...")
		}
	}
}

func sendFetchRequest(videoURL string, closeBody bool) error {
	client := &http.Client{}

	parsedUrl, err := url.Parse(videoURL)
	if err != nil {
		return err
	}
	req := http.Request{URL: parsedUrl, Method: "GET", Header: http.Header{}}

	// First request
	first := req.Clone(context.Background())
	resp, err := client.Do(first)
	if err != nil {
		return err
	}
	if closeBody {
		resp.Body.Close()
	}
	return nil
}
