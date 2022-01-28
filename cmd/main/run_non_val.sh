#!/bin/bash

rm -rf ./datadir_non_val &&  \
./main node \
  --datadir=./datadir_non_val \
  --nodeKey=node_non_val.key \
  --genesisTimeMs 1 \
  --validatorSet=0x564D965830b6081506c6de0625F089F751Af134a \
  --port 8998 \
  --bootstrap /ip4/127.0.0.1/udp/8999/quic/p2p/12D3KooWEZ94qZgJgUNYiLwXahknkniYgozxw5eocijZJkew6Mj5 \
  $@

