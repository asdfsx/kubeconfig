FROM alpine
RUN apk --no-cache add ca-certificates
WORKDIR /root/
ADD kubeconfig .
CMD ["./kubeconfig", "-incluster=true"]