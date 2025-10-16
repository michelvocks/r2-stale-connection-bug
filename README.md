# Cloudflare R2 Stale Connection Bug

This repository contains a minimal example to demonstrate the stale connection bug in Cloudflare R2. The bug causes connections to remain open for 60 seconds and locking the object for that region.
During that time, any attempts to read the object within the same region will result into a stale connection until the connection is closed.

## Prerequisites

- Golang installed on your machine. You can download it from [here](https://golang.org/dl/).
- Access to a Cloudflare R2 bucket. You can create one by following the instructions [here](https://developers.cloudflare.com/r2/get-started/).
- Your Cloudflare R2 Access Key ID and Secret Access Key. You can find them in the Cloudflare dashboard under R2 settings.
- A custom endpoint URL for your R2 bucket. You can find it in the Cloudflare dashboard under R2 settings.

## Setup

1. Clone this repository to your local machine:

   ```bash
   git clone
    cd r2-stale-connection-bug
    ``` 

2. Create a new Cloudflare R2 bucket if you don't have one already.
3. Create a custom endpoint URL for your R2 bucket if you don't have one already.
4. Set your Cloudflare R2 credentials, Bucket, and endpoint URL in the `main.go` file.
5. Run the Go program:

   ```bash
   go run main.go
   ```

## What the Program Does

The program will perform the following actions:
1. Start a simple HTTP server that serves a static HTML page with a video element.
2. When the user presses the return key, it will upload a file `BunnyShort.mp4` to the specified Cloudflare R2 bucket with a unique name.
3. Immediately attempt to download the same file from the bucket while not gracefully reading the response body and therefore not closing the connection.
4. Once the file is uploaded, the user is able to access the video via the HTTP endpoint (http://localhost:8080). 
5. The video element will try to download the video but will run into a stale connection. Any attempts from the same region to read the object will result in a stale connection until the connection is closed e.g. the program is closed or after 60 seconds.

## What I assume happens in the background (not confirmed by Cloudflare)

I assume that the immediate call to fetch the file after uploading it, while not closing the connection properly, causes Cloudflare to keep the connection open for 60 seconds. 
During this time, the object is locked by the Cloudflare Worker (or proxy?) and any subsequent requests to read the object from the same region will result in a stale connection until the connection is closed.

It is kinda surprising that one fetch can result into a full lock of the object for that region, but that seems to be the case.
