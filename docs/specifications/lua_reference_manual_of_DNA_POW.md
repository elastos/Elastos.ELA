# package elaapi

elaapi: the lua API of ELA
vesion: 0.1
date: 2017.11.15

## Index
* [1. module](#module)
 + [1.1 elaapi](#elaapi)
   - [1.1.1 hexStrReverse](#elaapi_hexStrReverse)
   - [1.1.2 sendRawTx](#elaapi_sendRawTx)
   - [1.1.3 getRawTx](#elaapi_getRawTx)
   - [1.1.4 getAssetID](#elaapi_getAssetID)
   - [1.1.5 getCoinbaseHashByHeight](#elaapi_getCoinbaseHashByHeight)
   - [1.1.6 getBlockByHeight](#elaapi_getBlockByHeight)
   - [1.1.7 getBlockByHash](#elaapi_getBlockByHash)
   - [1.1.8 getCurrentBlockHash](#elaapi_getCurrentBlockHash)
   - [1.1.9 getCurrentBlockHeight](#elaapi_getCurrentBlockHeight)
   - [1.1.10 getCurrentTimeStamp](#elaapi_getCurrentTimeStamp)
   - [1.1.11 submitBlock](#elaapi_submitBlock)
   - [1.1.12 togglemining](#elaapi_togglemining)
   - [1.1.13 discreteMining](#elaapi_discreteMining)
   - [1.1.14 getUnspendOutput](#elaapi_getUnspendOutput)
   - [1.1.15 getCurrentBits](#elaapi_getCurrentBits)
* [2. types](#types)
 + [2.1 asset](#asset)
   - [2.1.1 new](asset_new)
   - [2.1.2 get](asset_get)
 + [2.2 balancetxinput](#balancetxinput)
   - [2.2.1 new](#balancetxinput_new)
   - [2.2.2 get](#balancetxinput_get)
 + [2.3 blockdata](#blockdata)
   - [2.3.1 new](#blockdata_new)
   - [2.3.2 get](#blockdata_get)
 + [2.4 block](#block)
   - [2.4.1 new](#block_new)
   - [2.4.2 get](#block_get)
   - [2.4.3 getPrevHash](#block_getPrevHash)
   - [2.4.4 getTxRoot](#block_getTxRoot)
   - [2.4.5 getTimeStamp](#block_getTimeStamp)
   - [2.4.6 getHeight](#block_getHeight)
   - [2.4.7 getBits](#block_getBits)
   - [2.4.8 appendtx](#block_appendtx)
   - [2.4.9 updataRoot](#block_updataRoot)
   - [2.4.10 hash](#block_hash)
   - [2.4.11 serialize](#block_serialize)
   - [2.4.12 deserialize](#block_deserialize)
   - [2.4.13 mining](#block_mining)
 + [2.5 client](#client)
   - [2.5.1 new](#client_new)
   - [2.5.2 get](#client_get)
   - [2.5.3 getAddr](#client_getAddr)
   - [2.5.4 getPubkey](client_getPubkey)
 + [2.6 functioncode](#functioncode)
   - [2.6.1 new](#functioncode_new)
   - [2.6.2 get](#functioncode_get)
 + [2.7 bookkeeper](#bookkeeper)
   - [2.7.1 new](#bookkeeper_new)
   - [2.7.2 get](#bookkeeper_get)
 + [2.8 bookkeeping](#bookkeeping)
   - [2.8.1 new](#bookkeeping_new)
   - [2.8.2 get](#bookkeeping_get)
 + [2.9 coinbase](#coinbase)
   - [2.9.1 new](#coinbase_new)
   - [2.9.2 get](#coinbase_get)
 + [2.10 issueasset](#issueasset)
   - [2.10.1 new](#issueasset_new)
   - [2.10.2 get](#issueasset_get)
 + [2.11 transferasset](#transferasset)
   - [2.11.1 new](#transferasset_new)
   - [2.11.2 get](#transferasset_get)
 + [2.12 registerasset](#registerasset)
   - [2.12.1 new](#registerasset_new)
   - [2.12.2 get](#registerasset_get)
 + [2.13 record](#record)
   - [2.13.1 new](#record_new)
   - [2.13.2 get](#record_get)
 + [2.14 datafile](#datafile)
   - [2.14.1 new](#datafile_new)
   - [2.14.2 get](#datafile_get)
 + [2.15 privacypayload](#privacypayload)
   - [2.15.1 new](#privacypayload_new)
   - [2.15.2 get](#privacypayload_get)
 + [2.16 deploycode](#deploycode)
   - [2.16.1 new](#deploycode_new)
   - [2.16.2 get](#deploycode_get)
 + [2.17 transaction](#transaction)
   - [2.17.1 new](#transaction_new)
   - [2.17.2 get](#transaction_get)
   - [2.17.3 appendtxin](#transaction_appendtxin)
   - [2.17.4 appendtxout](#transaction_appendtxout)
   - [2.17.5 appendattr](#transaction_appendattr)
   - [2.17.6 appendbalance](#transaction_appendbalance)
   - [2.17.7 sign](#transaction_sign)
   - [2.17.8 hash](#transaction_hash)
   - [2.17.9 serialize](#transaction_serialize)
   - [2.17.10 deserialize](#transaction_deserialize)
 + [2.18 txattribute](#txattribute)
   - [2.18.1 new](#txattribute_new)
   - [2.18.2 get](#txattribute_new)
 + [2.19 txoutput](#txoutput)
   - [2.19.1 new](#txoutput_new)
   - [2.19.2 get](#txoutput_get)
 + [2.20 utxotxinput](#utxotxinput)
   - [2.20.1 new](#utxotxinput_new)
   - [2.20.2 get](#utxotxinput_get)


## Package Files
assettype.go       blocktype.go       codetype.go        payloadtype.go     txattributetype.go utxotxinputtype.go
balancetxinput.go  clienttype.go      elaapi.go          transactiontype.go txoutputtype.go

## API reference

<h3 id="module">1. module</h3> 
<h4 id="elaapi">1.1 elaapi</h4> 
<h5 id="elaapi_hexStrReverse">1.1.1 hexStrReverse</h5> 

``` go
name: hexStrReverse
usage: reverse the hex string.
params: hexStr [string]
return: hex string reversed [string]
```

<h5 id="elaapi_sendRawTx">1.1.2 sendRawTx</h5> 

``` go
name: sendRawTx
usage: send raw transaction to ELA node.
params: txn [transaction]
return: error code or tx hash. [string]
```

<h5 id="elaapi_getRawTx">1.1.3 getRawTx</h5> 

``` go
name: getRawTx
usage: get raw transaction from ELA node.
params: txn hash [string]
return: transaction, timestamp and confirmination. [transaction, number, number]
```

<h5 id="elaapi_getAssetID">1.1.4 getAssetID</h5> 

``` go
name: getAssetID
usage: get asset ID
params: null
return: asset ID. [string]
```

<h5 id="elaapi_getCoinbaseHashByHeight">1.1.5 getCoinbaseHashByHeight</h5> 

``` go
name: getCoinbaseHashByHeight
usage: get hash of coinbase transaction by height.
params: height [number]
return: coinbase hash. [string]
```

<h5 id="elaapi_getBlockByHeight">1.1.6 getBlockByHeight</h5> 

```  go
name: getBlockByHeight
usage: get block by height.
params: height [number]
return: block, confirmination [block, number]
```

<h5 id="elaapi_getBlockByHash">1.1.7 getBlockByHash</h5> 

```  go
name: getBlockByHash
usage: get block by hash.
params: hash [string]
return: block, confirmination [block, number]
```

<h5 id="elaapi_getCurrentBlockHash">1.1.8 getCurrentBlockHash</h5> 

``` go
name: getCurrentBlockHash
usage: get current block hash.
params: null
return: current block hash [string]
```

<h5 id="elaapi_getCurrentBlockHeight">1.1.9 getCurrentBlockHeight</h5> 

``` go
name: getCurrentBlockHeight
usage: get current block height.
params: null
return: current block height[number]
```

<h5 id="elaapi_getCurrentTimeStamp">1.1.10 getCurrentTimeStamp</h5> 

``` go
name: getCurrentTimeStamp
usage: get timestamp of current block.
params: null
return: timestamp of current block [number]
```

<h5 id="elaapi_submitBlock">1.1.11 submitBlock</h5> 

``` go
name: submitBlock
usage: submit a block to ELA node.
params: a block [block]
return: error code or block hash [string]
```

<h5 id="elaapi_togglemining">1.1.12 togglemining</h5> 

``` go
name: togglemining 
usage: a switch to start or stop the cpu mining.
params: mining or not [bool]
return: null
```

<h5 id="elaapi_discreteMining">1.1.13 discreteMining</h5> 

``` go
name: discreteMining
usage: discrete mining.
params: number of block to mine [number]
return: null
```

<h5 id="elaapi_getUnspendOutput">1.1.14 getUnspendOutput</h5> 

``` go
name: getUnspendOutput
usage: get Unspend output
params: address [string]
        assetID [string]
return: null
```

<h5 id="elaapi_getUnspendOutput">1.1.15 getCurrentBits</h5> 

``` go
name: getCurrentBits
usage: get bits of prevBlock
params: null
return: bits [number]
```

<h3 id="types">2. types</h3> 
<h4 id="asset">2.1 asset</h4> 
<h5 id="asset_new">2.1.1 new</h5> 

``` go
name: new
usage: create a new asset object.
params: name [string]
		desription [string]
		precision [number]
		assetType [number]
		recordType [number]
return: a new asset [asset]
```

<h5 id="asset_get">2.1.2 get</h5> 

``` go
name: get
usage: print asset object.
params: null
return: null
```

<h4 id="balancetxinput">2.2 balancetxinput</h4> 
<h5 id="balancetxinput_new">2.2.1 new</h5> 

``` go
name: new
usage: create a new balancetxinput object.
params: name [string]
		assetID [string]
		value [number]
		programhash [string]
return: a new balancetxinput [balancetxinput]
```

<h5 id="balancetxinput_get">2.2.2 get</h5> 

``` go
name: get
usage: print balancetxinput object.
params: null
return: null
```

<h4 id="blockdata">2.3 blockdata</h4> 
<h5 id="blockdata_new">2.3.1 new</h5> 

``` go
name: new
usage: create a new blockdata object.
params: version [number]
		prevBlockHash [string]
		txRootHash [string]
		timestamp [number]
		bits [number]
		height [number]
		nonce [number]
return: a new blockdata [blockdata]
```

<h5 id="blockdata_get">2.3.2 get</h5> 

``` go
name: get
usage: print blockdata object.
params: null
return: null
```

<h4 id="block">2.4 block</h4> 
<h5 id="block_new">2.4.1 new</h5> 

``` go
name: new
usage: create a new block object.
params: header[blockdata]
return: a new block [block]
```

<h5 id="block_get">2.4.2 get</h5> 

``` go
name: get
usage: print block object.
params: null
return: null
```

<h5 id="block_getPrevHash">2.4.3 getPrevHash</h5> 

``` go
name: getPrevHash
usage: get previous block hash from block
params: null
return: previous block hash [string]
```

<h5 id="block_getTxRoot">2.4.4 getTxRoot</h5> 

``` go
name: getTxRoot
usage: get transactions root from block
params: null
return: transactions root  [string]
```

<h5 id="block_getTimeStamp">2.4.5 getTimeStamp</h5> 

``` go
name: getTimeStamp
usage: get timestamp from block
params: null
return: timestamp [number]
```

<h5 id="block_getHeight">2.4.6 getHeight</h5> 

``` go
name: getHeight
usage: get height from block
params: null
return: height [number]
```

<h5 id="block_getBits">2.4.7 getBits</h5> 

``` go
name: getBits
usage: get bits from block
params: null
return: bits [number]
```

<h5 id="block_appendtx">2.4.8 appendtx</h5> 

``` go
name: appendtx
usage: append transaction to block
params: transaction [transaction]
return: null
```

<h5 id="block_updataRoot">2.4.9 updataRoot</h5> 

``` go
name: updataRoot
usage: rebuild transactions root for block
params: null
return: null
```

<h5 id="block_hash">2.4.10 hash</h5> 

``` go
name: hash
usage: calculate hash of block
params: null
return: block hash [string]
```

<h5 id="block_serialize">2.4.11 serialize</h5> 

``` go
name: serialize
usage: serialize block
params: null
return: length of block, block serialized [number, string]
```

<h5 id="block_deserialize">2.4.12 deserialize</h5> 

``` go
name: deserialize
usage: deserialize block
params: block serialized [string] 
return: null
```

<h5 id="block_mining">2.4.13 mining</h5> 

``` go
name: mining
usage: solve the difficulty of block
params: null
return: null
```

<h4 id="client">2.5 client</h4> 
<h5 id="client_new">2.5.1 new</h5> 

``` go
name: new
usage: create a new client object.
params: name [string]
		password [string]
		create [bool]
return: a new client [client]
```

<h5 id="client_get">2.5.2 get</h5> 

``` go
name: get
usage: print client object.
params: null
return: null
```

<h5 id="client_getAddr">2.5.3 getAddr</h5> 

``` go
name: getAddr
usage: get address from wallet.
params: null
return: address [string]
```

<h5 id="client_getPubkey">2.5.4 getPubkey</h5> 

``` go
name: getPubkey
usage: get public key compressed from wallet.
params: null
return: public key [string]
```

<h4 id="functioncode">2.6 functioncode</h4> 
<h5 id="functioncode_new">2.6.1 new</h5> 

``` go
name: new
usage: create a new functioncode object.
params: code [string]
		parameterType [string]
		returnType [bool]
return: a new functioncode [functioncode]
```

<h5 id="functioncode_get">2.6.2 get</h5> 

``` go
name: get
usage: print functioncode object.
params: null
return: null
```

<h4 id="bookkeeper">2.7 bookkeeper</h4> 
<h5 id="bookkeeper_new">2.7.1 new</h5> 

``` go
name: new
usage: create a new bookkeeper object.
params: publicKey [string]
		action [number]
		cert [string]
		issuer [string]
return: a new bookkeeper [bookkeeper]
```

<h5 id="bookkeeper_get">2.7.2 get</h5> 

``` go
name: get
usage: print client object.
params: null
return: null
```

<h4 id="bookkeeping">2.8 bookkeeping</h4> 
<h5 id="bookkeeping_new">2.8.1 new</h5> 

``` go
name: new
usage: create a new bookkeeping object.
params: nonce [number]
return: a new bookkeeping [bookkeeping]
```

<h5 id="bookkeeping_get">2.8.2 get</h5> 

``` go
name: get
usage: print bookkeeping object.
params: null
return: null
```

<h4 id="coinbase">2.9 coinbase</h4> 
<h5 id="coinbase_new">2.9.1 new</h5> 

``` go
name: new
usage: create a new coinbase object.
params: coinbaseData [string]
return: a new coinbase [coinbase]
```

<h5 id="coinbase_get">2.9.2 get</h5> 

``` go
name: get
usage: print coinbase object.
params: null
return: null
```

<h4 id="issueasset">2.10 issueasset</h4> 
<h5 id="issueasset_new">2.10.1 new</h5> 

``` go
name: new
usage: create a new issueasset object.
params: null
return: a new issueasset [issueasset]
```

<h5 id="issueasset_get">2.10.2 get</h5> 

``` go
name: get
usage: print issueasset object.
params: null
return: null
```

<h4 id="transferasset">2.11 transferasset</h4> 
<h5 id="transferasset_new">2.11.1 new</h5> 

``` go
name: new
usage: create a new transferasset object.
params: null
return: a new transferasset [transferasset]
```

<h5 id="transferasset_get">2.11.2 get</h5> 

``` go
name: get
usage: print transferasset object.
params: null
return: null
```

<h4 id="registerasset">2.12 registerasset</h4> 
<h5 id="registerasset_new">2.12.1 new</h5> 

``` go
name: new
usage: create a new registerasset object.
params: asset [asset]
		amount [number]
		publickey [string]
return: a new registerasset [registerasset]
```
<h5 id="registerasset_get">2.12.2 get</h5> 

``` go
name: get
usage: print registerasset object.
params: null
return: null
```

<h4 id="record">2.13 record</h4> 
<h5 id="record_new">2.13.1 new</h5> 

``` go
name: new
usage: create a new record object.
params: recordType [string]
		recordData [string]
return: a new record [record]
```

<h5 id="record_get">2.13.2 get</h5> 

``` go
name: get
usage: print record object.
params: null
return: null
```

<h4 id="datafile">2.14 datafile</h4> 
<h5 id="datafile_new">2.14.1 new</h5> 

``` go
name: new
usage: create a new datafile object.
params: filePath [string]
		fileName [string]
		node [string]
		publicKey [string]
return: a new datafile [datafile]
```
<h5 id="datafile_get">2.14.2 get</h5> 

``` go
name: get
usage: print datafile object.
params: null
return: null
```

<h4 id="privacypayload">2.15 privacypayload</h4> 
<h5 id="privacypayload_new">2.15.1 new</h5> 

``` go
name: new
usage: create a new privacypayload object.
params: wallet [client]
		toPublicKey [string]
		data [string]
return: a new privacypayload [privacypayload]
```
<h5 id="privacypayload_get">2.15.2 get</h5> 

``` go
name: get
usage: print privacypayload object.
params: null
return: null
```

<h4 id="deploycode">2.16 deploycode</h4> 
<h5 id="deploycode_new">2.16.1 new</h5> 

``` go
name: new
usage: create a new deploycode object.
params: codes [functioncode]
		name [string]
		version [number]
		author [string]
		email [string]
		description [strintg]
return: a new deploycode [deploycode]
```

<h5 id="deploycode_get">2.16.2 get</h5> 

``` go
name: get
usage: print deploycode object.
params: null
return: null
```

<h4 id="transaction">2.17 transaction</h4> 
<h5 id="transaction_new">2.17.1 new</h5> 

``` go
name: new
usage: create a new transaction object.
params: txType [number]
		payloadVersion [number]
		payload [transferasset | coinbase | ...... ]
		lockTime [number]
return: a new transaction [transaction]
```

<h5 id="transaction_get">2.17.2 get</h5> 

``` go
name: get
usage: print transaction object.
params: null
return: null
```
<h5 id="transaction_appendtxin">2.17.3 appendtxin</h5> 

``` go
name: appendtxin
usage: append a txinput to transaction.
params: txin [utxotxinput]
return: null
```

<h5 id="transaction_appendtxout">2.17.4 appendtxout</h5> 

``` go
name: appendtxout
usage: append a txoutput to transaction.
params: txout [txoutput]
return: null
```

<h5 id="transaction_appendattr">2.17.5 appendattr</h5> 

``` go
name: appendattr
usage: append a attribute to transaction.
params: attr [txattribute]
return: null
```

<h5 id="transaction_appendbalance">2.17.6 appendbalance</h5> 

``` go
name: appendbalance
usage: append a balance to transaction.
params: balance [balancetxinput]
return: null
```

<h5 id="transaction_sign">2.17.7 sign</h5> 

``` go
name: sign
usage: sign a transacton
params: wallet [client]
return: null
```

<h5 id="transaction_hash">2.17.8 hash</h5> 

``` go
name: hash
usage: calculate hash of transaction
params: null
return: transaction hash [string]
```

<h5 id="transaction_serialize">2.17.9 serialize</h5> 

``` go
name: serialize
usage: serialize transaction
params: null
return: length of transaction, transaction serialized [number, string]
```

<h5 id="transaction_deserialize">2.17.10 deserialize</h5> 

``` go
name: deserialize
usage: deserialize transaction
params: transaction serialized [string] 
return: null
```

<h4 id="txattribute">2.18 txattribute</h4> 
<h5 id="txattribute_new">2.18.1 new</h5> 

``` go
name: new
usage: create a new txattribute object.
params: usage [number]
		data [string]
		size [number]
return: a new txattribute [txattribute]
```

<h5 id="txattribute_get">2.18.2 get</h5> 

``` go
name: get
usage: print txattribute object.
params: null
return: null
```

<h4 id="txoutput">2.19 txoutput</h4> 
<h5 id="txoutput_new">2.19.1 new</h5> 

``` go
name: new
usage: create a new txoutput object.
params: assetID [string]
		value [number]
		address [string]
return: a new txoutput [txoutput]
```

<h5 id="txoutput_get">2.19.2 get</h5> 

``` go
name: get
usage: print txoutput object.
params: null
return: null
```

<h4 id="utxotxinput">2.20 utxotxinput</h4> 
<h5 id="utxotxinput_new">2.20.1 new</h5> 

``` go
name: new
usage: create a new utxotxinput object.
params: referTxID [string]
		referTxOutputIndex [number]
		sequence [number]
return: a new utxotxinput [utxotxinput]
```

<h5 id="utxotxinput_get">2.20.2 get</h5> 

``` go
name: get
usage: print utxotxinput object.
params: null
return: null
```

