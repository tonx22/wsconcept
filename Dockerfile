FROM golang:1.19 as builder
WORKDIR /
COPY . ./wsconcept
WORKDIR /wsconcept
RUN CGO_ENABLED=0 GOOS=linux go build -a -o ws-app
#CMD [ "./cloud-app" ]

FROM alpine:3.16
WORKDIR /wsconcept
COPY --from=builder /wsconcept/ws-app .
CMD [ "./ws-app" ]