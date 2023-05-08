FROM golang:1.20
COPY . /
RUN ls /src
WORKDIR /src
RUN CGO_ENABLED=0 go build -o /bin/app /src/cmd/lm

FROM alpine:3.17.3
COPY --from=0 /bin/app /opt/lm/lm
CMD ["/opt/lm/lm"]
