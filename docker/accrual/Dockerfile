FROM golang:1.26-alpine

WORKDIR /app

COPY --parents --from=bin accrual_linux_amd64 ./

CMD ["./accrual_linux_amd64"]
