#!/bin/bash
NODE_ID=${1}
MAX_PEER_COUNT=${2}
rm -rf ./datadir_4v_n${NODE_ID} &&  \

port=$((8999 + ${NODE_ID}))

./main node \
  --datadir ./datadir_4v_n${NODE_ID} \
  --valKey=val${NODE_ID}.key \
  --nodeKey=node${NODE_ID}.key \
  --genesisTimeMs 1 \
  --validatorSet=0x564D965830b6081506c6de0625F089F751Af134a \
  --validatorSet=0x0a700e9B59d92259C68E50a978c851214916BE52 \
  --validatorSet=0x6cB42599986aF998cF4223D7C839E2dbDeC5f40f \
  --validatorSet=0x817959ec9f31a998Aa7614bCa158f22a9Cf5AE48 \
  --bootstrap /ip4/127.0.0.1/udp/8999/quic/p2p/12D3KooWEZ94qZgJgUNYiLwXahknkniYgozxw5eocijZJkew6Mj5,/ip4/127.0.0.1/udp/9000/quic/p2p/12D3KooWRAPv94qoUn8dAa3NQpZGKjaBcdiaqCETrcuyo2rT2ZvV \
  --port ${port} \
  --maxPeerCount ${MAX_PEER_COUNT}
  $@
