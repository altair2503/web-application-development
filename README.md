# web-application-development

KBTU Web Application Development course repository.

## Homework 1 — Dockerized web application

Simple Go HTTP server in `hw1/`, containerized with a multi-stage Dockerfile.

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/)

### Build the image

From the repository root:

```bash
cd hw1
docker build -t hw1-web-app .
```

### Run the container

Map host port `8080` to the container (the app listens on `8080`):

```bash
docker run --rm -p 8080:8080 hw1-web-app
```

The image sets `DOCKER_ENV=Web App Dev`. Override it at runtime:

```bash
docker run --rm -p 8080:8080 -e DOCKER_ENV="My value" hw1-web-app
```

### Verify

```bash
curl http://localhost:8080/hello
```

Expected response includes the value of `DOCKER_ENV`.
