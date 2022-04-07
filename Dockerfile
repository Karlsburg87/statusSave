FROM docker.io/library/golang:1.18-alpine  AS build-env
WORKDIR /go/src/saveStatus

#Let us cache modules retrieval as they do not change often.
#Better use of cache than go get -d -u
COPY go.mod .
COPY go.sum .
RUN go mod download

#Update certificates
RUN apk --update add ca-certificates

#Get source and build binary
COPY . .

#Need git for Go Get to work. Apline does not have this installed by default
RUN apk --no-cache add git

#Path to main function
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /saveStatus/bin

#Production image - scratch is the smallest possible but Alpine is a good second for bash-like access
FROM scratch
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /saveStatus/bin /bin/saveStatus

#Default root user container envars
ARG PORT="8080"
ARG HOST
ARG USERNAME
ARG PASSWORD
ARG DATABASE_NAME
ARG ROUTING_ID
ARG DB_PORT
#or
ARG DATABASE_URL
#gcp
ARG PROJECT_ID

ENV PORT=${PORT}
ENV HOST=${HOST}
ENV USERNAME=${USERNAME}
ENV PASSWORD=${PASSWORD}
ENV DATABASE_NAME=${DATABASE_NAME}
ENV ROUTING_ID=${ROUTING_ID}
ENV DB_PORT=${DB_PORT}
#or
ENV DATABASE_URL=${DATABASE_URL}
#gcp
ENV PROJECT_ID=${PROJECT_ID}

#Expose port for webhook server
EXPOSE ${PORT}

CMD ["/bin/saveStatus"]