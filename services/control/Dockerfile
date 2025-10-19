FROM golang:1.18-bullseye AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY proto/ proto/
COPY services/control/ services/control/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/decider ./services/control

FROM gcr.io/distroless/base-debian11
COPY --from=build /out/decider /decider
ENV CSN_USE_CONFORMAL=true
ENV CSN_EDGES_UP=1
ENV CSN_EDGE_CAP_COEF=0.15
EXPOSE 7002 9102
ENTRYPOINT ["/decider"]
