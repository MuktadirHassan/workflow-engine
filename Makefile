run-coordinator:
	LEASE_SECONDS=10 DB_PATH=./dev.db go run ./cmd/coordinator/

run-worker-1:
	WORKER_ID=worker-1 LEASE_SECONDS=10 DB_PATH=./dev.db go run ./cmd/worker
run-worker-2:
	WORKER_ID=worker-2 LEASE_SECONDS=10 DB_PATH=./dev.db go run ./cmd/worker
	
test:
	go test ./...

reset-db:
	rm -f dev.db

seed-db: 
	LEASE_SECONDS=10 DB_PATH=./dev.db go run seed.go