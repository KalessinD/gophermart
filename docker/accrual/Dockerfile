# FROM golang:1.26-alpine
FROM debian:bookworm-slim

WORKDIR /app

ENV RUN_ADDRESS $RUN_ADDRESS
ENV DATABASE_URI $DATABASE_URI

COPY --from=bin accrual_linux_amd64 ./

CMD ["./accrual_linux_amd64"]
