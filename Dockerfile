# Stage 1: Build Frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend
WORKDIR /app
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 3: Production Image
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=backend /app/server /app/server
COPY --from=backend /app/data /app/data
COPY --from=frontend /app/frontend/dist /app/static
EXPOSE 8080
CMD ["/app/server"]
