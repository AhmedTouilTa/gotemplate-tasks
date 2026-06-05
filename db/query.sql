-- name: GetTask :one
SELECT * FROM tasks
WHERE id = ? LIMIT 1;

-- name: ListTasks :many
SELECT * FROM tasks
ORDER BY name;

-- name: CreateTask :one
INSERT INTO tasks (
  name, description,done
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: UpdateTask :one
UPDATE tasks
set name = ?,
description = ?,
done = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = ?;