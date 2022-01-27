#!/bin/bash

rm -rf ./datadir &&  \
./main node \
  --valKey=val0.key \
  --nodeKey=node0.key \
  --validatorSet=0x564D965830b6081506c6de0625F089F751Af134a \
  --bootstrap /ip4/127.0.0.1/udp/8999/quic/p2p/12D3KooWEZ94qZgJgUNYiLwXahknkniYgozxw5eocijZJkew6Mj5
