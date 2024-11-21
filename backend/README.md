# Backend Setup

## Running the Backend

To run the backend server, execute the following commands:

```sh
cd backend
go run cmd/server/main.go
```

Make sure to set up your database and apply migrations before running the server.

## Applying Migrations

Use a tool like `golang-migrate` to apply SQL migrations.

```sh
migrate -path ./pkg/db/migrations -database sqlite3://app.db up
```
