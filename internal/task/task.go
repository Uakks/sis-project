package task

// Status describes the lifecycle of a password-analysis task.
type Status string

const (
	Pending    Status = "pending"
	Queued     Status = "queued"
	Processing Status = "processing"
	Completed  Status = "completed"
	Failed     Status = "failed"
)

// Task is one independent password-analysis request.
// Password is retained in memory for this educational checkpoint.
// Never submit real passwords to this demo application.
type Task struct {
	ID       int
	Password string
	Status   Status
	Result   *Result
	Error    string
}

// Result contains the deterministic output of password analysis.
type Result struct {
	Score      int
	Strength   string
	Length     int
	HasUpper   bool
	HasLower   bool
	HasDigit   bool
	HasSpecial bool
}
