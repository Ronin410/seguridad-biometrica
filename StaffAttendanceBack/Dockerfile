FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /staffattendance .

FROM alpine:3.20
# ca-certificates: llamadas HTTPS a AWS. tzdata: time.LoadLocation("America/Mazatlan")
# necesita la base de datos de zonas horarias, que Alpine no trae por defecto.
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /staffattendance .

EXPOSE 8100
CMD ["./staffattendance"]
