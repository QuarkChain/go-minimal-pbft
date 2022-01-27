#!/bin/bash

rm -rf ./datadir && ./main node --valKey=val0.key --nodeKey=node0.key --validatorSet=0x564D965830b6081506c6de0625F089F751Af134a --validatorSet=0x0a700e9B59d92259C68E50a978c851214916BE52 $@
