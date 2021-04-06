#!/bin/bash

# run incognito chain
./run_node.sh beacon-0 &
./run_node.sh beacon-1 &
./run_node.sh beacon-2 &
./run_node.sh beacon-3 &

./run_node.sh shard0-0 &
./run_node.sh shard0-1 &
./run_node.sh shard0-2 &
./run_node.sh shard0-3 &

cd /go/incognito-highway && ./highway -privatekey CAMSeTB3AgEBBCDtIHJcnRKCWVtitn0gkRTHlKvJCvSO12XVtzHna3oSEqAKBggqhkjOPQMBB6FEA0IABKQXV3mHcxNSmL3n4mtWTO4vNP2IuPvizYngBGxf6Fx9cCJhYUYH8r+Plp40dVcz53iXFxbtxIU3Z5oIVVOsYvI= -support_shards all -host "0.0.0.0" --loglevel debug