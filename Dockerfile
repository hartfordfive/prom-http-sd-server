FROM golang:1.17.0 as builder

LABEL maintainer="Alain Lefebvre <hartfordfive@gmail.com>"
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN make DOCKER=1 build

FROM golang:1.17-alpine 

COPY --from=builder /app/prom-http-sd-server /bin/prom-http-sd-server
ADD conf/conf.yaml /etc/prom-http-sd-server/conf.yaml

EXPOSE 80
ENTRYPOINT [ "/bin/prom-http-sd-server" ]
#CMD [ "-conf-path", "/etc/prom-http-sd-server/conf.yaml" ]
