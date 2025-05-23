-- name: GetRequestsCount :one
SELECT COUNT(*) FROM requests;

-- name: InsertRequest :exec
INSERT INTO requests (method, path, headers, query, body)
VALUES (?, ?, ?, ?, ?);
