-- name: InsertJob :exec
INSERT INTO jobs (
    id,
    description,
    definition_of_done,
    progress,
    status,
    priority,
    scheduled_at,
    due_at,
    workspace_path,
    assigned_session_id,
    lease_expires_at,
    created_by,
    source_session_id,
    source_platform,
    source_channel_id,
    deleted_at,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetJob :one
SELECT *
FROM jobs
WHERE id = ?;

-- name: ListJobs :many
SELECT *
FROM jobs
WHERE deleted_at IS NULL
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
ORDER BY priority DESC, updated_at ASC;

-- name: ListReadyJobs :many
SELECT *
FROM jobs
WHERE deleted_at IS NULL
  AND status = 'todo'
  AND (scheduled_at IS NULL OR scheduled_at <= ?)
ORDER BY priority DESC, updated_at ASC;

-- name: ListOngoingJobs :many
SELECT *
FROM jobs
WHERE deleted_at IS NULL
  AND status = 'ongoing';

-- name: ListDeletedJobsBefore :many
SELECT id
FROM jobs
WHERE deleted_at IS NOT NULL
  AND deleted_at < ?;

-- name: UpdateJobTodoFields :execrows
UPDATE jobs
SET
    description = ?,
    definition_of_done = ?,
    priority = ?,
    scheduled_at = ?,
    due_at = ?,
    workspace_path = ?,
    updated_at = ?
WHERE id = ?
  AND status = 'todo'
  AND deleted_at IS NULL;

-- name: UpdateJobProgress :execrows
UPDATE jobs
SET
    progress = ?,
    updated_at = ?
WHERE id = ?
  AND status = 'ongoing'
  AND deleted_at IS NULL;

-- name: ClaimJob :execrows
UPDATE jobs
SET
    status = 'ongoing',
    assigned_session_id = ?,
    lease_expires_at = ?,
    updated_at = ?
WHERE id = ?
  AND status = 'todo'
  AND deleted_at IS NULL
  AND assigned_session_id IS NULL;

-- name: ReleaseJob :execrows
UPDATE jobs
SET
    status = 'todo',
    assigned_session_id = NULL,
    lease_expires_at = NULL,
    updated_at = ?
WHERE id = ?
  AND status = 'ongoing'
  AND deleted_at IS NULL;

-- name: CompleteJob :execrows
UPDATE jobs
SET
    status = 'done',
    assigned_session_id = NULL,
    lease_expires_at = NULL,
    updated_at = ?
WHERE id = ?
  AND status = 'ongoing'
  AND deleted_at IS NULL;

-- name: HeartbeatJob :execrows
UPDATE jobs
SET
    lease_expires_at = ?,
    updated_at = ?
WHERE id = ?
  AND status = 'ongoing'
  AND assigned_session_id = ?
  AND deleted_at IS NULL;

-- name: SoftDeleteJob :execrows
UPDATE jobs
SET
    deleted_at = ?,
    assigned_session_id = NULL,
    lease_expires_at = NULL,
    status = CASE WHEN status = 'ongoing' THEN 'todo' ELSE status END,
    updated_at = ?
WHERE id = ?
  AND deleted_at IS NULL;

-- name: HardDeleteJob :exec
DELETE FROM jobs
WHERE id = ?;

-- name: GetJobByAssignedSession :one
SELECT *
FROM jobs
WHERE assigned_session_id = ?
  AND status = 'ongoing'
  AND deleted_at IS NULL
LIMIT 1;

-- name: InsertJobEvent :exec
INSERT INTO job_events (
    id,
    job_id,
    action,
    detail,
    actor_session_id,
    created_at
) VALUES (?, ?, ?, ?, ?, ?);

-- name: ListJobEvents :many
SELECT *
FROM job_events
WHERE job_id = ?
ORDER BY created_at ASC;
