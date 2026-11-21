# Password Strength Analyzer — SIS Checkpoint 2

Go application for **Golang Application Development, SIS Checkpoint 2: Concurrent Processing**. This extends the Checkpoint 1 project: tasks still contain passwords, the analyzer still calculates their character statistics and educational strength rating, and tasks are still held in memory. Checkpoint 2 adds a bounded queue, a reusable worker pool, thread-safe task storage, and controlled shutdown.

> **Safety:** This is a course demonstration, not a production password checker. It retains submitted password strings in memory until the service exits, so **use invented sample passwords only**. Do not log or send real credentials. The 0–5 scoring system is a simple teaching exercise, not a real security estimate.

## Requirements

- Go **1.27**, as specified by the course
- No third-party dependencies
- No database, HTTP server, or external queue

The module path in `go.mod` is `password-strength-analyzer`. All imports use that module path. If you change it to your GitHub path, update the imports too.

## Project structure

```text
password-strength-analyzer/
├── go.mod
├── main.go
├── README.md
├── .gitignore
└── internal/
    ├── task/
    │   ├── task.go
    │   └── task_test.go
    ├── analyzer/
    │   ├── analyzer.go
    │   └── analyzer_test.go
    └── service/
        ├── service.go
        ├── service_test.go
        ├── worker.go
        └── worker_test.go
```

## Domain logic (preserved from Checkpoint 1)

Each `Task` has an integer ID, password input, status, optional `Result`, and optional error message. IDs increase within one in-memory `Service` instance. The analyzer awards one point for each rule:

1. Password length is at least 8 Unicode characters.
2. Has at least one uppercase letter.
3. Has at least one lowercase letter.
4. Has at least one digit.
5. Has at least one punctuation or symbol character.

Scores `0–2` are **Weak**, `3–4` are **Medium**, and `5` is **Strong**. Empty/whitespace-only inputs are rejected with errors.

## Concurrent architecture

```text
main goroutine
  CreateTask(password) ──> map[int]*Task (protected by sync.RWMutex)
  SubmitTask(id) ────────> buffered chan int (capacity 10)
                                    │
                               ┌────┴────┐
                               ▼         ▼
                            Worker 1  Worker 2
                               │         │
                          Analyze()  Analyze()    (overlap is allowed)
                               │         │
                               └────┬────┘
                                    ▼
                             task results and states
                             (protected by sync.RWMutex)
```

`main.go` starts **2 workers** with a **queue capacity of 10**. Worker count and capacity are configurable via `s.Start(workers, queueCapacity)`; the number of workers must be at least 2 and capacity must be positive.

- Workers are **fixed and reusable**, not one goroutine per task.
- The **buffered channel** stores task IDs awaiting processing.
- `SubmitTask` is **non-blocking**: if the channel is full, it returns `ErrQueueFull`. The rejected task **stays Pending**; the queue does not grow and no additional goroutine is created.
- The service holds `s.mu` across the submission state check, channel send, and status update. This prevents duplicate submission and races with shutdown closing the channel.
- A worker transitions `Queued → Processing` under the mutex, releases it while calling `Analyze`, then locks again to write `Result`, `Error`, and final status.
- `GetTask` takes `RLock` and returns an independent copy of the task and result, avoiding external mutation or races.
- Different workers may complete tasks out of order. Tests never assume a fixed completion order.

## Task lifecycle

```text
CreateTask                  SubmitTask                 Worker
   │                            │                        │
   ▼                            ▼                        ▼
Pending ────────────────> Queued ────────────────> Processing
                                                        │
                                           ┌────────────┴───────────┐
                                           ▼                        ▼
                                        Completed                 Failed
                                        (has result)              (has error)
```

Only **Pending** tasks can be submitted. Attempts to submit a `Queued`, `Processing`, `Completed`, or `Failed` task return `ErrInvalidState`. Once a task is accepted, the worker processes it only once. A failed analysis is recorded with `Failed` status; automatic retries are not implemented.

For compatibility with Checkpoint 1, `ProcessTask(id)` still runs synchronously **before** workers are started; after `Start`, it returns `ErrAsyncOnly`. The concurrent Checkpoint 2 path is `SubmitTask(id)`.

## Startup and shutdown

`Start(2, 10)` starts two worker goroutines (once per service). `Stop()`:

1. Acquires the same mutex used by `SubmitTask`.
2. Marks the service stopped, rejecting new task creation, submissions, and restarts.
3. Closes the queue **once** (only the service closes it).
4. Lets workers drain every previously accepted task via `for id := range queue`.
5. Uses `sync.WaitGroup.Wait()` to wait for **all** worker goroutines to exit.

Repeated or concurrent calls to `Stop()` are safe. No task accepted into the queue is silently discarded by normal shutdown. `Stop` waits for running analysis to finish; cancellation/timeouts are intentionally outside Checkpoint 2.

## Running

From the project root:

```bash
go version
go run .
```

`main.go` creates fictional sample passwords, submits them, calls `Stop()` to finish accepted jobs, then prints IDs, statuses, scores and strengths (never raw passwords).

## Testing

```bash
gofmt -l .          # should show no filenames
go vet ./...        # should succeed
go test ./...       # should pass
go test -race ./... # must report no data races
```

For more detailed output:

```bash
go test -v ./...
```

Tests cover the nine Checkpoint 1 cases and the Checkpoint 2 requirements: worker startup, successful submission, multi-task processing and distinct results, actual overlap of two analyses, status transitions, duplicate rejection, bounded/full queue, concurrent map access, shutdown drain and worker termination, rejection after shutdown, and concurrent `SubmitTask`/`Stop` safety. The overlap test uses blocking test-controlled channels, **not `time.Sleep`**; a timeout exists only to fail if progress stops.

## Files and responsibilities

| File | Responsibility |
| --- | --- |
| `internal/task/task.go` | Task, status, result domain types |
| `internal/analyzer/analyzer.go` | Analyzer interface and five-rule scoring algorithm |
| `internal/service/service.go` | Task creation/retrieval, synchronization, synchronous compatibility method |
| `internal/service/worker.go` | Worker pool, bounded queue, submission, safe shutdown |
| `internal/**/**/*_test.go` | Automated tests |
| `main.go` | Runnable Checkpoint 2 demo |

## Key points for the individual defense

- **Why use channels?** A buffered channel is a bounded queue that communicates task IDs to reusable worker goroutines.
- **Why use `RWMutex`?** Concurrent map access and task status/result mutation must be synchronized. `RLock` allows concurrent task snapshots.
- **Why release the lock during analysis?** Workers can perform independent analyses simultaneously; task retrieval is not blocked by a long analysis.
- **How are duplicates prevented?** The task state check and queue submission occur under a single lock; only Pending tasks may enter the queue.
- **What happens when the queue is full?** `ErrQueueFull` is returned immediately; the task remains Pending.
- **How do workers stop?** `Stop` closes the queue once after submissions are disabled; workers drain the channel and exit; `WaitGroup` joins them.
- **What can go wrong without synchronization?** Data races, corrupted map accesses, duplicate processing, or a panic when sending to a closed channel.
