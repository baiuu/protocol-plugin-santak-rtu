FROM golang:alpine
WORKDIR $GOPATH/src/app
ADD . ./
ENV GO111MODULE=on
ENV GOPROXY="https://goproxy.cn"
ENV SANTAK_PLATFORM_URL=http://127.0.0.1:9999
ENV SANTAK_SERVER_PORT=5300
ENV SANTAK_SERVER_HTTPPORT=4441
ENV SANTAK_PLATFORM_MQTTBROKER=mqtt://127.0.0.1:1883
RUN cd cmd && go build -o ./protocol-plugin-santak-rtu
EXPOSE 4441
EXPOSE 5300
RUN chmod +x cmd/protocol-plugin-santak-rtu
RUN pwd
RUN ls -lrt
WORKDIR $GOPATH/src/app/cmd
ENTRYPOINT [ "./protocol-plugin-santak-rtu" ]
