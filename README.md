# Elastos.ELA

## Summary

This repo is used to hold source code of ELA main chain, and only holds exclusive source code of main chain. This repo depends on Elastos.ELA.Core which holds the common source code used by both ELA main chain and side chain.

## Build

- put it under $GOPATH/src
- run `glide update && glide install` to install depandencies.
- then run `make` to build files.

## Run

- run ./node to run the node program.
