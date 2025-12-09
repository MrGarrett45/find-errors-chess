package models

// JobStatus summarizes a batch processing job.
type JobStatus struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	CompletedBatches int    `json:"completed_batches"`
	TotalBatches     int    `json:"total_batches"`
}
