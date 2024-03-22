FROM        golang:alpine AS build

ADD . /src
WORKDIR /src
RUN go build -o /server

FROM golang:alpine
EXPOSE     9308
COPY --from=build /server /
ENTRYPOINT [ "/server" ]