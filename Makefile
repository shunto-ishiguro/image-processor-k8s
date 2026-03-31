.PHONY: build run test docker-build docker-run k8s-apply k8s-delete load-test

# Go commands
build:
	go build -o bin/server ./cmd/api

run:
	go run ./cmd/api

test:
	go test -v ./...

# 画像処理APIに負荷テスト
load-test:
	./scripts/load-test-image.sh http://localhost:8080 5 50

# Docker commands
docker-build:
	docker build -t image-api:latest .

docker-run:
	docker run -p 8080:8080 image-api:latest

# Kubernetes commands
k8s-apply:
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/secret.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml

k8s-delete:
	kubectl delete -f k8s/ --ignore-not-found

k8s-logs:
	kubectl logs -f -l app=image-api -n image-api

k8s-status:
	kubectl get all -n image-api
