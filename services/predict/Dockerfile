FROM golang:1.18-bullseye AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY proto/ proto/
COPY services/predict/ services/predict/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/predictor ./services/predict

FROM gcr.io/distroless/base-debian11
COPY --from=build /out/predictor /predictor
EXPOSE 7001
ENTRYPOINT ["/predictor"]
