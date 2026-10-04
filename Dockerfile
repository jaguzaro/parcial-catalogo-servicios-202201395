# Etapa 1: interfaz. La revision de tipos corre dentro de "npm run build".
FROM node:24.21.0-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Etapa 2: compilacion de Go. La usa tambien el servicio de pruebas.
FROM golang:1.26.8-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/catalogo ./cmd/catalogo

# Etapa 3: imagen final, sin shell ni curl, con usuario sin privilegios.
FROM gcr.io/distroless/static-debian12:nonroot AS final
WORKDIR /app
COPY --from=build /out/catalogo /app/catalogo
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/catalogo"]
CMD ["servir"]
