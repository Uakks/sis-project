Password Strength Analyzer
A synchronous, in-memory Go application for SIS Checkpoint 1 (Core Go). Each task analyzes a sample password, produces a simple strength report, and tracks its processing status.
Educational project: The score measures a few character properties, not actual resistance to password attacks. Use fictional sample passwords only. In this checkpoint, raw task input is retained in memory and must not be used for real passwords.

Features
- Create a task from a password string and assign a unique integer ID.
- Reject empty or whitespace-only input.
- Store multiple tasks and their results independently in memory.
- Retrieve a task by its ID, or return an error if it does not exist.
- Process a pending task and store its analysis result.
- Track task status, with errors for invalid operations such as processing a completed task again.
- Test the main application behavior using Go's standard testing package.
Project structure
password-strength-analyzer/
├── go.mod
├── main.go
├── README.md
└── internal/
    ├── task/
    │   ├── task.go
    │   └── task_test.go
    ├── analyzer/
    │   ├── analyzer.go
    │   └── analyzer_test.go
    └── service/
        ├── service.go
        └── service_test.go
- task: defines task status, input, and analysis result types.
- analyzer: defines the Analyzer interface and implements password analysis.
- service: creates, stores, retrieves, and processes tasks.
- main.go: demonstrates the application with fictional passwords.
Task lifecycle
Pending -> Processing -> Completed
              |
              | analysis returns an error
              v
            Pending
- Pending: the task was created but has not been processed.
- Processing: the service has started analyzing the task.
- Completed: processing succeeded and the result is stored.
- Only a Pending task may be processed.
- Processing a Completed or Processing task again returns an error.
- If the analyzer returns an error, the current implementation moves the task back to Pending for a potential retry.
- Creating a task with empty or whitespace-only input returns an error; fetching an unknown task ID also returns an error.
Checkpoint 1 runs synchronously: the Processing state is assigned during the call and is not usually observable from main.go while processing is in progress.
Password analysis rules
For each nonblank input, the analyzer checks the following conditions and awards 1 point for each condition met:
Condition	Points
At least 8 Unicode characters	1
At least one uppercase letter	1
At least one lowercase letter	1
At least one digit	1
At least one punctuation or symbol character	1


The score ranges from 0 to 5:
Score	Label
0–2	Weak
3–4	Medium
5	Strong


The result also includes length and Boolean flags for uppercase, lowercase, digits, and special characters.
Example (fictional input):
Input: "Hello123!"
Status before processing: pending
Status after processing: completed
Length: 9
Score: 5
Strength: Strong
Requirements
- Go 1.27, as specified by the course.
- No database, HTTP server, external API, or third-party package is needed for Checkpoint 1.
Run
From the project root (the directory containing go.mod):
go version
go run .
The included main.go demonstration creates and processes several independent sample tasks.
Test and verify
# Format source files if needed
gofmt -w .

# This should print nothing after formatting
gofmt -l .

# Run static checks
go vet ./...

# Run all unit tests
go test ./...

# Show individual test results
go test -v ./...
The tests cover successful and invalid task creation, task retrieval, missing IDs, processing and result correctness, status transitions, processing a completed task, and independence of multiple tasks. The analyzer tests use table-driven cases.
Core Go concepts used
Concept	Where it is demonstrated
Structs	task.Task, task.Result, service.Service, analyzer.PasswordAnalyzer
Methods	Analyze, CreateTask, GetTask, ProcessTask
Interface	analyzer.Analyzer describes the Analyze behavior; PasswordAnalyzer and a test fake implement it
Map	map[int]*task.Task stores tasks by unique ID
Slice	[]string in main.go; test case slices in table-driven tests
Value semantics	PasswordAnalyzer.Analyze has a value receiver; copyTask duplicates task values and results
Pointer semantics	*Service method receivers; pointers to stored tasks and results
Error handling	(..., error) returns, if err != nil, and wrapped processing errors
Code organization	Separate internal/task, internal/analyzer, and internal/service packages


Scope and limitations
- All data is lost when the process exits.
- Processing is synchronous and not concurrency-safe yet.
- The score is a simple teaching example, not a professional password-security estimate.
- Never enter real credentials in this example because raw input is currently retained in application memory.
- Concurrency and an HTTP API are intentionally left for later checkpoints.