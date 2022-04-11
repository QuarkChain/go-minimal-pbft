#!/bin/bash

rm -rf ./datadir_2v_n2 &&  \
./main node \
  --datadir ./datadir_2v_n2 \
  --valKey=val2.key \
  --nodeKey=node2.key \
  --genesisTimeMs 1 \
  --validatorSet=0x564D965830b6081506c6de0625F089F751Af134a \
  --validatorSet=0x0a700e9B59d92259C68E50a978c851214916BE52 \
  --validatorSet=0x00D6bF1DbB497e8C638330ffb351B21F950934e0 \
  --validatorSet=0xcad3ADBB6985D33e85e10fdAdE8E5b132Ba4E965 \
  --bootstrap /ip4/127.0.0.1/udp/8999/quic/p2p/12D3KooWEZ94qZgJgUNYiLwXahknkniYgozxw5eocijZJkew6Mj5 \
  --port 8997 \
  --maxPeerCount 2 \
  $@
