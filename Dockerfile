# Stage 1: dev — Go backend with Air for hot reload
FROM golang:1.25-alpine AS dev

RUN apk add --no-cache git && go install github.com/air-verse/air@latest

WORKDIR /app
CMD ["air"]

# Stage 2: frontend-build — Build the React/Vite frontend
FROM node:22-alpine AS frontend-build

WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 3: backend-build — Compile the Go binary
FROM golang:1.25-alpine AS backend-build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

# Stage 4: prod — Distroless runtime with backend binary
FROM gcr.io/distroless/static-debian12 AS prod

COPY --from=backend-build /app/server /server
COPY --from=frontend-build /app/dist /app/dist

ENTRYPOINT ["/server"]

# Stage 5: nginx-prod — Nginx serving static assets + proxying /api/
FROM nginx:1.25-alpine AS nginx-prod

COPY --from=frontend-build /app/dist /usr/share/nginx/html
COPY nginx/nginx.prod.conf /etc/nginx/conf.d/default.conf
