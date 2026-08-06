package campusconnector

import (
	"context"
	"errors"
	"sync"
	"time"

	connectorprotocol "github.com/StuHelper/StuHelper/server/internal/pkg/campusconnectorprotocol"
)

var (
	ErrUnavailable      = errors.New("campus connector is unavailable")
	ErrRejected         = errors.New("school account credentials were rejected")
	ErrAccountLocked    = errors.New("school account is locked or disabled")
	ErrNotStudent       = errors.New("school account is not an eligible student account")
	ErrInvalidResult    = errors.New("campus connector returned an invalid result")
	ErrRequestNotFound  = errors.New("campus connector request not found")
	ErrRequestInFlight  = errors.New("campus connector request is already in flight")
	ErrSnapshotRejected = errors.New("campus connector snapshot rejected")
	ErrAuthentication   = errors.New("campus connector authentication failed")
	ErrReplay           = errors.New("campus connector request replayed")
)

const (
	ResultSuccess       = "success"
	ResultRejected      = "credentials_rejected"
	ResultAccountLocked = "account_locked"
	ResultNotStudent    = "not_student"
	ResultUnavailable   = "upstream_unavailable"
	ResultTLSFailure    = "tls_failure"
	ResultSchemaUnknown = "schema_unknown"
	ResultCancelled     = "cancelled"
)

type InteractiveRequest struct {
	ID             string
	NodeID         string
	SchoolID       int64
	SchoolCode     string
	OperationKey   string
	AdapterID      string
	AdapterVersion string
	StudentID      string
	Password       []byte
	DeadlineAt     time.Time
	ApplicationID  *string
}

type InteractiveResult struct {
	ResultCode     string
	AccountSubject string
	StudentID      string
}

type interactiveJob struct {
	request InteractiveRequest
	result  chan InteractiveResult
	once    sync.Once
}

type Delivery struct {
	Metadata connectorprotocol.InteractiveMetadata
	Password []byte
}

// Broker is intentionally process-local. It cannot persist or replay a school
// password: a process restart, timeout, cancellation, or broken delivery makes
// the request fail and the user must enter the password again.
type Broker struct {
	mu       sync.Mutex
	queues   map[string]chan *interactiveJob
	inflight map[string]*interactiveJob
	closed   bool
	capacity int
}

func NewBroker(capacityPerNode int) *Broker {
	if capacityPerNode <= 0 || capacityPerNode > 256 {
		capacityPerNode = 16
	}
	return &Broker{
		queues:   make(map[string]chan *interactiveJob),
		inflight: make(map[string]*interactiveJob),
		capacity: capacityPerNode,
	}
}

func (b *Broker) Submit(ctx context.Context, request InteractiveRequest) (InteractiveResult, error) {
	if request.ID == "" || request.NodeID == "" || request.SchoolCode == "" || request.OperationKey == "" ||
		request.StudentID == "" || len(request.Password) == 0 || request.DeadlineAt.IsZero() {
		return InteractiveResult{}, ErrUnavailable
	}
	password := append([]byte(nil), request.Password...)
	request.Password = password
	job := &interactiveJob{request: request, result: make(chan InteractiveResult, 1)}
	queue, err := b.queue(request.NodeID)
	if err != nil {
		wipe(password)
		return InteractiveResult{}, err
	}

	deadlineCtx, cancel := context.WithDeadline(ctx, request.DeadlineAt)
	defer cancel()
	select {
	case queue <- job:
	case <-deadlineCtx.Done():
		wipe(password)
		return InteractiveResult{}, ErrUnavailable
	}

	defer func() {
		b.mu.Lock()
		delete(b.inflight, request.ID)
		b.mu.Unlock()
		wipe(password)
	}()
	select {
	case result := <-job.result:
		return result, nil
	case <-deadlineCtx.Done():
		return InteractiveResult{}, ErrUnavailable
	}
}

func (b *Broker) Claim(ctx context.Context, nodeID string) (*Delivery, error) {
	queue, err := b.queue(nodeID)
	if err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case job := <-queue:
			if job == nil {
				return nil, ErrUnavailable
			}
			if !job.request.DeadlineAt.After(time.Now()) {
				job.complete(InteractiveResult{ResultCode: ResultCancelled})
				continue
			}
			b.mu.Lock()
			if b.closed {
				b.mu.Unlock()
				job.complete(InteractiveResult{ResultCode: ResultCancelled})
				return nil, ErrUnavailable
			}
			b.inflight[job.request.ID] = job
			b.mu.Unlock()
			return &Delivery{
				Metadata: connectorprotocol.InteractiveMetadata{
					RequestID: job.request.ID, SchoolID: job.request.SchoolID,
					SchoolCode:   job.request.SchoolCode,
					OperationKey: job.request.OperationKey, AdapterID: job.request.AdapterID,
					AdapterVersion: job.request.AdapterVersion, StudentID: job.request.StudentID,
					DeadlineAt: job.request.DeadlineAt,
				},
				Password: job.request.Password,
			}, nil
		}
	}
}

func (b *Broker) Complete(nodeID string, requestID string, result InteractiveResult) error {
	b.mu.Lock()
	job := b.inflight[requestID]
	b.mu.Unlock()
	if job == nil || job.request.NodeID != nodeID {
		return ErrRequestNotFound
	}
	job.complete(result)
	return nil
}

func (b *Broker) FailDelivery(requestID string) {
	b.mu.Lock()
	job := b.inflight[requestID]
	b.mu.Unlock()
	if job != nil {
		job.complete(InteractiveResult{ResultCode: ResultUnavailable})
	}
}

func (b *Broker) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	jobs := make([]*interactiveJob, 0, len(b.inflight))
	for _, job := range b.inflight {
		jobs = append(jobs, job)
	}
	b.mu.Unlock()
	for _, job := range jobs {
		job.complete(InteractiveResult{ResultCode: ResultCancelled})
	}
}

func (b *Broker) queue(nodeID string) (chan *interactiveJob, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrUnavailable
	}
	queue := b.queues[nodeID]
	if queue == nil {
		queue = make(chan *interactiveJob, b.capacity)
		b.queues[nodeID] = queue
	}
	return queue, nil
}

func (j *interactiveJob) complete(result InteractiveResult) {
	j.once.Do(func() { j.result <- result })
}

func wipe(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
