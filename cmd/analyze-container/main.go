package main

import (
	"context"
	"encoding/json"
	"example/my-go-api/app"
	"example/my-go-api/app/config"
	"example/my-go-api/app/models"
	"log"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func main() {
	// Global-ish init
	baseCtx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app.MustInitDB()

	queueURL := os.Getenv("QUEUE_URL")
	if queueURL == "" {
		log.Fatal("QUEUE_URL environment variable is required")
	}

	// AWS config & SQS client
	awsCfg, err := awsconfig.LoadDefaultConfig(baseCtx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}
	sqsClient := sqs.NewFromConfig(awsCfg)

	log.Printf("Worker started, listening on SQS queue: %s", queueURL)

	for {
		// Long-poll SQS
		recvCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		resp, err := sqsClient.ReceiveMessage(recvCtx, &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: 5,   // up to 10; tune as you like
			WaitTimeSeconds:     20,  // enable long polling
			VisibilityTimeout:   180, // seconds; must be > max batch processing time
		})
		cancel()

		if err != nil {
			log.Printf("ReceiveMessage error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(resp.Messages) == 0 {
			// No work; small sleep to avoid hot loop
			time.Sleep(2 * time.Second)
			continue
		}

		for _, m := range resp.Messages {
			if m.Body == nil {
				log.Printf("received message with empty body, skipping: %#v", m)
				continue
			}

			var job models.JobMessage
			if err := json.Unmarshal([]byte(*m.Body), &job); err != nil {
				log.Printf("failed to unmarshal job message: %v, body=%s", err, *m.Body)
				// Option: send to DLQ or delete to avoid poison pill
				// Here we delete to avoid infinite retries:
				deleteMessage(sqsClient, queueURL, m)
				continue
			}

			log.Printf("Received job: user=%s batch_index=%d num_games=%d job_id=%s",
				job.User, job.BatchIndex, job.NumGames, job.JobID)

			// Per-job timeout (you can tune this)
			jobCtx, jobCancel := context.WithTimeout(baseCtx, 2*time.Minute)
			err := app.ProcessBatch(jobCtx, cfg, job)
			jobCancel()

			if err != nil {
				log.Printf("error processing job job_id=%s user=%s batch_index=%d: %v",
					job.JobID, job.User, job.BatchIndex, err)

				// IMPORTANT: decide retry strategy
				// - If you want SQS to retry: DO NOT delete the message
				//   (it will become visible again after VisibilityTimeout)
				// - If the error is permanent (bad payload): delete it
				//
				// For now we *don't* delete so it can be retried:
				continue
			}

			// Success: delete message from queue
			deleteMessage(sqsClient, queueURL, m)
		}
	}
}

func deleteMessage(sqsClient *sqs.Client, queueURL string, m sqstypes.Message) {
	if m.ReceiptHandle == nil {
		return
	}
	_, err := sqsClient.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: m.ReceiptHandle,
	})
	if err != nil {
		log.Printf("failed to delete SQS message: %v", err)
	}
}
