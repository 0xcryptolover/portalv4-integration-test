#!/bin/bash
# change TestnetBTCChainID
sed -i s":TestnetBTCChainID        = .*:TestnetBTCChainID        = \"regtest\":" blockchain/constants.go
# change start mtc block in genesic inc block
sed -i s"|blockchain.TestnetBTCChainID:   int32.*|blockchain.TestnetBTCChainID: int32\(190\)\,|" incognito.go
# change BTC chain_id relaying
sed -i s"|return putGenesisBlockIntoChainParams(genesisHash, genesisBlock, .*|return putGenesisBlockIntoChainParams\(genesisHash, genesisBlock, chaincfg.RegressionNetParams\)|" relaying/btc/relayinggenesis.go
# change hardcore block btc in genesis block
# cchange blockhash start sync
sed -i s'|000000000000003b095b39f4048771e77cc8b2e0885228b6df12cf684242cdf1|401c099c64d22cf30aee64e91dfdec8371948eb0fbf2105cb27b89b0606440d1|' relaying/btc/relayinggenesis.go
# change previous block
sed -i s'|0000000000000034b04cb66e042432ef3114e5834abb4cf60706b5f4a1c33ea6|1188a53049c09bae75d4b856a20ead81b2d4ece91efc99e498797d19a4cca7da|' relaying/btc/relayinggenesis.go
# change merkeroot
sed -i s'|c3079548c42fb4f76b12cd3f28be6a7e80b3c854e33bc259c8353327a0cfda31|f3601dd9918bebaf082cd6cddbdf63152188f928932a31cbbb6eed8310a549a1|' relaying/btc/relayinggenesis.go
# change version
sed -i s'|1073733632|805306368|' relaying/btc/relayinggenesis.go
# change Timestamp
sed -i s'|1607582969|1615520523|' relaying/btc/relayinggenesis.go
# change Bits
sed -i s'|424004321|545259519|' relaying/btc/relayinggenesis.go
# change Nonce
sed -i s'|2341547277|0|' relaying/btc/relayinggenesis.go