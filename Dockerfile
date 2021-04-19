FROM golang:1.15

RUN apt-get update && apt-get upgrade -y
RUN git clone --depth 1 --branch testnet_bcfn_libp2p_20201018_02 https://github.com/incognitochain/incognito-highway && \
    cd incognito-highway && \
    go build -o highway
ENV TEST="312333we23222wr1w3"
RUN git clone --depth 1 --branch dev/portal-v4-privacy-v2 https://github.com/incognitochain/incognito-chain
RUN apt install libleveldb-dev -y
WORKDIR /go/incognito-chain
COPY params.txt ./blockchain/params.go
COPY proof.txt ./relaying/btc/proof.go
COPY constant.txt ./common/constants.go
COPY constant2.txt ./blockchain/constants.go
RUN go get -d
COPY init_param.sh init_param.sh
RUN chmod a+x init_param.sh
RUN ./init_param.sh
RUN go build -o incognito
COPY run.sh run.sh
COPY run_node.sh run_node.sh
RUN chmod a+x run.sh run_node.sh

EXPOSE 9334 9338

CMD ["./run.sh"]