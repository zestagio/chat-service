package jobsrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/zestagio/chat-service/internal/store"
	"github.com/zestagio/chat-service/internal/store/job"
	"github.com/zestagio/chat-service/internal/types"
)

var ErrNoJobs = errors.New("no jobs found")

type Job struct {
	ID       types.JobID
	Name     string
	Payload  string
	Attempts int
}

func (r *Repo) FindAndReserveJob(ctx context.Context, until time.Time) (Job, error) {
	var j *store.Job
	var err error

	err = r.db.RunInTx(ctx, func(ctx context.Context) error {
		j, err = r.db.Job(ctx).Query().
			ForUpdate(sql.WithLockAction(sql.SkipLocked)).
			Unique(false).
			Where(
				job.AvailableAtLTE(time.Now()),
				job.ReservedUntilLTE(time.Now()),
			).
			Order(store.Asc(job.FieldCreatedAt)).
			First(ctx)

		if store.IsNotFound(err) {
			return ErrNoJobs
		}
		if err != nil {
			return fmt.Errorf("find job: %v", err)
		}

		j, err = j.Update().
			SetReservedUntil(until).
			AddAttempts(1).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("update job for reserve: %v", err)
		}

		return nil
	})
	if err != nil {
		return Job{}, err
	}

	return Job{
		ID:       j.ID,
		Name:     j.Name,
		Payload:  j.Payload,
		Attempts: j.Attempts,
	}, nil
}

func (r *Repo) CreateJob(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error) {
	j, err := r.db.Job(ctx).Create().
		SetName(name).
		SetPayload(payload).
		SetReservedUntil(time.Now()).
		SetAvailableAt(availableAt).
		Save(ctx)
	if err != nil {
		return types.JobIDNil, fmt.Errorf("create new job: %v", err)
	}
	return j.ID, nil
}

func (r *Repo) CreateFailedJob(ctx context.Context, name, payload, reason string) error {
	err := r.db.FailedJob(ctx).Create().
		SetName(name).
		SetPayload(payload).
		SetReason(reason).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("create new failed job: %v", err)
	}
	return nil
}

func (r *Repo) DeleteJob(ctx context.Context, jobID types.JobID) error {
	err := r.db.Job(ctx).DeleteOneID(jobID).Exec(ctx)
	if err != nil {
		if store.IsNotFound(err) {
			return ErrNoJobs
		}
		return fmt.Errorf("delete job: %v", err)
	}
	return nil
}
