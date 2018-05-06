# build stage
FROM golang:alpine AS build-env
ADD . /src
# RUN cd /src && go build -o goapp
RUN cd /src && ./linux_build.sh

# final stage
FROM scratch

WORKDIR /app
COPY --from=build-env /src/main /app/

EXPOSE 3080

ENTRYPOINT ./main

