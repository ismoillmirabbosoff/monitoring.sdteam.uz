# ---- 1-bosqich: frontend (Vue) build ----
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# ---- 2-bosqich: Go backend build ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# ---- 3-bosqich: yakuniy image ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Tashkent
WORKDIR /app
COPY --from=build /out/server /usr/local/bin/server
COPY --from=web /web/dist /app/web/dist
ENV WEB_DIR=/app/web/dist
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
