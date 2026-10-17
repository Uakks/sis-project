package task

type Status string

const (
	Pending    Status = "pending"
	Processing Status = "processing"
	Completed  Status = "completed"
)

type Task struct {
	ID       int
	Password string
	Status   Status
	Result   *Result
}

type Result struct {
	Score      int
	Strength   string
	Length     int
	HasUpper   bool
	HasLower   bool
	HasDigit   bool
	HasSpecial bool
}
