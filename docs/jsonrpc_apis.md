Instructions
===============

this is the document of ela json rpc interfaces.
it follows json-rpc 2.0 protocol but also keeps compatible with 1.0 version. 
That means both named params and positional params are acceptable.

"id" is optional, which will be sent back in the result samely if you add it in a request. 
It is needed when you want to distinguish different requests.

"jsonrpc" is optional. It tells which version this request uses.
In version 2.0 it is required, while in version 1.0 it does not exist.

#### getbestblockhash  
description: return the hash of the most recent block 

parameters: none

result: 

| name   | type | description |
| ------ | ---- | ----------- |
| blockhash | string | the hash of the most recent block | 

arguments sample:
```json
{
  "method":"getbestblockhash"
}
```

result sample:
```json
{
    "id": null,
    "jsonrpc": "2.0",
    "result": "68692d63a8bfc8887553b97f99f09e523d34a2b599bf5b388436b2ddc85ed76e",
    "error": null
}
```
#### getblockhash  
description: return the hash of the specific blockchain height.

paramters: 

| name | type | description |
| ---- | ---- | ----------- |
| height | integer | the height of blockchain |

result:

| name   | type | description |
| ------ | ---- | ----------- |
| blockhash | string | the hash of the block | 

arguments sample:
```json
{
	"method":"getblockhash",
	"params":{"height":1}
}
```
result sample:
```javascript
{
    "id": null,
    "jsonrpc": "2.0",
    "result": "3893390c9fe372eab5b356a02c54d3baa41fc48918bbddfbac78cf48564d9d72",
    "error": null
}
```

#### getblock  
description: return the block information of the specific blockchain hash.

parameters:

| name | type | description |
| ---- | ---- | ----------- |
| blockhash | string | the blockchain hash | 
| verbosity | int | the verbosity of result, can be 0, 1, 2 |

result:(verbosity=0)

raw hash

result sample:
```
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": "00000000c0433b918f500392869aa14cf7a909430fd94502b5c9f05421c9da7519bd6a65219184ea3c0a2973b90b8402c8405b76d7fbe10a268f6de7e4f48e93f5d03df7c31e095bffff7f2000000000d107000001000000010000000000000000000000000000000000000000000000000000000000000000000000002cfabe6d6d3ca6bcc86bada4642fea709731f1653bd34b28ab15b790e102e14e0d7bd138d80100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffff7f00000000000000000000000000000000000000000000000000000000000000000ce39baabcdbb4adce38c5f23314c5f63a536bbcc8f0a47c7054c36ca27f5acd771d095b00000000020000000101000000000403454c4101000846444170b0e427d2010000000000000000000000000000000000000000000000000000000000000000ffffffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a31b2913000000000000000000129e9cf1c5f336fcf3a6c954444ed482c5d916e506b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a341b52c000000000000000000219e9cc4320c3018ced30242b25c03e13a1b2f57c7d107000000"
}
```

result:(verbosity=1)
| name | type | description |
| ---- | ---- | ----------- |
| hash | string | the blockchain hash |
| confirmations | integer | confirmations |
| size | integer | the size of a block in bytes |
| strippedsize | integer | equals to size |
| weight | integer | This block’s weight |
| height | integer | the height of block |
| version | integer | block header's version |
| versionhex | string | block header's version in hex format |
| merkleroot | string | the merkleroot hash of this block |
| tx | array[string] | transaction hashes of this block, in an array|
| time | integer | the Unix timestamp of this block |
| mediantime | integer | equals to time |
| nonce | integer | the nonce of this block |
| bits | integer | bits of this block |
| difficulty | string | difficulty of this block |
| chainwork | string | The estimated number of block header hashes miners had to check from the genesis block to this block, encoded as big-endian hex |
| previousblockhash | string | previous block hash |
| nextblockhash | string | next block hash |
| auxpow | string | Auxpow information in hex format |

arguments sample:
```javascript
{
  "method":"getblock",
  "params":["0000000000000c128adadedd348061952fa5c9bd78320ee25052d2b74a10573f"],
  "id": 123
}
```

result sample
```json
{
    "id": null,
    "error": null,
    "jsonrpc": "2.0",
    "result": {
        "hash": "3893390c9fe372eab5b356a02c54d3baa41fc48918bbddfbac78cf48564d9d72",
        "confirmations": 5156,
        "strippedsize": 498,
        "size": 498,
        "weight": 1992,
        "height": 1,
        "version": 0,
        "versionhex": "00000000",
        "merkleroot": "764691821f937fd566bcf533611a5e5b193008ea1ba1396f67b7b0da22717c02",
        "tx": [
            "764691821f937fd566bcf533611a5e5b193008ea1ba1396f67b7b0da22717c02"
        ],
        "time": 1524737598,
        "mediantime": 1524737598,
        "nonce": 0,
        "bits": 545259519,
        "difficulty": "1",
        "chainwork": "00001423",
        "previousblockhash": "8d7014f2f941caa1972c8033b2f0a860ec8d4938b12bae2c62512852a558f405",
        "nextblockhash": "aa98305779686e66294a9b667e6ac77f5231bb2ce09fe7d9ca641775413ecb5a",
        "auxpow": "01000000010000000000000000000000000000000000000000000000000000000000000000000000002cfabe6d6d3893390c9fe372eab5b356a02c54d3baa41fc48918bbddfbac78cf48564d9d720100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffff7f0000000000000000000000000000000000000000000000000000000000000000a19035123a440356c0a41a85fe26865620fb4c34dcf1b83b46b5e11efcbbdd893ea6e15a0000000000000000"
    }
}
```

result:(verbosity=2)

result format except 'tx' is the same as it is when verbosity=1

result format in 'tx' please see interface 'getrawtransaction'

```
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "hash": "3ca6bcc86bada4642fea709731f1653bd34b28ab15b790e102e14e0d7bd138d8",
        "confirmations": 1,
        "strippedsize": 498,
        "size": 498,
        "weight": 1992,
        "height": 2001,
        "version": 0,
        "versionhex": "00000000",
        "merkleroot": "219184ea3c0a2973b90b8402c8405b76d7fbe10a268f6de7e4f48e93f5d03df7",
        "tx": [
            {
                "txid": "219184ea3c0a2973b90b8402c8405b76d7fbe10a268f6de7e4f48e93f5d03df7",
                "hash": "219184ea3c0a2973b90b8402c8405b76d7fbe10a268f6de7e4f48e93f5d03df7",
                "size": 192,
                "vsize": 192,
                "version": 0,
                "locktime": 2001,
                "vin": [
                    {
                        "txid": "0000000000000000000000000000000000000000000000000000000000000000",
                        "vout": 65535,
                        "sequence": 4294967295
                    }
                ],
                "vout": [
                    {
                        "value": "0.01255707",
                        "n": 0,
                        "address": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",
                        "assetid": "b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3",
                        "outputlock": 0
                    },
                    {
                        "value": "0.02929985",
                        "n": 1,
                        "address": "EXca4DJwqCXa6vbJmpovwatHiP8HRTVS1Z",
                        "assetid": "b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3",
                        "outputlock": 0
                    }
                ],
                "blockhash": "3ca6bcc86bada4642fea709731f1653bd34b28ab15b790e102e14e0d7bd138d8",
                "confirmations": 1,
                "time": 1527324355,
                "blocktime": 1527324355,
                "type": 0,
                "payloadversion": 4,
                "payload": {
                    "CoinbaseData": "ELA"
                },
                "attributes": [
                    {
                        "usage": 0,
                        "data": "46444170b0e427d2"
                    }
                ],
                "programs": []
            }
        ],
        "time": 1527324355,
        "mediantime": 1527324355,
        "nonce": 0,
        "bits": 545259519,
        "difficulty": "1",
        "chainwork": "00000000",
        "previousblockhash": "c0433b918f500392869aa14cf7a909430fd94502b5c9f05421c9da7519bd6a65",
        "nextblockhash": "0000000000000000000000000000000000000000000000000000000000000000",
        "auxpow": "01000000010000000000000000000000000000000000000000000000000000000000000000000000002cfabe6d6d3ca6bcc86bada4642fea709731f1653bd34b28ab15b790e102e14e0d7bd138d80100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffff7f00000000000000000000000000000000000000000000000000000000000000000ce39baabcdbb4adce38c5f23314c5f63a536bbcc8f0a47c7054c36ca27f5acd771d095b0000000002000000"
    }
}
```

#### getblockcount

description: get block count

parameters: none

argument sample:
```javascript
{   
  "method":"getblockcount"
}
```
result sample:
```javascript
{
    "result": 171454
    "id": null,
    "error": null,
    "jsonrpc": "2.0",
}
```

#### getrawtransaction

description: get transaction infomation of given transaction hash.

parameters:

| name | type | description |
| ---- | ---- | ----------- |
| txid | string | transaction hash |
| verbose | bool | verbose of result |

results:

| name | type | description |
| ---- | ---- | ----------- |
| txid | string | transaction id |
| hash | string | transaction id |
| size | integer | transaction size |
| vsize | integer | The virtual transaction size, equals to size || version | integer | The transaction format version number |
| locktime | integer | The transaction’s locktime |
| sequence | integer | The transaction’s sequence number |
| vin | array | input utxo vector of this transaction |
| n   | integer | index of utxo outputs |
| vout | array | output utxo vector of this transaction |
| assetid | string | asset id |
| outputlock | string | outputlock of this transaction |

argument sample:
```javascript
{
	"method":"getrawtransaction",
	"params":["caa0d52ea2b90a08480834b97c271a8b847aadf90057318a33ccc8674b77c796"]
}
```
result sample:(verbose=true)
```javascript
{
    "id": null,
    "error": null,
    "jsonrpc": "2.0",
    "result": {
        "txid": "6864bbf52a3e140d40f1d707bae31d006265efc54dcb58e34037645060ce3e16",
        "hash": "6864bbf52a3e140d40f1d707bae31d006265efc54dcb58e34037645060ce3e16",
        "size": 192,
        "vsize": 192,
        "version": 0,
        "locktime": 1000,
        "vin": [
            {
                "txid": "0000000000000000000000000000000000000000000000000000000000000000",
                "vout": 65535,
                "sequence": 4294967295
            }
        ],
        "vout": [
            {
                "value": "0.01255707",
                "n": 0,
                "address": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",
                "assetid": "b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3",
                "outputlock": 0
            },
            {
                "value": "0.02929985",
                "n": 1,
                "address": "ENTogr92671PKrMmtWo3RLiYXfBTXUe13Z",
                "assetid": "b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3",
                "outputlock": 0
            }
        ],
        "blockhash": "0000000000000000000000000000000000000000000000000000000000000000",
        "confirmations": 4158,
        "time": 1524737766,
        "blocktime": 1524737766,
        "type": 0,
        "payloadversion": 4,
        "payload": {
            "CoinbaseData": "ELA"
        },
        "attributes": [
            {
                "usage": 0,
                "data": "b52165c186769037"
            }
        ],
        "programs": []
    }
}
```
result sample:(verbose=false)

return raw data
```
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": "000403454c4101000846444170b0e427d2010000000000000000000000000000000000000000000000000000000000000000ffffffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a31b2913000000000000000000129e9cf1c5f336fcf3a6c954444ed482c5d916e506b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a341b52c000000000000000000219e9cc4320c3018ced30242b25c03e13a1b2f57c7d107000000"
}
```

#### getrawmempool

description: return hashes of transactions in memory pool.

parameters: none

argument sample:
```javascript
{
  "method":"getrawmempool"
}
```

result sample:

```javascript
{
  "result":["5da460632a154fe75df0d5ec98560e4bc1115374a37a75e984a534f8da3ca941", "5da460632a154fe75df0d5ec98560e4bc1115374a37a75e984a534f8da3ca941"]
  "error": null,
  "id": null,
  "jsonrpc": "2.0",
}
```

#### getreceivedbyaddress
description: get the balance of an address

parameters:

| name | type | description |
| ---- | ---- | ----------- |
| address | string | address |

result: the balance of the address

argument sample:
```json
{
	"method":"getreceivedbyaddress",
	"params":{"address":"8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"}
}
```
result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": "33000000"
}
```
#### listunspent

description: list all utxo of given addresses 

parameters:

| name | type | description |
| ---- | ---- | ----------- |
| addresses | array[string] | addresses |

result:
please see below

argument sample:
```json
{
    "method":"listunspent",
    "params":{"addresses": ["8ZNizBf4KhhPjeJRGpox6rPcHE5Np6tFx3", "EeEkSiRMZqg5rd9a2yPaWnvdPcikFtsrjE"]}
}
```
result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": [
        {
            "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
            "txid": "9132cf82a18d859d200c952aec548d7895e7b654fd1761d5d059b91edbad1768",
            "vout": 0,
            "address": "8ZNizBf4KhhPjeJRGpox6rPcHE5Np6tFx3",
            "amount": "33000000",
            "confirmations": 1102,
            "outputlock": 0
        },
        {
            "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
            "txid": "3edbcc839fd4f16c0b70869f2d477b56a006d31dc7a10d8cb49bd12628d6352e",
            "vout": 0,
            "address": "8ZNizBf4KhhPjeJRGpox6rPcHE5Np6tFx3",
            "amount": "0.01255707",
            "confirmations": 846,
            "outputlock": 0
        }
    ]
```
#### setloglevel

description: set log level

parameters: 

| name | type | description |
| ---- | ---- | ----------- |
| level | integer | the log level |

result:
please see below

argument sample:
```json
{
	"method":"setloglevel",
	"params":{
		"level":0
	}
}
```

result sample:
```json
{
    "id": null,
    "jsonrpc": "2.0",
    "error": null,
    "result": "log level has been set to 0"
}
```
#### getconnectioncount

description: get peer's count of this node

argument sample:
```javascript
{
  "method": "getconnectioncount"
}
```

result sample:
```json
{
    "id": null,
    "error": null,
    "jsonrpc": "2.0",
    "result": 0
}
```
#### getneighbors

description: get peer's info

parameters: none

results:

| name | type | description |
| ---- | ---- | ----------- |
| Time | integer | current time in unix nano format |
| Services | integer | node service type. 4 is spv service and 0 is no spv service |
| IP | array[integer] | ip in 16-byte representation |
| Port | integer | p2p network port |
| ID | integer | node's id | 

argument sample:
```json
{
  "method":"getneighbors"
}
```
result sample:
```javascript
{
    "id": null,
    "error": null,
    "jsonrpc": "2.0",
    "result": [
        {
            "Time": 1524798750979702000,
            "Services": 4,
            "IP": [0,0,0,0,0,0,0,0,0,0,255,255,127,0,0,1],
            "Port": 30338,
            "ID": 8775829619427993046
        }
    ]
}
```

#### getnodestate

description: get node state

parameters: none

results:

| name | type | description |
| ---- | ---- | ----------- |
| Compile | string | node's compile version |
| ID | integer | node's id |
| HexID | string | node's id in hex format |
| Height | integer | current height |
| Version | integer | node's version in config.json |
| Services | integer | node service type. 4 is spv service and 0 is no spv service |
| Relay | bool | whether node will relay transaction or not |
| TxnCnt | integer | transactions transmitted by this node |
| RxTxnCnt | integer | The transaction received by this node |
| Port | integer | p2p network port |
| RPCPort | integer | json-RPC service port |
| RestPort | integer | RESTful service port |
| WSPort | integer | webservice port |
| OpenPort | integer | open service port |
| OpenService | bool | if opens service enabled |
| Neighbors | array[neighbor] | neighbor nodes information |

neighbor:

| name | type | description |
| ---- | ---- | ----------- |
| ID | integer | neighbor's id |
| HexID | string | neighbor's id in hex format |
| Height | integer | neighbor current height |
| Services | integer | neighbor service type. 4 is spv service and 0 is no spv service |
| Relay | bool | whether neighbor will relay transaction or not |
| External | bool | whether neighbor is from external network |
| State | string | neighbor state in string format |
| NetAddress | string | neighbor tcp address |

argument sample:
```json
{
  "method":"getnodestate"
} 
```

```json
{
    "id": null,
    "error": null,
    "jsonrpc": "2.0",
    "result": {
        "Compile": "v0.1.1-50-gcd97",
        "ID": 10544939963783245780,
        "HexID": "0x925727070f1eefd4",
        "Height": 168748,
        "Version": 0,
        "Services": 4,
        "Relay": true,
        "TxnCnt": 0,
        "RxTxnCnt": 0,
        "Port": 20338,
        "PRCPort": 20336,
        "RestPort": 20334,
        "WSPort": 20335,
        "OpenPort": 20866,
        "OpenService": true,
        "Neighbors": [
            {
                "ID": 8978226977158442839,
                "HexID": "0x7c9911ddf65baf57",
                "Height": 168748,
                "Services": 4,
                "Relay": true,
                "External": false,
                "State": "ESTABLISH",
                "NetAddress": "13.229.160.170:20866"
            }
        ]
    }
}
```

#### sendrawtransaction

description: send a raw transaction to node

parameters: 

| name | type | description |
| ---- | ---- | ----------- |
| data | string | raw transaction data in hex |

result:

| name | type | description |
| ---- | ---- | ----------- |
| hash | string | transaction hash |

argument sample:
```json
{
  "method":"sendrawtransaction",
  "params": ["xxxxxx"]
}
```

result sample:
```json
{
  "result":"764691821f937fd566bcf533611a5e5b193008ea1ba1396f67b7b0da22717c02",
  "id": null,
  "jsonrpc": "2.0",
  "error": null
}
```

#### togglemining

description: the switch of mining

parameters:

| name | type | description |
| ---- | ---- | ----------- |
| mining | bool | whether mine or not | 

argument sample:
```json
{
	"method":"togglemining",
	"params":{"mining":false}
}
```

result sample:
```json
{
    "id": null,
    "jsonrpc": "2.0",
    "result": "mining stopped",
    "error": null
}
}
```

#### discretemining

description: generate one or more blocks instantly  
parameters:

| name | type | description |
| ---- | ---- | ----------- |
| count | integer | count of blocks | 

argument sample:
```json
{
	"method":"discretemining",
	"params":{"count":1}
}
```

result sample:
```json
{
    "id": null,
    "jsonrpc": "2.0",
    "result": [
        "741d8131f0eea94c1c72c8bb1f0e9051a0a98441e131585bf5bf01868bf0ef46"
    ],
    "error": null
}
```

#### createauxblock

description: generate an auxiliary block  
parameters:

| name | type | description |
| ---- | ---- | ----------- |
| paytoaddress | string | miner's address | 

named arguments sample:
```json
{
	"method":"createauxblock",
	"params":{"paytoaddress":"Ef4UcaHwvFrFzzsyVf5YH4JBWgYgUqfTAB"}
}
```
positional arguments sample:
```json
{
	"method": "createauxblock",
	"params": ["Ef4UcaHwvFrFzzsyVf5YH4JBWgYgUqfTAB"]
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "chainid": 1224,
        "height": 152789,
        "coinbasevalue": 175799086,
        "bits": "1d36c855",
        "hash": "e28a262b38316fddefb0b5c753f7cc0022afe94e95f881576ad6b8f33f4e49fe",
        "previousblockhash": "f297d03791f4cf2c6ef093b02a77465ea876b040b7772e56b8e140f3bff73871"
    }
}
```

#### submitauxblock

description: submit the solved auxpow of an auxiliary block   
parameters:

| name | type | description |
| ---- | ---- | ----------- |
| blockhash | string | the auxiliary block hash |
| auxpow | string | the solved auxpow of this auxiliary block |  

named arguments sample:
```json
{
	"method":"submitauxblock",
	"params":{
	  "blockhash": "7926398947f332fe534b15c628ff0cd9dc6f7d3ea59c74801dc758ac65428e64",
	  "auxpow": "02000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4b0313ee0904a880495b742f4254432e434f4d2ffabe6d6d9581ba0156314f1e92fd03430c6e4428a32bb3f1b9dc627102498e5cfbf26261020000004204cb9a010f32a00601000000000000ffffffff0200000000000000001976a914c0174e89bd93eacd1d5a1af4ba1802d412afc08688ac0000000000000000266a24aa21a9ede2f61c3f71d1defd3fa999dfa36953755c690689799962b48bebd836974e8cf90000000014acac4ee8fdd8ca7e0b587b35fce8c996c70aefdf24c333038bdba7af531266000000000001ccc205f0e1cb435f50cc2f63edd53186b414fcb22b719da8c59eab066cf30bdb0000000000000020d1061d1e456cae488c063838b64c4911ce256549afadfc6a4736643359141b01551e4d94f9e8b6b03eec92bb6de1e478a0e913e5f733f5884857a7c2b965f53ca880495bffff7f20a880495b"
	}
}
```
positional arguments sample:
```json
{
	"method":"submitauxblock",
	"params":[
	  "7926398947f332fe534b15c628ff0cd9dc6f7d3ea59c74801dc758ac65428e64",
	  "02000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4b0313ee0904a880495b742f4254432e434f4d2ffabe6d6d9581ba0156314f1e92fd03430c6e4428a32bb3f1b9dc627102498e5cfbf26261020000004204cb9a010f32a00601000000000000ffffffff0200000000000000001976a914c0174e89bd93eacd1d5a1af4ba1802d412afc08688ac0000000000000000266a24aa21a9ede2f61c3f71d1defd3fa999dfa36953755c690689799962b48bebd836974e8cf90000000014acac4ee8fdd8ca7e0b587b35fce8c996c70aefdf24c333038bdba7af531266000000000001ccc205f0e1cb435f50cc2f63edd53186b414fcb22b719da8c59eab066cf30bdb0000000000000020d1061d1e456cae488c063838b64c4911ce256549afadfc6a4736643359141b01551e4d94f9e8b6b03eec92bb6de1e478a0e913e5f733f5884857a7c2b965f53ca880495bffff7f20a880495b"
	]
}
```

result sample:
```json
{
  "error": null,
  "id": null,
  "jsonrpc": "2.0",
  "result": true
}
```

#### getinfo

description: return node information.  
warning: this interface is ready to be deprecated. So no api information will be supplied.


#### listproducers

description: show producers infromation
parameters:

| name | type | description |
| ---- | ---- | ----------- |
| start | integer | the start index of producers |
| limit | integer | the limit index of producers |  

named arguments sample:
```json
{
  "method": "listproducers",
  "params":{
  	"start": 0,
   	"limit": 3
   }
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "producers": [
            {
                "publickey": "0237a5fb316caf7587e052125585b135361be533d74b5a094a68c64c47ccd1e1eb",
                "nickname": "elastos1",
                "url": "http://www.google.com",
                "location": 404,
                "active": true,
                "votes": "3.11100000",
                "ip": "127.0.0.1:20618"
            },
            {
                "publickey": "030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9",
                "nickname": "elastos2",
                "url": "http://www.google.com",
                "location": 404,
                "active": true,
                "votes": "3.11100000",
                "ip": "127.0.0.1:20618"
            },
            {
                "address": "0288e79636e41edce04d4fa95d8f62fed73a76164f8631ccc42f5425f960e4a0c7",
                "publickey": "elastos3",
                "url": "http://www.google.com",
                "location": 404,
                "active": true,
                "votes": "0",
                "ip": "127.0.0.1:20618"
            }
        ],
        "total_votes": "6.22200000"
    }
}
```


#### producerstatus

description: show producer status
parameters:

| name | type | description |
| ---- | ---- | ----------- |
| address | string | the address of producer |

named arguments sample:
```json
{
  "method": "producerstatus",
  "params":{
  	"publickey": "0237a5fb316caf7587e052125585b135361be533d74b5a094a68c64c47ccd1e1eb"
  }
}
```

result sample:
```json
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": 1
}
```



#### votestatus

description: show producer vote status
parameters:

| name | type | description |
| ---- | ---- | ----------- |
| address | string | the address of producer |

named arguments sample:
```json
{
  "method": "votestatus",
  "params":{
  	"address": "EZwPHEMQLNBpP2VStF3gRk8EVoMM2i3hda"
  }
}
```

result sample:
```
{
    "error": null,
    "id": null,
    "jsonrpc": "2.0",
    "result": {
        "total": "4.66088900",
        "voting": "0",
        "pending": true
    }
}
```

#### estimatesmartfee

description: estimate transaction fee smartly.

parameters:
| name | type | description |
| ---- | ---- | ----------- |
| confirmations | int | in how many blocks do you want your transaction to be packed |

result:
| name | type | description |
| ---- | ---- | ----------- |
|  -   | int  | fee rate, the unit is sela per KB |

named arguments sample:
```json
{
  "method": "votestatus",
  "params":{
	"confirmations": 5
  }
}
```
result sample:
```json
{
	"error": null,
	"id": null,
	"jsonrpc": "2.0",
	"result": 10000
}
```

