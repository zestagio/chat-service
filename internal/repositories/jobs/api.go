package jobsrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	const query = `
	with cte as (
		select "id" from "jobs"
		where "available_at" <= now()
			and "reserved_until" <= now()
		limit 1 for update skip locked
	)
	update "jobs" as "j"
	set "attempts" = "attempts" + 1, "reserved_until" = $1
	from cte
	where "cte"."id" = "j"."id" returning
		"j".id,
		"j".name,
		"j".payload,
		"j".attempts;`

	rows, err := r.db.Job(ctx).QueryContext(ctx, query, until)
	if err != nil {
		return Job{}, fmt.Errorf("query context: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return Job{}, fmt.Errorf("rows err: %v", err)
		}
		return Job{}, ErrNoJobs
	}

	var j Job
	if err := rows.Scan(&j.ID, &j.Name, &j.Payload, &j.Attempts); err != nil {
		return Job{}, fmt.Errorf("scan job: %v", err)
	}
	return j, nil
}

func (r *Repo) CreateJob(ctx context.Context, name, payload string, availableAt time.Time) (types.JobID, error) {
	j, err := r.db.Job(ctx).Create().
		SetName(name).
		SetPayload(payload).
		SetAvailableAt(availableAt).
		Save(ctx)
	if err != nil {
		return types.JobIDNil, fmt.Errorf("create job: %v", err)
	}

	return j.ID, nil
}

func (r *Repo) CreateFailedJob(ctx context.Context, name, payload, reason string) error {
	return r.db.FailedJob(ctx).Create().
		SetName(name).
		SetPayload(payload).
		SetReason(reason).
		Exec(ctx)
}

func (r *Repo) DeleteJob(ctx context.Context, jobID types.JobID) error {
	return r.db.Job(ctx).DeleteOneID(jobID).Exec(ctx)
}
