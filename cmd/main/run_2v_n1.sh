#!/bin/bash

rm -rf ./datadir_2v_n1 &&  \
./main node \
  --datadir ./datadir_2v_n1 \
  --valKey=val1.key \
  --nodeKey=node1.key \
  --genesisTimeMs 1 \
  --validatorSet=0x564D965830b6081506c6de0625F089F751Af134a \
  --validatorSet=0x0a700e9B59d92259C68E50a978c851214916BE52 \
  --bootstrap /ip4/127.0.0.1/udp/8999/quic/p2p/12D3KooWEZ94qZgJgUNYiLwXahknkniYgozxw5eocijZJkew6Mj5 \
  --port 8998
  $@
