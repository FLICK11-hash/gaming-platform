-- name: CreateFraudReport :one
INSERT INTO fraud_reports (reporter_user_id, reported_user_id, tournament_id, match_id, reason, status)
VALUES ($1, $2, $3, $4, $5, 'open')
RETURNING *;

-- name: GetFraudReport :one
SELECT * FROM fraud_reports WHERE fraud_report_id = $1;

-- name: ListFraudReports :many
SELECT * FROM fraud_reports
WHERE (sqlc.narg('status')::varchar IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC;

-- name: UpdateFraudReportStatus :one
UPDATE fraud_reports SET status = $2 WHERE fraud_report_id = $1
RETURNING *;
