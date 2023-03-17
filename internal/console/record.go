package console

type Record struct {
	NewProgress   *NewProgressRecord   `json:"newProgress"`
	ProgressDone  *ProgressDoneRecord  `json:"progressDone"`
	ProgressError *ProgressErrorRecord `json:"progressError"`
	NewStatus     *NewStatusRecord     `json:"newStatus"`
	StatusUpdate  *StatusUpdateRecord  `json:"statusUpdate"`
	StatusDone    *StatusDoneRecord    `json:"statusDone"`
	StatusError   *StatusErrorRecord   `json:"statusError"`
}

type NewProgressRecord struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}

type ProgressDoneRecord struct {
	Address []int `json:"address"`
}

type ProgressErrorRecord struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}

type NewStatusRecord struct {
	Address []int `json:"address"`
}

type StatusUpdateRecord struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}

type StatusDoneRecord struct {
	Address []int `json:"address"`
}

type StatusErrorRecord struct {
	Address []int  `json:"address"`
	Message string `json:"message"`
}
