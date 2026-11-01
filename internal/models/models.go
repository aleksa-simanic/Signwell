package models

import "time"

type Substrate string

const (
	SubstrateWood  Substrate = "wood"
	SubstrateMetal Substrate = "metal"
	SubstrateGlass Substrate = "glass"
	SubstrateVinyl Substrate = "vinyl"
)

type Medium string

const (
	MediumEnamel   Medium = "enamel"
	MediumGoldleaf Medium = "goldleaf"
	MediumVinyl    Medium = "vinyl"
)

type JobStatus string

const (
	StatusQuoted    JobStatus = "quoted"
	StatusSketched  JobStatus = "sketched"
	StatusLettered  JobStatus = "lettered"
	StatusFinishing JobStatus = "finishing"
	StatusReady     JobStatus = "ready"
	StatusDelivered JobStatus = "delivered"
)

var StatusFlow = map[JobStatus]JobStatus{
	StatusQuoted:    StatusSketched,
	StatusSketched:  StatusLettered,
	StatusLettered:  StatusFinishing,
	StatusFinishing: StatusReady,
	StatusReady:     StatusDelivered,
}

type Note struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Status    JobStatus `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Job struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Client    string    `json:"client"`
	Title     string    `json:"title"`
	Substrate Substrate `json:"substrate"`
	Medium    Medium    `json:"medium"`
	DueOn     string    `json:"dueOn"`
	Status    JobStatus `json:"status"`
	Notes     []Note    `json:"notes"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateJobRequest struct {
	Code      string    `json:"code"`
	Client    string    `json:"client"`
	Title     string    `json:"title"`
	Substrate Substrate `json:"substrate"`
	Medium    Medium    `json:"medium"`
	DueOn     string    `json:"dueOn"`
	Status    JobStatus `json:"status"`
	Tags      []string  `json:"tags"`
}

type UpdateJobRequest struct {
	Code      *string    `json:"code"`
	Client    *string    `json:"client"`
	Title     *string    `json:"title"`
	Substrate *Substrate `json:"substrate"`
	Medium    *Medium    `json:"medium"`
	DueOn     *string    `json:"dueOn"`
	Status    *JobStatus `json:"status"`
	Tags      *[]string  `json:"tags"`
}

type AddNoteRequest struct {
	Message string    `json:"message"`
	Status  JobStatus `json:"status"`
}

type JobListResponse struct {
	Items  []Job `json:"items"`
	Total  int   `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}
