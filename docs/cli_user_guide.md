# ELA CLI Instruction

[English|[中文](cli_user_guide_CN.md)]

ela-cli is an ELA node command line client for managing user wallets, sending transactions, and getting blockchain information.

```
NAME:
   ela-cli - command line tool for ELA blockchain

USAGE:
   ela-cli [global options] command [command options] [args]

VERSION:
   v0.3.1-129-gd74b

COMMANDS:
     wallet    Wallet operations
     info      Show node information
     mine      Toggle cpu mining or manual mine
     script    Test the blockchain via lua script
     rollback  Rollback blockchain data
     help, h   Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --rpcuser value      username for JSON-RPC connections
   --rpcpassword value  password for JSON-RPC connections
   --rpcport <number>   JSON-RPC server listening port <number>
   --help, -h           show help
   --version, -v        print the version
```

#### RPC Client Parameters

--rpcport

The `rpcport` parameter specifies the port number to which the RPC server is bound. The default value is 20336.

--rpcuser

The `rpcuser` parameter is used to set the username of the BasicAuth. The default value is "".

--rpcpassword

The `rpcpassword` parameter is used to set the password of the BasicAuth. The default value is "".

For example, getting the current height of blockchain.

```
./ela-cli --rpcport 20336 --rpcuser user123 --rpcpassword pass123 info getcurrentheight
```

Result:

```
301
```

You can configure an `ela-cli.sh` script to simplify the commands.

```
#!/bin/bash

./ela-cli --rpcport 20336 --rpcuser user123 --rpcpassword pass123 $*
```

Getting the current height of blockchain.

```
./ela-cli.sh info getcurrentheight
```



## 1. Wallet Management

Wallet management commands can be used to add, view, modify, delete, and import account.
You can use `./ela-cli wallet -h` command to view help information of wallet management command.

```
NAME:
   ela-cli wallet - With ela-cli wallet, you could control your asset.

USAGE:
   ela-cli wallet command [command options] [args]

COMMANDS:

   Account:
     create, c       Create an account
     account, a      Show account address and public key
     balance, b      Check account balance
     add             Add a standard account
     addmultisig     Add a multi-signature account
     delete          Delete an account
     import          Import an account by private key hex string
     export          Export all account private keys in hex string
     depositaddr     Generate deposit address
     dposv2addr      Generate dposv2 address
     didaddr         Generate did address
     crosschainaddr  Generate cross chain address

   Transaction:
     buildtx       Build a transaction
     signtx        Sign a transaction
     signdigest    sign digest
     verifydigest  verify digest
     sendtx        Send a transaction
     showtx        Show info of raw transaction

OPTIONS:
   --help, -h  show help
```

--wallet <file>, -w <file>

The `wallet` parameter specifies the wallet path. The default value is "./keystore.dat".

--password <value>, -p <value>

The `password` parameter is used to specify the account password. You can also enter a password when prompted.

### 1.1 Create Wallet

The create account command is used to create a standard account and store the private key encryption in the keystore file. Each wallet has a default account, which is generally the first account added. The default account cannot be deleted.

Command:

```
./ela-cli wallet create -p 123
```

Result:

```
ADDRESS                            PUBLIC KEY
---------------------------------- ------------------------------------------------------------------
ESVMKLVB1j1KQR8TYP7YbksHpNSHg5NZ8i 032d3d0e8125ac6215c237605486c9fbd4eb764f52f89faef6b7946506139d26f8
---------------------------------- ------------------------------------------------------------------
```

### 1.2 View Public Key

```
./ela-cli wallet account -p 123
```

Result:

```
ADDRESS                            PUBLIC KEY
---------------------------------- ------------------------------------------------------------------
ESVMKLVB1j1KQR8TYP7YbksHpNSHg5NZ8i 032d3d0e8125ac6215c237605486c9fbd4eb764f52f89faef6b7946506139d26f8
---------------------------------- ------------------------------------------------------------------
```

### 1.3 Check Balance

```
./ela-cli wallet balance
```

Result:

```
INDEX                            ADDRESS BALANCE                           (LOCKED)
----- ---------------------------------- ------------------------------------------
    0 EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR 505.08132198                (174.04459514)
----- ---------------------------------- ------------------------------------------
    0 8PT1XBZboe17rq71Xq1CvMEs8HdKmMztcP 10                                     (0)
----- ---------------------------------- ------------------------------------------
```

### 1.4 Add Standard Account

```
./ela-cli wallet add
```

Result:

```
ADDRESS                            PUBLIC KEY
---------------------------------- ------------------------------------------------------------------
ET15giWpFNSYcTKVbj3s18TsR6i8MBnkvk 031c862055158e50dd6e2cf6bb33f869aaac42c5e6def66a8c12efc1fdd16d5597
---------------------------------- ------------------------------------------------------------------
```

### 1.5 Add Multi-Signature Account

Adding multi-signature account requires specifying the public key list `pks`, and a minimum number of signatures `m`.

-- pks <pubkey list>

The `pks` parameter is used to set the list of public keys separated by commas.

-- m <value>, -m <value>

The `m` parameter is used to set the minimum number of signatures.

For example, generate a 2-of-3 multi-signature account:

```
./ela-cli wallet addmultisig -m 3 --pks 0325406f4abc3d41db929f26cf1a419393ed1fe5549ff18f6c579ff0c3cbb714c8,0353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd,03fd77d569f766638677755e8a71c7c085c51b675fbf7459ca1094b29f62f0b27d,0353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd
```

Enter a password when prompted.

Result:

```
8PT1XBZboe17rq71Xq1CvMEs8HdKmMztcP
```

At this point, `8PT1XBZboe17rq71Xq1CvMEs8HdKmMztcP` is added to the specified keystore file.

### 1.6 Delete Account

The default account cannot be deleted.

View account:

```
./ela-cli wallet account
```

Result:

```
ADDRESS                            PUBLIC KEY
---------------------------------- ------------------------------------------------------------------
ET15giWpFNSYcTKVbj3s18TsR6i8MBnkvk 031c862055158e50dd6e2cf6bb33f869aaac42c5e6def66a8c12efc1fdd16d5597
---------------------------------- ------------------------------------------------------------------
EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR 034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16c
---------------------------------- ------------------------------------------------------------------
```

Delete Account:

```
./ela-cli wallet delete ET15giWpFNSYcTKVbj3s18TsR6i8MBnkvk
```

Result:

```
ADDRESS                            PUBLIC KEY
---------------------------------- ------------------------------------------------------------------
EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR 034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16c
---------------------------------- ------------------------------------------------------------------
```

### 1.7 Export Account

```
./ela-cli wallet export
```

Enter a password when prompted.

Result:

```
ADDRESS                            PRIVATE KEY
---------------------------------- ------------------------------------------------------------------
EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR c779d181658b112b584ce21c9ea3c23d2be0689550d790506f14bdebe6b3fe38
---------------------------------- ------------------------------------------------------------------
```

### 1.8 Import Account

```
./ela-cli wallet import b2fe3300e44b27b199d97af43ed3e82df4670db66727732ba8d9442ce680da35 -w keystore1.dat
```

Enter a password when prompted.

Result:

```
ADDRESS                            PUBLIC KEY
---------------------------------- ------------------------------------------------------------------
EQJP3XT7rshteqE1D3u9nBqXL7xQrfzVh1 03f69479d0f6aa11aae5fbe5e5bfca201d717d9fa97a45ca7b89544047a1a30f59
---------------------------------- ------------------------------------------------------------------
```

### 1.9 Generate Deposit Address

Generate a deposit address from a standard address:

```
./ela-cli wallet depositaddr EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR
```

Result:

```
DVgnDnVfPVuPa2y2E4JitaWjWgRGJDuyrD
```

### 1.10 Generate Cross Chain Address

Generate a cross chain address from a side chain genesis block hash:

```
./ela-cli wallet crosschainaddr 56be936978c261b2e649d58dbfaf3f23d4a868274f5522cd2adb4308a955c4a3
```

Result:

```
XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ
```



### 2.1 Build Transaction

Build transaction command can build transaction raw data. Note that before sending to ela node, the transaction should be signed by the private key.

--from
The `from` parameter specifies the transfer-out account address. The default value is the default account of the keystore file.

--to
The `to` parameter specifies the transfer-in account address.

-- tomany
The `tomany` parameter specifies the file which saves multiple outputs.

--amount
The `amount` parameter specifies the transfer amount.

--fee
The `fee` parameter specifies the transfer fee cost.

--outputlock
The `outputlock` parameter specifies the block height when the received asset can be spent.

--txlock
The `txlock` parameter specifies the block height when the transaction can be packaged.

The details of `outputlock` and `txlock` specification in the document [Locking_transaction_recognition](Locking_transaction_recognition.md).

#### 2.1.1 Build standard signature transaction

```
./ela-cli wallet buildtx --to EJbTbWd8a9rdutUfvBxhcrvEeNy21tW1Ee --amount 0.1 --fee 0.01
```

Result:

```
Hex:  090200010013373934373338313938363837333037373435330139eaa4137a047cdaac7112fe04617902e3b17c1685029d259d93c77680c950f30100ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3809698000000000000000000210fcd528848be05f8cffe5d99896c44bdeec7050200b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a36ea2d2090000000000000000210d4109bf00e6d782db40ab183491c03cf4d6a37a000000000001002321034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16cac
File:  to_be_signed.txn
```

#### 2.1.2 Build multi-signature transaction

The from address must exist in the keystore file.

Command:

```
./ela-cli wallet buildtx --from 8PT1XBZboe17rq71Xq1CvMEs8HdKmMztcP --to EQJP3XT7rshteqE1D3u9nBqXL7xQrfzVh1 --amount 0.51 --fee 0.001
```

Result:

```
Hex:
0902000100133334393234313234323933333338333335313701737a31035ebe8dfe3c58c7b9ff7eb13485387cd2010d894f39bf670ccd1f62180000ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3c0320a030000000000000000214e6334d41d86e3c3a32698bdefe974d6960346b300b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3a0108f380000000000000000125bc115b91913c9c6347f0e0e3ba3b75c80b94811000000000001008b53210325406f4abc3d41db929f26cf1a419393ed1fe5549ff18f6c579ff0c3cbb714c8210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd2103fd77d569f766638677755e8a71c7c085c51b675fbf7459ca1094b29f62f0b27d54ae
File:  to_be_signed.txn
```

#### 2.1.3 Build multi-output transaction

To build a multi-output transaction, you need to prepare a file in CSV format that specifies the recipient's address and amount. For example: addresses.csv

```
EY55SertfPSAiLxgYGQDUdxQW6eDZjbNbX,0.001
Eeqn3kNwbnAsu1wnHNoSDbD8t8oq58pubN,0.002
EXWWrRQxG2sH5U8wYD6jHizfGdDUzM4vGt,0.003
```

The first column is the recipient's address and the second column is the amount. (note that the above is sample data, and you need to fill in your own data when sending the real transaction)

Specify the addresses.csv file with the `—tomany` parameter.

```
./ela-cli wallet buildtx --tomany addresses.csv --fee 0.001
```

Result：

```
Multi output address: EY55SertfPSAiLxgYGQDUdxQW6eDZjbNbX , amount: 0.001
Multi output address: Eeqn3kNwbnAsu1wnHNoSDbD8t8oq58pubN , amount: 0.002
Multi output address: EXWWrRQxG2sH5U8wYD6jHizfGdDUzM4vGt , amount: 0.003
Hex:  09020001001235353938383739333333353636383730383501b9932e31681c63ce0425eecb50c2dfc8298bb3e1bcf31b1db648c11f65fd2caf0000ffffffff03b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3a0860100000000000000000021a3a01cc2e25b0178010f5d7707930b7e41359d7e00b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3400d0300000000000000000021ede51096266b26ca8695c5452d4d209760385c3600b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3e09304000000000000000000219d7798bd544db55cd40ed3d5ba55793be4db170a00000000000100232103c5b92b875b9820aba064dd1c93007c8a971fc43d318f7dc7fd6ea1509a424195ac
File:  to_be_signed.txn
```

#### 2.1.4 Build special transaction

##### 2.1.4.1 Build producer transaction

- Build producer register transaction

```shell
$ ./ela-cli wallet buildtx producer register -w temp.dat -p 123 --amount 5000 --fee 0.0001 --nodepublickey 032895050b7de1a9cf43416e6e5310f8e909249dcd9c4166159b04a343f7f141b5 --nickname test --url test.com --location 86 --netaddress 127.0.0.1 --stakeuntil 2000000
```

- Build producer update transaction

```shell
$ ./ela-cli wallet buildtx producer update -w temp.dat -p 123 --fee 0.0001 --nodepublickey 032895050b7de1a9cf43416e6e5310f8e909249dcd9c4166159b04a343f7f141b5 --nickname test --url test.com --location 86 --netaddress 127.0.0.1 --stakeuntil 2000000
```

- Build producer activation transaction

```
NAME:
   ela-cli wallet buildtx producer activate - Build a tx to activate producer which have been inactivated

USAGE:
   ela-cli wallet buildtx producer activate [command options] [arguments...]

OPTIONS:
   --nodepublickey value       the node public key of an arbitrator which have been inactivated
   --wallet <file>, -w <file>  wallet <file> path (default: "keystore.dat")
   --password value, -p value  wallet password
```

The `nodepublickey` parameter is used to specify the node public key of an arbiter.

The account associated with node publickey must exist in the specified keystore file.
If the nodepublickey parameter is not set, the publickey of the main account in the keystore file is used by default.

```
./ela-cli wallet buildtx producer activate --nodepublickey 032895050b7de1a9cf43416e6e5310f8e909249dcd9c4166159b04a343f7f141b5
```

You need to enter a password to sign the payload of the transaction.

Result:

```
Hex:  090d0021032895050b7de1a9cf43416e6e5310f8e909249dcd9c4166159b04a343f7f141b540b78018ce13a67366aab73584c66237a5345420049d141f1ba84876d0c015345cc6e503b2a3cd7c9b20e7677e8e0f8f5bb2bf311c480a9aba1eed217501d3e2ab0000000000000000
File:  ready_to_send.txn
```

- Build producer unregister transaction

```shell
./ela-cli wallet buildtx producer unregister -w temp.dat -p 123 --fee 0.0001
```

- Build producer return deposit transaction

```shell
./ela-cli wallet buildtx producer returndeposit -w temp.dat -p 123 --fee 0.0001 --amount 5000
```

##### 2.1.4.2 Build CRC transaction

- Build crc register transaction

```shell
  $ ./ela-cli wallet buildtx crc register  -w temp.dat -p 123 --amount 5000 --fee 0.0001 --nickname test --url 127.0.0.1 --location 86
```

- Build crc update transaction

```shell
$ ./ela-cli wallet buildtx crc update -w temp.dat -p 123 --nickname test --url 127.0.0.1 --location 86 --fee 0.0001
```

- Build crc unregister transaction

```shell
$ ./ela-cli wallet buildtx crc unregister -w temp.dat -p 123 --fee 0.0001
```

- Build crc returndeposit transaction

```shell
$ ./ela-cli wallet buildtx crc returndeposit -w temp.dat -p 123 --amount 5000 --fee 0.0001
```

##### 2.1.4.3 Build Voting transaction

```
NAME:
   ela-cli wallet buildtx vote - Build a tx to vote for candidates using ELA

USAGE:
   ela-cli wallet buildtx vote [command options] [arguments...]

OPTIONS:
   --for <file>                the <file> path that holds the list of candidates
   --amount <amount>           the transfer <amount> of the transaction
   --from <address>            the sender <address> of the transaction
   --fee <fee>                 the transfer <fee> of the transaction
   --wallet <file>, -w <file>  wallet <file> path (default: "keystore.dat")
   --password value, -p value  wallet password
```

--for

The `for` parameter is used to specify the file which saves multiple candidates' public key.

--amount

The `amount` parameter is used to specify the number of votes.

To build a voting transaction, you need to prepare a file that specifies the candidates' public key.

For example: candidates.csv

```
033b4606d3cec58a01a09da325f5849754909fec030e4cf626e6b4104328599fc7
032895050b7de1a9cf43416e6e5310f8e909249dcd9c4166159b04a343f7f141b5
```

Vote 1 ELA for each of the above candidates.

```
./ela-cli wallet buildtx vote --for candidates.csv --amount 1 --fee 0.1
```

Result:

```
candidate: 033b4606d3cec58a01a09da325f5849754909fec030e4cf626e6b4104328599fc7
candidate: 032895050b7de1a9cf43416e6e5310f8e909249dcd9c4166159b04a343f7f141b5
Hex:  0902000100133931333133393830343439313038373335313101b9932e31681c63ce0425eecb50c2dfc8298bb3e1bcf31b1db648c11f65fd2caf0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300e1f5050000000000000000214cbc08129018f205d99007d8b57be7600c772afe010001000221033b4606d3cec58a01a09da325f5849754909fec030e4cf626e6b4104328599fc721032895050b7de1a9cf43416e6e5310f8e909249dcd9c4166159b04a343f7f141b5000000000100232103c5b92b875b9820aba064dd1c93007c8a971fc43d318f7dc7fd6ea1509a424195ac
File:  to_be_signed.txn
```

##### 2.1.4.4 Build cross chain transaction

```
NAME:
   ela-cli wallet buildtx crosschain - Build a cross chain tx

USAGE:
   ela-cli wallet buildtx crosschain [command options] [arguments...]

OPTIONS:
   --saddress <address>        the locked <address> on main chain represents one side chain
   --amount <amount>           the transfer <amount> of the transaction
   --from <address>            the sender <address> of the transaction
   --to <address>              the recipient <address> of the transaction
   --fee <fee>                 the transfer <fee> of the transaction
   --wallet <file>, -w <file>  wallet <file> path (default: "keystore.dat")
```

--saddress
The `saddress` parameter is used to specify the locked address on main chain which represents one side chain

--to
The `to` parameter is used to specify the target address on side chain

```
./ela-cli wallet buildtx crosschain --saddress XKUh4GLhFJiqAMTF6HyWQrV9pK9HcGUdfJ --to EKn3UGyEoycACJxKu7F8R5U1Pe6NUpni1H --amount 1 --fee 0.1
```

Result:

```
Hex:  0908000122454b6e33554779456f796341434a784b75374638523555315065364e55706e69314800804a5d050000000001001332373635333731343930333630363035373639019c050bc9bee2fde996792b7b7225922a8be739a5e6408d209ad8a2b622c48f900100ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300e1f50500000000000000004b5929cbd09401eb2ce4134cb5ee117a01152c387e00b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3ae03ec030000000000000000214cbc08129018f205d99007d8b57be7600c772afe00000000000100232103c5b92b875b9820aba064dd1c93007c8a971fc43d318f7dc7fd6ea1509a424195ac
File:  to_be_signed.txn
```

##### 2.1.4.5 Build proposal transaction

- Build normal proposal

  - step 1: generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal normal ownerpayload --category category --ownerpubkey 02029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1 --drafthash 8ae42e1933bc328cf0d8308e5d10c31dddf891d36f5922fc5681f0422e42a16b --draftdata 12345678 --budgets "0,1,1000|1,2,2000|2,3,0" --recipient EWRwrmBWpYFwnvAQffcP1vrPCS5sGTgWEB

  payload:  00000863617465676f72792102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb16ba1422e42f08156fc22596fd391f8dd1dc3105d8e30d8f08c32bc33192ee48a041234567803000100e8764817000000010200d0ed902e000000020300000000000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d
  digest:  85588e912018f16e1977059f5d85db8f2b355f0516f20229aec716f9337b0c6d
  ```

  - step 2: owner sign the digest of payload

  ```shell
  $ ./ela-cli wallet signdigest -w owner.dat -p 123 --digest 85588e912018f16e1977059f5d85db8f2b355f0516f20229aec716f9337b0c6d

  Signature:  190caf19d22ba1d131a638e922c3814cd4fcea6e98a5f015811f350e7c3ace2ab81110619805bb192f5ebc9d63a94dcbe5b1c85f1a992e4704455038581f8f1f
  ```

  - step 3: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal normal crcmemberpayload --payload 00000863617465676f72792102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb16ba1422e42f08156fc22596fd391f8dd1dc3105d8e30d8f08c32bc33192ee48a041234567803000100e8764817000000010200d0ed902e000000020300000000000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d --ownersignature 190caf19d22ba1d131a638e922c3814cd4fcea6e98a5f015811f350e7c3ace2ab81110619805bb192f5ebc9d63a94dcbe5b1c85f1a992e4704455038581f8f1f --crcmemberdid idFPSjznfyrFRivJuBWLiYZCTbC4wcnoVi

  payload:  00000863617465676f72792102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb16ba1422e42f08156fc22596fd391f8dd1dc3105d8e30d8f08c32bc33192ee48a041234567803000100e8764817000000010200d0ed902e000000020300000000000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d40190caf19d22ba1d131a638e922c3814cd4fcea6e98a5f015811f350e7c3ace2ab81110619805bb192f5ebc9d63a94dcbe5b1c85f1a992e4704455038581f8f1f677278335e5528c304acda6fb59d612c340cc17022
  digest:  50f3446e60f487d0a54264791bcdd9c7b991e091d46c3405362d4706b4909a2b
  ```

  - step 4: crcouncil member sign the digest of payload

  ```shell
  $ ./ela-cli wallet signdigest -w crcouncilmember.dat -p 123 --digest 50f3446e60f487d0a54264791bcdd9c7b991e091d46c3405362d4706b4909a2b
  Signature:  d7c52471e6df68925433ededb5668c01a584837c6b227c91ed2124023bfc7cffcf9cca138573eb3db04f8c09141b74a15fb462f827505a69ff7b11e91ca28c5f
  ```

  - step 5: create normal proposal transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal normal -w temp.dat --payload 00000863617465676f72792102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb16ba1422e42f08156fc22596fd391f8dd1dc3105d8e30d8f08c32bc33192ee48a041234567803000100e8764817000000010200d0ed902e000000020300000000000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d40190caf19d22ba1d131a638e922c3814cd4fcea6e98a5f015811f350e7c3ace2ab81110619805bb192f5ebc9d63a94dcbe5b1c85f1a992e4704455038581f8f1f677278335e5528c304acda6fb59d612c340cc17022 --fee 0.0001 --crcmembersignature d7c52471e6df68925433ededb5668c01a584837c6b227c91ed2124023bfc7cffcf9cca138573eb3db04f8c09141b74a15fb462f827505a69ff7b11e91ca28c5f

  Hex:  09250100000863617465676f72792102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb16ba1422e42f08156fc22596fd391f8dd1dc3105d8e30d8f08c32bc33192ee48a041234567803000100e8764817000000010200d0ed902e000000020300000000000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d40190caf19d22ba1d131a638e922c3814cd4fcea6e98a5f015811f350e7c3ace2ab81110619805bb192f5ebc9d63a94dcbe5b1c85f1a992e4704455038581f8f1f677278335e5528c304acda6fb59d612c340cc1702240d7c52471e6df68925433ededb5668c01a584837c6b227c91ed2124023bfc7cffcf9cca138573eb3db04f8c09141b74a15fb462f827505a69ff7b11e91ca28c5f0100133736383932333636333031303838303633383501044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```



- Build review proposal

  - step 1: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal review crcmemberpayload --proposalhash f6c7dea2ebfc962b16942f1b5213b19c63d1d8c7f0d49c561654e10c714523c1 --voteresult 0 --opinionhash 962520376792843a44ca368d2db1c267c539a9fbfaf06943250ee43f46b912d4 --opiniondata 12345678 -did idFPSjznfyrFRivJuBWLiYZCTbC4wcnoVi

  payload:  c12345710ce15416569cd4f0c7d8d1639cb113521b2f94162b96fceba2dec7f600d412b9463fe40e254369f0fafba939c567c2b12d8d36ca443a849267372025960412345678677278335e5528c304acda6fb59d612c340cc17022
  digest:  206d27b8ecf234e7a02ecea450eba16208cfecb831fff90022bada28e39fa2f5
  ```

  - step 2:  crcouncil member sign the digest of payload

  ```shell
  $ ./ela-cli wallet signdigest -w crcouncilmember.dat -p 123 --digest 206d27b8ecf234e7a02ecea450eba16208cfecb831fff90022bada28e39fa2f5

  Signature:  514c10f9c12575fc747dfdcec1a21ba8796256b12f444bd27400e2c99a6fa4c94e41b20324a1c549566b8a1e0442eaf9abb90d485ecd7f9953599a02798a813a
  ```

  - step 3: create proposal review transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal review -w temp.dat --fee 0.0001 --payload c12345710ce15416569cd4f0c7d8d1639cb113521b2f94162b96fceba2dec7f600d412b9463fe40e254369f0fafba939c567c2b12d8d36ca443a849267372025960412345678677278335e5528c304acda6fb59d612c340cc17022 --ownersignature 514c10f9c12575fc747dfdcec1a21ba8796256b12f444bd27400e2c99a6fa4c94e41b20324a1c549566b8a1e0442eaf9abb90d485ecd7f9953599a02798a813a

  Hex:  092601c12345710ce15416569cd4f0c7d8d1639cb113521b2f94162b96fceba2dec7f600d412b9463fe40e254369f0fafba939c567c2b12d8d36ca443a849267372025960412345678677278335e5528c304acda6fb59d612c340cc1702240514c10f9c12575fc747dfdcec1a21ba8796256b12f444bd27400e2c99a6fa4c94e41b20324a1c549566b8a1e0442eaf9abb90d485ecd7f9953599a02798a813a0100133537333232383732303132303236343036313501044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```



- Build tracking proposal

  - step 1:  generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal tracking ownerpayload --proposalhash 47aa11310c587706963e5f4cc5009b06f74ebe5afbf8d515f446682db0aa6bd6 --messagehash 6ee596f99bb40da00f8cc53982cb0ace8620b4da926d14d0a9b3e3d4c6074404 --messagedata 123456 --stage 0 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842

  payload:  d66baab02d6846f415d5f8fb5abe4ef7069b00c54c5f3e960677580c3111aa47044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e031234560021024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984200
  digest:  482d2f8479847707407fdfca0f4a50ffd9db1e6edaf15e2f519d92dadc320da7
  ```

  - step 2: owner sign the digest of payload

  - step 3: generate new owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal tracking newownerpayload --payload d66baab02d6846f415d5f8fb5abe4ef7069b00c54c5f3e960677580c3111aa47044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e031234560021024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984200 --ownersignature 99683652421572f92f9baea2b25db7a72bec53d2d57bf7e5d64dbd9a78dcc39f42dc31139e6c6c5e2520444adca107ae0bf64f52fe312c848b95c730c3a55006

  payload:  d66baab02d6846f415d5f8fb5abe4ef7069b00c54c5f3e960677580c3111aa47044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e031234560021024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842004099683652421572f92f9baea2b25db7a72bec53d2d57bf7e5d64dbd9a78dcc39f42dc31139e6c6c5e2520444adca107ae0bf64f52fe312c848b95c730c3a55006
  digest:  984b8a522d54e14b822459958a2f7f96858d7bc5ed5c1c2703d090449fb7a2bd
  ```

  - step 4: new owner sign the digest of payload (If this proposal tracking is not use for changing owner, skip this step and signature will be empty.)

  - step 5:  generate secretary general unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal tracking secretarygeneralpayload --payload d66baab02d6846f415d5f8fb5abe4ef7069b00c54c5f3e960677580c3111aa47044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e031234560021024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842004099683652421572f92f9baea2b25db7a72bec53d2d57bf7e5d64dbd9a78dcc39f42dc31139e6c6c5e2520444adca107ae0bf64f52fe312c848b95c730c3a55006 --type 0 --secretarygeneralopinionhash 206d27b8ecf234e7a02ecea450eba16208cfecb831fff90022bada28e39fa2f5 --secretarygeneralopiniondata 12345678

  payload:  d66baab02d6846f415d5f8fb5abe4ef7069b00c54c5f3e960677580c3111aa47044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e031234560021024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842004099683652421572f92f9baea2b25db7a72bec53d2d57bf7e5d64dbd9a78dcc39f42dc31139e6c6c5e2520444adca107ae0bf64f52fe312c848b95c730c3a550060000f5a29fe328daba2200f9ff31b8eccf0862a1eb50a4ce2ea0e734f2ecb8276d200412345678
  digest:  499f8a67f7a8d3f468378b0b42dae9ec503ccd02600ed007fff636fd235379b9
  ```

  - step 6: secretary general sign the digest of payload

  - step 7: create proposal tracking transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal tracking -w temp.dat --fee 0.0001 --payload d66baab02d6846f415d5f8fb5abe4ef7069b00c54c5f3e960677580c3111aa47044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e031234560021024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842004099683652421572f92f9baea2b25db7a72bec53d2d57bf7e5d64dbd9a78dcc39f42dc31139e6c6c5e2520444adca107ae0bf64f52fe312c848b95c730c3a550060000f5a29fe328daba2200f9ff31b8eccf0862a1eb50a4ce2ea0e734f2ecb8276d200412345678 --secretarygeneralsignature da798522a17142192ef05f0bb219f988f12acc657f7e1381f1950eb1a6dd1f25674d5ab05f22c68f160e65a3e4e890a8793720f1cebd95e2cfe7600b93a38582

  Hex:  092701d66baab02d6846f415d5f8fb5abe4ef7069b00c54c5f3e960677580c3111aa47044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e031234560021024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842004099683652421572f92f9baea2b25db7a72bec53d2d57bf7e5d64dbd9a78dcc39f42dc31139e6c6c5e2520444adca107ae0bf64f52fe312c848b95c730c3a550060000f5a29fe328daba2200f9ff31b8eccf0862a1eb50a4ce2ea0e734f2ecb8276d20041234567840da798522a17142192ef05f0bb219f988f12acc657f7e1381f1950eb1a6dd1f25674d5ab05f22c68f160e65a3e4e890a8793720f1cebd95e2cfe7600b93a3858201001237383236303037383438333039303439303001044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```

- Build election proposal

  - step 1:  generate unsigned payload for owner and secretary general

  ```shell
  $ ./ela-cli wallet buildtx proposal election unsignedpayload --category 123456 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --drafthash 588fd76241b31901d8a6c57d4cfb3a533540309627abaa58062e36a2e9cdb384 --draftdata 12345678 --secretarypublickey 02029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1 --secretarydid iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL

  payload:  00040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984284b3cde9a2362e0658aaab2796304035533afb4c7dc5a6d80119b34162d78f5804123456782102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb167374ca50b174e928024db2441ff0b3d6682130ab9
  digest:  c7c5199e099c20ea87057d91d1865aa831ddf1c9bcaf0615cb65006252b301f8
  ```

  - step 2: owner and secretary general sign the digest of payload

  - step 3: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal election crcmemberpayload --payload 00040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984284b3cde9a2362e0658aaab2796304035533afb4c7dc5a6d80119b34162d78f5804123456782102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb167374ca50b174e928024db2441ff0b3d6682130ab9 --ownersignature b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e --secretarygeneralsignature b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e --crcmemberdid iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL

  payload:  00040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984284b3cde9a2362e0658aaab2796304035533afb4c7dc5a6d80119b34162d78f5804123456782102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb167374ca50b174e928024db2441ff0b3d6682130ab940b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e40b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e67374ca50b174e928024db2441ff0b3d6682130ab9
  digest:  f2eb841de9be0053541e4f9ce8524974b7c5d039c970693741a830dfd5aec4cf
  ```

  - step 4: crcouncil member sign the digest of payload

  - step 5: create proposal election transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal election -w temp.dat --fee 0.0001 --payload 00040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984284b3cde9a2362e0658aaab2796304035533afb4c7dc5a6d80119b34162d78f5804123456782102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb167374ca50b174e928024db2441ff0b3d6682130ab940b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e40b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e67374ca50b174e928024db2441ff0b3d6682130ab9 --crcmembersignature 15f14a593813b78355d75307ba75ba77a2536760f2957ae9e0e1f07511563ae95473a9f4b5f59d2ef0fb7a61631938a8b015501cc4ecb9fd65167aab1b0d1263

  Hex:  09250100040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984284b3cde9a2362e0658aaab2796304035533afb4c7dc5a6d80119b34162d78f5804123456782102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb167374ca50b174e928024db2441ff0b3d6682130ab940b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e40b058c9ef497fc5923f31131cf42778985f77acdf18c0deacf4a9a5b73bb15ef0ca7477ecc3cc059d7dae96cf76a39d0c995843e62c68a35d966b226052d90e3e67374ca50b174e928024db2441ff0b3d6682130ab94015f14a593813b78355d75307ba75ba77a2536760f2957ae9e0e1f07511563ae95473a9f4b5f59d2ef0fb7a61631938a8b015501cc4ecb9fd65167aab1b0d12630100133335303634353931393535383032373330353501044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```



- Build change owner proposal

  - step 1:  generate unsigned payload for owner and new owner

  ```shell
  $ ./ela-cli wallet buildtx proposal changeowner unsignedpayload --category 123456 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --drafthash b63a76a99c9843584e220796005bfc256d46c09f8fa651772555bcb44fdb640f --draftdata 1234 --targetproposalhash 9ed0b81fd676f9816b5509fa1610c65065bf0c10ce727bc2206ea994435ae759 --newrecipient EWRwrmBWpYFwnvAQffcP1vrPCS5sGTgWEB --newownerpubkey 02029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1

  payload:  01040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420f64db4fb4bc55257751a68f9fc0466d25fc5b009607224e5843989ca9763ab602123459e75a4394a96e20c27b72ce100cbf6550c61016fa09556b81f976d61fb8d09e2191a243d0877a3e8e8093acad39a14b3b56a4e00d2102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1
  digest:  b12f95d4cde661bc7091a9c5bab34a7d60d138f04bddbc5e06e19921bc20fc6f
  ```

  - step 2: owner and new owner sign the digest of payload

  - step 3: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal changeowner crcmemberpayload --payload 01040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420f64db4fb4bc55257751a68f9fc0466d25fc5b009607224e5843989ca9763ab602123459e75a4394a96e20c27b72ce100cbf6550c61016fa09556b81f976d61fb8d09e2191a243d0877a3e8e8093acad39a14b3b56a4e00d2102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1 --ownersignature b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db --newownersignature b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db --crcmemberdid iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL

  payload:  01040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420f64db4fb4bc55257751a68f9fc0466d25fc5b009607224e5843989ca9763ab602123459e75a4394a96e20c27b72ce100cbf6550c61016fa09556b81f976d61fb8d09e2191a243d0877a3e8e8093acad39a14b3b56a4e00d2102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb140b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db40b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db67374ca50b174e928024db2441ff0b3d6682130ab9
  digest:  6a98f83b6f14e82c3a7c73fca9954977812f13197d97e3134683045fb2881d79
  ```

  - step 4: crcouncil member sign the digest of payload
  - step 5: create change owner proposal transaction

  ```shell
  $./ela-cli wallet buildtx proposal changeowner --payload 01040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420f64db4fb4bc55257751a68f9fc0466d25fc5b009607224e5843989ca9763ab602123459e75a4394a96e20c27b72ce100cbf6550c61016fa09556b81f976d61fb8d09e2191a243d0877a3e8e8093acad39a14b3b56a4e00d2102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb140b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db40b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db67374ca50b174e928024db2441ff0b3d6682130ab9 -w temp.dat --fee 0.0001 --crcmembersignature d5c8b26ba659acd9261e5789015d2401ebe5490bc6f7521a445a92a9c64272be105ea2102dc50fd482b514a83450d63201bb8c408d40cf676c0b09342c3a79e0

  Hex:  09250101040631323334353621024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420f64db4fb4bc55257751a68f9fc0466d25fc5b009607224e5843989ca9763ab602123459e75a4394a96e20c27b72ce100cbf6550c61016fa09556b81f976d61fb8d09e2191a243d0877a3e8e8093acad39a14b3b56a4e00d2102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb140b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db40b62059cf528326a6a2c52d66335169e171d1bbfdb834a488de1d562587300b8cfabe384f62825764981e14126c83f8f84afc4282577620dfc62b537fb4caa6db67374ca50b174e928024db2441ff0b3d6682130ab940d5c8b26ba659acd9261e5789015d2401ebe5490bc6f7521a445a92a9c64272be105ea2102dc50fd482b514a83450d63201bb8c408d40cf676c0b09342c3a79e00100133333343330333137313733363538393739333701044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```

- Build terminate proposal

  - step 1:  generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal terminate ownerpayload --category 1234 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --drafthash 213808581f0198fcdcce0fc0bd63c9522d5de857aeec1ed020d0a6ec57c98ba2 --draftdata 12345678 --targetproposalhash 90d21147b9c56ec3feea8e3d9c26274c8804cdc03c368e596e091ca14ddca20f

  payload:  0204043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a28bc957eca6d020d01eecae57e85d2d52c963bdc00fcedcfc98011f5808382104123456780fa2dc4da11c096e598e363cc0cd04884c27269c3d8eeafec36ec5b94711d290
  digest:  898501da6f21f00113b609ae4d9ffeccf9c0d56fcd6cb4b4ac96130895198c6c
  ```

  - step 2: owner sign the digest of payload
  - step 3: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal terminate crcmemberpayload --payload 0204043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a28bc957eca6d020d01eecae57e85d2d52c963bdc00fcedcfc98011f5808382104123456780fa2dc4da11c096e598e363cc0cd04884c27269c3d8eeafec36ec5b94711d290 --ownersignature d32ac7c5c66cfca0037a38aac47fbe4f83a6ecd11a5f9fed6d9aaf7ca4896820f4bb55628b2462a7e8f63db40fcc3c08002f3ab6781a2593b1cd6b4f3bb945b3 --crcmemberdid iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL

  payload:  0204043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a28bc957eca6d020d01eecae57e85d2d52c963bdc00fcedcfc98011f5808382104123456780fa2dc4da11c096e598e363cc0cd04884c27269c3d8eeafec36ec5b94711d29040d32ac7c5c66cfca0037a38aac47fbe4f83a6ecd11a5f9fed6d9aaf7ca4896820f4bb55628b2462a7e8f63db40fcc3c08002f3ab6781a2593b1cd6b4f3bb945b367374ca50b174e928024db2441ff0b3d6682130ab9
  digest:  0f8669e0125c7d41851592304a09a1a1f2896e154355a594e2587ee68e62c24a
  ```

  - step 4: crcouncil member sign the digest of payload
  - step 5: create terminate proposal transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal terminate -w temp.dat --fee 0.0001 --payload 0204043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a28bc957eca6d020d01eecae57e85d2d52c963bdc00fcedcfc98011f5808382104123456780fa2dc4da11c096e598e363cc0cd04884c27269c3d8eeafec36ec5b94711d29040d32ac7c5c66cfca0037a38aac47fbe4f83a6ecd11a5f9fed6d9aaf7ca4896820f4bb55628b2462a7e8f63db40fcc3c08002f3ab6781a2593b1cd6b4f3bb945b367374ca50b174e928024db2441ff0b3d6682130ab9 --crcmembersignature a8c1b74bf14abee12a3c6d61bb5d125dc5d210c087e6b7718ca8849a68858cb9e844e0e68a5740d2ba49a38424c72b3cb5331b73a8bd5f5dba43ac8cbf8683dc

  Hex:  0925010204043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a28bc957eca6d020d01eecae57e85d2d52c963bdc00fcedcfc98011f5808382104123456780fa2dc4da11c096e598e363cc0cd04884c27269c3d8eeafec36ec5b94711d29040d32ac7c5c66cfca0037a38aac47fbe4f83a6ecd11a5f9fed6d9aaf7ca4896820f4bb55628b2462a7e8f63db40fcc3c08002f3ab6781a2593b1cd6b4f3bb945b367374ca50b174e928024db2441ff0b3d6682130ab940a8c1b74bf14abee12a3c6d61bb5d125dc5d210c087e6b7718ca8849a68858cb9e844e0e68a5740d2ba49a38424c72b3cb5331b73a8bd5f5dba43ac8cbf8683dc0100133630333433353031393532333033333831323101044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```



- Build reserve custom id proposal

  - step 1: generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal reservecustomid ownerpayload --category 1234 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --drafthash 81a0f6fb01476cf5b1ee857f514c682c78cf31cfa66397d819aeb8c5e73a4ca4 --draftdata 123456 --reservedcustomidlist "iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL|iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL|iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL"

  payload:  0005043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a44c3ae7c5b8ae19d89763a6cf31cf782c684c517f85eeb1f56c4701fbf6a08103123456032269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c
  digest:  93234304d28625869653ce976b43df8b03dc71a3d370a37b11a79e1b732a943d
  ```

  - step 2: owner sign the digest of payload
  - step 3: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal reservecustomid crcmemberpayload --payload 0005043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a44c3ae7c5b8ae19d89763a6cf31cf782c684c517f85eeb1f56c4701fbf6a08103123456032269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c --ownersignature 00e0afd1f94c5194a10d7ef051aacd3b6370c075033dfe2108838d631622f2abf9512eacba3fb7a6799465305a71460e80e42046cd87915422db8c60ab50f1c3 --crcmemberdid iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL

  payload:  0005043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a44c3ae7c5b8ae19d89763a6cf31cf782c684c517f85eeb1f56c4701fbf6a08103123456032269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c4000e0afd1f94c5194a10d7ef051aacd3b6370c075033dfe2108838d631622f2abf9512eacba3fb7a6799465305a71460e80e42046cd87915422db8c60ab50f1c367374ca50b174e928024db2441ff0b3d6682130ab9
  digest:  d48cb04eff71ca1cb291773ec64cce6d30c296814bbb2b19f1be14e9d1ee3ece
  ```

  - step 4: crcouncil member sign the digest of payload
  - step 5: create reserve custom id proposal transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal reservecustomid -w temp.dat --fee 0.0001 --payload 0005043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a44c3ae7c5b8ae19d89763a6cf31cf782c684c517f85eeb1f56c4701fbf6a08103123456032269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c4000e0afd1f94c5194a10d7ef051aacd3b6370c075033dfe2108838d631622f2abf9512eacba3fb7a6799465305a71460e80e42046cd87915422db8c60ab50f1c367374ca50b174e928024db2441ff0b3d6682130ab9 --crcmembersignature b8e0c80a3c18bd48196f22c4df355f9ed2a3e8c6bee77bee38f7574517753ccd2cb54ab863493553ed001776e59faf504d27482d736c583fef7955d7ffb46b3d

  Hex:  0925010005043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842a44c3ae7c5b8ae19d89763a6cf31cf782c684c517f85eeb1f56c4701fbf6a08103123456032269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c4000e0afd1f94c5194a10d7ef051aacd3b6370c075033dfe2108838d631622f2abf9512eacba3fb7a6799465305a71460e80e42046cd87915422db8c60ab50f1c367374ca50b174e928024db2441ff0b3d6682130ab940b8e0c80a3c18bd48196f22c4df355f9ed2a3e8c6bee77bee38f7574517753ccd2cb54ab863493553ed001776e59faf504d27482d736c583fef7955d7ffb46b3d0100133634343338383538303331373539353339343001044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```

- Build receive custom id proposal

  - step 1: generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal receivecustomid ownerpayload --category 1234 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --drafthash 5d154fa751e8bb04e220c8237086d60cb2261bfffcbc646a9b4bcce60210fd0b --draftdata 123456 --receivedcustomidlist "iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL|iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL|iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL|iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL" --receiverdid idFPSjznfyrFRivJuBWLiYZCTbC4wcnoVi

  payload:  0105043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420bfd1002e6cc4b9b6a64bcfcff1b26b20cd6867023c820e204bbe851a74f155d03123456042269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c677278335e5528c304acda6fb59d612c340cc17022
  digest:  c8119a6b3e366fcce0d8898c433681830409c08f420352a5a664c785be34a9fa
  ```

  - step 2: owner sign the digest of payload
  - step 3: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal receivecustomid crcmemberpayload --payload 0105043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420bfd1002e6cc4b9b6a64bcfcff1b26b20cd6867023c820e204bbe851a74f155d03123456042269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c677278335e5528c304acda6fb59d612c340cc17022 --ownersignature 45eabc522587bcee261b4493b70993eeffa3d5cb4f016bc72bea5487d5865fad0e0873ece69003f3d5817d5742f1b3b993f05e4224f0efca5ad4bb3b633fff35 --crcmemberdid iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL

  payload:  0105043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420bfd1002e6cc4b9b6a64bcfcff1b26b20cd6867023c820e204bbe851a74f155d03123456042269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c677278335e5528c304acda6fb59d612c340cc170224045eabc522587bcee261b4493b70993eeffa3d5cb4f016bc72bea5487d5865fad0e0873ece69003f3d5817d5742f1b3b993f05e4224f0efca5ad4bb3b633fff3567374ca50b174e928024db2441ff0b3d6682130ab9
  digest:  df57b1cd87d1106de8d62a50cd7fd974685a2610b5e7d76757b837f1533ef65e
  ```

  - step 4: crcouncil member sign the digest of payload
  - step 5: create receive custom id proposal transaction

  ```shell
  $  ./ela-cli wallet buildtx proposal receivecustomid -w temp.dat --fee 0.0001 --payload 0105043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420bfd1002e6cc4b9b6a64bcfcff1b26b20cd6867023c820e204bbe851a74f155d03123456042269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c677278335e5528c304acda6fb59d612c340cc170224045eabc522587bcee261b4493b70993eeffa3d5cb4f016bc72bea5487d5865fad0e0873ece69003f3d5817d5742f1b3b993f05e4224f0efca5ad4bb3b633fff3567374ca50b174e928024db2441ff0b3d6682130ab9 --crcmembersignature 3619218d7f6cbddba75d08c66b1dee6c231234efef4522520885b12456b5acfc4d42f6a2ca67ea034cf1fea2f2b4b1f63f5313145446c1668fa04fe24a8beda4

  Hex:  0925010105043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98420bfd1002e6cc4b9b6a64bcfcff1b26b20cd6867023c820e204bbe851a74f155d03123456042269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c2269587258513677454e617a4a5641624d47634470335a594162517863655139674a4c677278335e5528c304acda6fb59d612c340cc170224045eabc522587bcee261b4493b70993eeffa3d5cb4f016bc72bea5487d5865fad0e0873ece69003f3d5817d5742f1b3b993f05e4224f0efca5ad4bb3b633fff3567374ca50b174e928024db2441ff0b3d6682130ab9403619218d7f6cbddba75d08c66b1dee6c231234efef4522520885b12456b5acfc4d42f6a2ca67ea034cf1fea2f2b4b1f63f5313145446c1668fa04fe24a8beda40100133737333036313339383330333539393939393801044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```

- Build change custom id fee proposal

  - step 1:  generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal changecustomidfee ownerpayload --category 1234 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --drafthash a1456736ed2ca5b8fc755b884a472deddcd8d5889572ea2ffeb265ed93c5b93b --draftdata 123456 --customidfeerate "10000|12345678"

  payload:  0205043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98423bb9c593ed65b2fe2fea729588d5d8dced2d474a885b75fcb8a52ced366745a10312345610270000000000004e61bc00
  digest:  bf9e0d7259a0f9f84b11433786c5d8f251f06c22d98387ac59cf0a682557cca4
  ```

  - step 2: owner sign the digest of payload
  - step 3: generate crcouncil member unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal changecustomidfee crcmemberpayload --payload 0205043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98423bb9c593ed65b2fe2fea729588d5d8dced2d474a885b75fcb8a52ced366745a10312345610270000000000004e61bc00 --ownersignature a872e9313fabd7a0cbd16ba74394390050abfaadadf6a28d24d972a1cadda90097f39d2bd82b2d6db9800dfae3e38f61dc6a4f4fdfd2a3e7e2b100fc11ce628b --crcmemberdid idFPSjznfyrFRivJuBWLiYZCTbC4wcnoVi

  payload:  0205043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98423bb9c593ed65b2fe2fea729588d5d8dced2d474a885b75fcb8a52ced366745a10312345610270000000000004e61bc0040a872e9313fabd7a0cbd16ba74394390050abfaadadf6a28d24d972a1cadda90097f39d2bd82b2d6db9800dfae3e38f61dc6a4f4fdfd2a3e7e2b100fc11ce628b677278335e5528c304acda6fb59d612c340cc17022
  digest:  4afaca6fdca4480e7bb8a838433ea993b4ac4dc00c5380bed75c8c184369a448
  ```

  - step 4: crcouncil member sign the digest of payload
  - step 5: create chage custom id fee proposal transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal changecustomidfee -w temp.dat --fee 0.0001 --payload 0205043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98423bb9c593ed65b2fe2fea729588d5d8dced2d474a885b75fcb8a52ced366745a10312345610270000000000004e61bc0040a872e9313fabd7a0cbd16ba74394390050abfaadadf6a28d24d972a1cadda90097f39d2bd82b2d6db9800dfae3e38f61dc6a4f4fdfd2a3e7e2b100fc11ce628b677278335e5528c304acda6fb59d612c340cc17022 --crcmembersignature cc7f0b3dd78718635698bbe0e4676245795a80f177bc43c78e30ddb7552074f36348a8e4ad5824c7b565649849e9af5ca59305c361537924b353c8170c3f6bec

  Hex:  0925010205043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98423bb9c593ed65b2fe2fea729588d5d8dced2d474a885b75fcb8a52ced366745a10312345610270000000000004e61bc0040a872e9313fabd7a0cbd16ba74394390050abfaadadf6a28d24d972a1cadda90097f39d2bd82b2d6db9800dfae3e38f61dc6a4f4fdfd2a3e7e2b100fc11ce628b677278335e5528c304acda6fb59d612c340cc1702240cc7f0b3dd78718635698bbe0e4676245795a80f177bc43c78e30ddb7552074f36348a8e4ad5824c7b565649849e9af5ca59305c361537924b353c8170c3f6bec01001238343235323437383334353938343636363001044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```

- Build register sidechain proposal

  - step 1: generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal registersidechain ownerpayload --category 1234 --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --drafthash 8986b676681c741428c05d6574cd2ab83630690c5205e4d8e71c8ddca069d947 --draftdata 123456 --sidechaininfo "ethterum|12348765|0dc1b91aeee9b8ab609283187d9738927935497992e9ffc6216d14b3452a4d09|120000|1010101|http://baidu.com"

  payload:  1004043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984247d969a0dc8d1ce7d8e405520c693036b82acd74655dc02814741c6876b686890312345608657468746572756d5d6dbc00094d2a45b3146d21c6ffe992794935799238977d18839260abb8e9ee1ab9c10dc0d4010000000000b5690f0010687474703a2f2f62616964752e636f6d
  digest:  c7c66307bed0f77ce3dfd172853c5bf51ca144d75df9573c9d32af4c4a0f2cba
  ```

  - step 2: owner sign the digest of payload
  - step 3: generate crcouncil member unsigned payloaod

  ```shell
  $ ./ela-cli wallet buildtx proposal registersidechain crcmemberpayload --payload 1004043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984247d969a0dc8d1ce7d8e405520c693036b82acd74655dc02814741c6876b686890312345608657468746572756d5d6dbc00094d2a45b3146d21c6ffe992794935799238977d18839260abb8e9ee1ab9c10dc0d4010000000000b5690f0010687474703a2f2f62616964752e636f6d --ownersignature d5d522e987e773ba74cdb0b2b251a90e70cddd4003ae602b2406cc280b82e66299816b865765c64da3b09c044db837acad2900b1e2f6dfbce127034e26777302 --crcmemberdid iXrXQ6wENazJVAbMGcDp3ZYAbQxceQ9gJL

  payload:  1004043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984247d969a0dc8d1ce7d8e405520c693036b82acd74655dc02814741c6876b686890312345608657468746572756d5d6dbc00094d2a45b3146d21c6ffe992794935799238977d18839260abb8e9ee1ab9c10dc0d4010000000000b5690f0010687474703a2f2f62616964752e636f6d40d5d522e987e773ba74cdb0b2b251a90e70cddd4003ae602b2406cc280b82e66299816b865765c64da3b09c044db837acad2900b1e2f6dfbce127034e2677730267374ca50b174e928024db2441ff0b3d6682130ab9
  digest:  104b9c0a2d6d803dc9159b4e876a19602b5dc22d267433b39a89908950953ded
  ```

  - step 4: crcouncil member sign the digest of payload
  - step 5: create register sidechain proposal transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal registersidechain  -w temp.dat --fee 0.0001 --payload 1004043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984247d969a0dc8d1ce7d8e405520c693036b82acd74655dc02814741c6876b686890312345608657468746572756d5d6dbc00094d2a45b3146d21c6ffe992794935799238977d18839260abb8e9ee1ab9c10dc0d4010000000000b5690f0010687474703a2f2f62616964752e636f6d40d5d522e987e773ba74cdb0b2b251a90e70cddd4003ae602b2406cc280b82e66299816b865765c64da3b09c044db837acad2900b1e2f6dfbce127034e2677730267374ca50b174e928024db2441ff0b3d6682130ab9 --crcmembersignature 2d76001194863e30df43119f1b4828fda6e3577654395c457467fa0495489cad64d6f83477cc6031cd2178a2c6df25c380b6335643dd41ca4c0670e765a92c6a

  Hex:  0925011004043132333421024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc984247d969a0dc8d1ce7d8e405520c693036b82acd74655dc02814741c6876b686890312345608657468746572756d5d6dbc00094d2a45b3146d21c6ffe992794935799238977d18839260abb8e9ee1ab9c10dc0d4010000000000b5690f0010687474703a2f2f62616964752e636f6d40d5d522e987e773ba74cdb0b2b251a90e70cddd4003ae602b2406cc280b82e66299816b865765c64da3b09c044db837acad2900b1e2f6dfbce127034e2677730267374ca50b174e928024db2441ff0b3d6682130ab9402d76001194863e30df43119f1b4828fda6e3577654395c457467fa0495489cad64d6f83477cc6031cd2178a2c6df25c380b6335643dd41ca4c0670e765a92c6a0100133737333733333533303337343231303830323901044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```

- Build withdraw proposal

  - step 1: generate owner unsigned payload

  ```shell
  $ ./ela-cli wallet buildtx proposal withdraw ownerpayload --proposalhash 9b136ed7dc7c9b7760c58c9a08cfc20e04615302a7ac792a51f337ee6c97e4aa --ownerpubkey 024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc9842 --recipient EWRwrmBWpYFwnvAQffcP1vrPCS5sGTgWEB --amount 5000

  payload:  aae4976cee37f3512a79aca7025361040ec2cf089a8cc560779b7cdcd76e139b21024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98422191a243d0877a3e8e8093acad39a14b3b56a4e00d0088526a74000000
  digest:  b253a795b8e10184853a3dcad70f13cb8de071810e8982a3e1258021df079c79
  ```

  - step 2: owner sign the digest of payload
  - step 3: create proposal withdraw transaction

  ```shell
  $ ./ela-cli wallet buildtx proposal withdraw -w temp.dat --fee 0.0001 --payload aae4976cee37f3512a79aca7025361040ec2cf089a8cc560779b7cdcd76e139b21024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98422191a243d0877a3e8e8093acad39a14b3b56a4e00d0088526a74000000 --signature a73e6dd48a42dc7cfdbc22041c0b50f27faae474976f3d9de9f6a417c73ea9855a4e1cc359d2eaf1483bcd8479b3ba1ab6b07c30114aa920d40be2e080439b29

  Hex:  092901aae4976cee37f3512a79aca7025361040ec2cf089a8cc560779b7cdcd76e139b21024b89fe23a982325126058cea21374bc6599ebe38fd950a8f9bf2804f2bfc98422191a243d0877a3e8e8093acad39a14b3b56a4e00d0088526a7400000040a73e6dd48a42dc7cfdbc22041c0b50f27faae474976f3d9de9f6a417c73ea9855a4e1cc359d2eaf1483bcd8479b3ba1ab6b07c30114aa920d40be2e080439b290100133638363731323538303135333734313332333501044407c6d4e3b3a9d0146d92dab42086ce0acb8239c58c0fa00db49bf996e56e0000ffffffff01b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a38ed1898627900300000000002191a243d0877a3e8e8093acad39a14b3b56a4e00d00000000000100232102029bf3f8d9945dda1ffa52ec6c1b168e9b3a56e74463453d2815891cdb9b8fb1ac
  File:  to_be_signed.txn
  ```

### 2.2 Sign To Transaction

The transaction build by buildtx command, should be signed before sending to ela node.

--hex

The `hex` parameter is used to specify the raw transaction hex string which to be signed.

--file, -f

The `file` parameter is used to specify the raw transaction file which to be signed.

#### 2.2.1 Standard Signature

```
./ela-cli wallet signtx -f to_be_signed.txn
```

Enter a password when prompted.

Result:

```
[ 1 / 1 ] Transaction successfully signed
Hex:  090200010013373934373338313938363837333037373435330139eaa4137a047cdaac7112fe04617902e3b17c1685029d259d93c77680c950f30100ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3809698000000000000000000210fcd528848be05f8cffe5d99896c44bdeec7050200b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a36ea2d2090000000000000000210d4109bf00e6d782db40ab183491c03cf4d6a37a0000000000014140b3963ac174e922592905154c34519765c246441132707ccd6b4fb98d6107cc391e180d106bb02ce8e895d4662eb353f72f4859366ddf01fb97ae32f39bf6d1752321034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16cac
File:  ready_to_send.txn
```

#### 2.2.2 Multi Signature

Attach the first signature:

```
./ela-cli wallet signtx -f to_be_signed.txn -w keystore1.dat
```

Result:

```
[ 1 / 3 ] Transaction successfully signed
Hex:  0902000100133334393234313234323933333338333335313701737a31035ebe8dfe3c58c7b9ff7eb13485387cd2010d894f39bf670ccd1f62180000ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3c0320a030000000000000000214e6334d41d86e3c3a32698bdefe974d6960346b300b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3a0108f380000000000000000125bc115b91913c9c6347f0e0e3ba3b75c80b9481100000000000141406952da35f525b4db087654f2b0317b6ea7d385d22539f5ecf2001cc6db169be0b6945e073180796f2a070a29e90c1c62e5b9a3f8b3f040b3fe8ea6be57da5a858b53210325406f4abc3d41db929f26cf1a419393ed1fe5549ff18f6c579ff0c3cbb714c8210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd2103fd77d569f766638677755e8a71c7c085c51b675fbf7459ca1094b29f62f0b27d54ae
File:  to_be_signed_1_of_3.txn
```

Attach the second signature:

```
./ela-cli wallet signtx -f to_be_signed_1_of_3.txn -w keystore2.dat
```

Result:

```
[ 2 / 3 ] Transaction successfully signed
Hex:  0902000100133334393234313234323933333338333335313701737a31035ebe8dfe3c58c7b9ff7eb13485387cd2010d894f39bf670ccd1f62180000ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3c0320a030000000000000000214e6334d41d86e3c3a32698bdefe974d6960346b300b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3a0108f380000000000000000125bc115b91913c9c6347f0e0e3ba3b75c80b9481100000000000182406952da35f525b4db087654f2b0317b6ea7d385d22539f5ecf2001cc6db169be0b6945e073180796f2a070a29e90c1c62e5b9a3f8b3f040b3fe8ea6be57da5a854089aef5f476793d22f2403b575463129041bc37392109f3a96b43a4a94876b43991d40b8fd0ee591a6faa948aea9053fa7a767a21d8772958c498dfb753f576d68b53210325406f4abc3d41db929f26cf1a419393ed1fe5549ff18f6c579ff0c3cbb714c8210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd2103fd77d569f766638677755e8a71c7c085c51b675fbf7459ca1094b29f62f0b27d54ae
File:  to_be_signed_2_of_3.txn
```

Attach the third signature:

```
./ela-cli wallet signtx -f to_be_signed_2_of_3.txn -w keystore3.dat
```

Result:

```
[ 3 / 3 ] Transaction successfully signed
Hex:  0902000100133334393234313234323933333338333335313701737a31035ebe8dfe3c58c7b9ff7eb13485387cd2010d894f39bf670ccd1f62180000ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3c0320a030000000000000000214e6334d41d86e3c3a32698bdefe974d6960346b300b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3a0108f380000000000000000125bc115b91913c9c6347f0e0e3ba3b75c80b94811000000000001c3406952da35f525b4db087654f2b0317b6ea7d385d22539f5ecf2001cc6db169be0b6945e073180796f2a070a29e90c1c62e5b9a3f8b3f040b3fe8ea6be57da5a854089aef5f476793d22f2403b575463129041bc37392109f3a96b43a4a94876b43991d40b8fd0ee591a6faa948aea9053fa7a767a21d8772958c498dfb753f576d6408dfd58568cd95be995fa69ab9924c28c23fdc738caf19fb243a285c1cda01793d676c0459d546e92cd69b4f9837f750adc3b6f66fe0eff5dcd5a9cc46c1ec6138b53210325406f4abc3d41db929f26cf1a419393ed1fe5549ff18f6c579ff0c3cbb714c8210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd210353059bf157d3eaca184cc10a80f10baf10676c4a39695a9a092fa3c2934818bd2103fd77d569f766638677755e8a71c7c085c51b675fbf7459ca1094b29f62f0b27d54ae
File:  ready_to_send.txn
```

### 2.3 Sign Digest

```
$ ./ela-cli wallet signdigest -w temp.dat -p 123 --digest 1cf5667b3913a004e8780b31c3ec7d0103277c94d50b2966bb5473db75c8055d
```

Result:

```
Signature:  eb6d7f598ba0f06bb298caff9294453f74c7b229bed598d159b1a223ec1c3f86facb3cf52d705a66f36f2d97cdae89e6bb785540ff537abc4d26225555980711
```

### 2.4 Verify Digest

```shell
$ ./ela-cli wallet verifydigest -w temp.dat -p 123 --digest 1cf5667b3913a004e8780b31c3ec7d0103277c94d50b2966bb5473db75c8055d --signature eb6d7f598ba0f06bb298caff9294453f74c7b229bed598d159b1a223ec1c3f86facb3cf52d705a66f36f2d97cdae89e6bb785540ff537abc4d26225555980711
```

Result:

```shell
verify digest: true
```

### 2.5 Send Transaction

```
./ela-cli wallet sendtx -f ready_to_send.txn
```

Result:

```
5b9673a813b90dd73f6d21f478736c7e08bba114c3772618fca232341af683b5
```

### 2.6 Show Transaction

The showtx command parses the contents from the raw transaction. You can specify the raw transaction by `hex` or `file` parameters.

```
./ela-cli wallet showtx --hex 0902000100123132333835343835313135373533343433340139eaa4137a047cdaac7112fe04617902e3b17c1685029d259d93c77680c950f30100ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a380778e06000000000000000012e194f97570ada85ec5df31e7c192edf1e3fc199900b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a36ec1dc030000000000000000210d4109bf00e6d782db40ab183491c03cf4d6a37a0000000000014140433f2f2fa7390db75af3a3288943ce178f9373b2362a3b09530dd30fbb50fa5f007716bf07abd7ee0281d422f3615b474ed332ba63fe7209f611e547d68e4c0f2321034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16cac
```

Result:

```
Transaction: {
	Hash: 4661854b9441d52f78f5f4c025b37270af3aaa60332c4bfe041fa3cbc91b0a7e
	Version: 9
	TxType: TransferAsset
	PayloadVersion: 0
	Payload: 00
	Attributes: [Attribute: {
		Usage: Nonce
		Data: 313233383534383531313537353334343334
		}]
	Inputs: [{TxID: 39eaa4137a047cdaac7112fe04617902e3b17c1685029d259d93c77680c950f3 Index: 1 Sequence: 4294967295}]
	Outputs: [Output: {
	AssetID: b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3
	Value: 1.10000000
	OutputLock: 0
	ProgramHash: 12e194f97570ada85ec5df31e7c192edf1e3fc1999
	Type: 0
	Payload: &{}
	} Output: {
	AssetID: b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3
	Value: 0.64799086
	OutputLock: 0
	ProgramHash: 210d4109bf00e6d782db40ab183491c03cf4d6a37a
	Type: 0
	Payload: &{}
	}]
	LockTime: 0
	Programs: [Program: {
		Code: 21034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16cac
		Parameter: 40433f2f2fa7390db75af3a3288943ce178f9373b2362a3b09530dd30fbb50fa5f007716bf07abd7ee0281d422f3615b474ed332ba63fe7209f611e547d68e4c0f
		}]
	}
```



## 3. Get Blockchian Information

```
NAME:
   ela-cli info - With ela-cli info, you could look up node status, query blocks, transactions, etc.

USAGE:
   ela-cli info command [command options] [args]

COMMANDS:
     getconnectioncount  Show how many peers are connected
     getneighbors        Show neighbor nodes information
     getnodestate        Show current node status
     getcurrentheight    Get best block height
     getbestblockhash    Get the best block hash
     getblockhash        Get a block hash by height
     getblock            Get a block details by height or block hash
     getrawtransaction   Get raw transaction by transaction hash
     getrawmempool       Get transaction details in node mempool

OPTIONS:
   --help, -h  show help
```

### 3.1 Get Connections Count

```
./ela-cli info getconnectioncount
```

Result:

```
1
```

### 3.2 Get Neighbors Information

```
./ela-cli info getneighbors
```

Result:

```
[
    "127.0.0.1:30338 (outbound)"
]
```

### 3.3 Get Node State

```
./ela-cli info getnodestate
```

Result:

```
[
    {
        "Addr": "127.0.0.1:30338",
        "ConnTime": "2019-02-28T14:32:44.996711+08:00",
        "ID": 4130841897781718000,
        "Inbound": false,
        "LastBlock": 395,
        "LastPingMicros": 0,
        "LastPingTime": "0001-01-01T00:00:00Z",
        "LastRecv": "2019-02-28T14:32:45+08:00",
        "LastSend": "2019-02-28T14:32:45+08:00",
        "RelayTx": 0,
        "Services": 3,
        "StartingHeight": 395,
        "TimeOffset": 0,
        "Version": 10002
    }
]
```

### 3.4 Get Current Height

```
./ela-cli info getcurrentheight
```

Result:

```
395
```

### 3.5 Get Best Block Hash

```
./ela-cli info getbestblockhash
```

Result:

```
"0affad77eacef8d5e69bebd1edd24b43ca8d8948dade9e23b14a9d8ceca060e6"
```

### 3.6 Get Block Hash By Height

```
./ela-cli info getblockhash 100
```

Result:

```
"1c1e1c22ce891184d390def30a9b8f15f355c05a7bd6e7e7912b571141e01415"
```

### 3.7 Get Block Information

Get block information by hash:

```
./ela-cli info getblock 1c1e1c22ce891184d390def30a9b8f15f355c05a7bd6e7e7912b571141e01415
```

Result:

```
{
    "auxpow": "01000000010000000000000000000000000000000000000000000000000000000000000000000000002cfabe6d6d1514e04111572b91e7e7d67b5ac055f3158f9b0af3de90d3841189ce221c1e1c0100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffff7f000000000000000000000000000000000000000000000000000000000000000073f93ed7b4401e54aef152505f4d4b88c7c5462453e77069455262bed04575df3f236e5c00000000e6ef0600",
    "bits": 520095999,
    "chainwork": "00000127",
    "confirmations": 296,
    "difficulty": "1",
    "hash": "1c1e1c22ce891184d390def30a9b8f15f355c05a7bd6e7e7912b571141e01415",
    "height": 100,
    "mediantime": 1550721855,
    "merkleroot": "8570ff4cd3d356be2f9d90333205f9d4b11bd234aa513bf16c53c503f6575718",
    "minerinfo": "",
    "nextblockhash": "0c1a3bddf4686c6be9be8c4be2d3915c542856adac8c7458bd056df83e434eaa",
    "nonce": 0,
    "previousblockhash": "05d6db028304da9c126ea34a2820bfda8a269e2ceb9bfdda73e70632ef13be7f",
    "size": 560,
    "strippedsize": 560,
    "time": 1550721855,
    "tx": [
        {
            "attributes": [
                {
                    "data": "33c33ebc08ea3c1a",
                    "usage": 0
                }
            ],
            "blockhash": "1c1e1c22ce891184d390def30a9b8f15f355c05a7bd6e7e7912b571141e01415",
            "blocktime": 1550721855,
            "confirmations": 296,
            "hash": "8570ff4cd3d356be2f9d90333205f9d4b11bd234aa513bf16c53c503f6575718",
            "locktime": 100,
            "payload": {
                "CoinbaseData": ""
            },
            "payloadversion": 4,
            "programs": [],
            "size": 254,
            "time": 1550721855,
            "txid": "8570ff4cd3d356be2f9d90333205f9d4b11bd234aa513bf16c53c503f6575718",
            "type": 0,
            "version": 0,
            "vin": [
                {
                    "sequence": 4294967295,
                    "txid": "0000000000000000000000000000000000000000000000000000000000000000",
                    "vout": 65535
                }
            ],
            "vout": [
                {
                    "address": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",
                    "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                    "n": 0,
                    "outputlock": 0,
                    "payload": null,
                    "type": 0,
                    "value": "1.50684931"
                },
                {
                    "address": "EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR",
                    "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                    "n": 1,
                    "outputlock": 0,
                    "payload": null,
                    "type": 0,
                    "value": "1.75799086"
                },
                {
                    "address": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",
                    "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                    "n": 2,
                    "outputlock": 0,
                    "payload": null,
                    "type": 0,
                    "value": "1.75799088"
                }
            ],
            "vsize": 254
        }
    ],
    "version": 0,
    "versionhex": "00000000",
    "weight": 2240
}
```

Get block information by height:

```
./ela-cli info getblock 100
```

Result:

```
{
    "auxpow": "01000000010000000000000000000000000000000000000000000000000000000000000000000000002cfabe6d6d1514e04111572b91e7e7d67b5ac055f3158f9b0af3de90d3841189ce221c1e1c0100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffff7f000000000000000000000000000000000000000000000000000000000000000073f93ed7b4401e54aef152505f4d4b88c7c5462453e77069455262bed04575df3f236e5c00000000e6ef0600",
    "bits": 520095999,
    "chainwork": "00000127",
    "confirmations": 296,
    "difficulty": "1",
    "hash": "1c1e1c22ce891184d390def30a9b8f15f355c05a7bd6e7e7912b571141e01415",
    "height": 100,
    "mediantime": 1550721855,
    "merkleroot": "8570ff4cd3d356be2f9d90333205f9d4b11bd234aa513bf16c53c503f6575718",
    "minerinfo": "",
    "nextblockhash": "0c1a3bddf4686c6be9be8c4be2d3915c542856adac8c7458bd056df83e434eaa",
    "nonce": 0,
    "previousblockhash": "05d6db028304da9c126ea34a2820bfda8a269e2ceb9bfdda73e70632ef13be7f",
    "size": 560,
    "strippedsize": 560,
    "time": 1550721855,
    "tx": [
        {
            "attributes": [
                {
                    "data": "33c33ebc08ea3c1a",
                    "usage": 0
                }
            ],
            "blockhash": "1c1e1c22ce891184d390def30a9b8f15f355c05a7bd6e7e7912b571141e01415",
            "blocktime": 1550721855,
            "confirmations": 296,
            "hash": "8570ff4cd3d356be2f9d90333205f9d4b11bd234aa513bf16c53c503f6575718",
            "locktime": 100,
            "payload": {
                "CoinbaseData": ""
            },
            "payloadversion": 4,
            "programs": [],
            "size": 254,
            "time": 1550721855,
            "txid": "8570ff4cd3d356be2f9d90333205f9d4b11bd234aa513bf16c53c503f6575718",
            "type": 0,
            "version": 0,
            "vin": [
                {
                    "sequence": 4294967295,
                    "txid": "0000000000000000000000000000000000000000000000000000000000000000",
                    "vout": 65535
                }
            ],
            "vout": [
                {
                    "address": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",
                    "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                    "n": 0,
                    "outputlock": 0,
                    "payload": null,
                    "type": 0,
                    "value": "1.50684931"
                },
                {
                    "address": "EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR",
                    "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                    "n": 1,
                    "outputlock": 0,
                    "payload": null,
                    "type": 0,
                    "value": "1.75799086"
                },
                {
                    "address": "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta",
                    "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                    "n": 2,
                    "outputlock": 0,
                    "payload": null,
                    "type": 0,
                    "value": "1.75799088"
                }
            ],
            "vsize": 254
        }
    ],
    "version": 0,
    "versionhex": "00000000",
    "weight": 2240
}
```

### 3.8 Get Raw Transaction

```
./ela-cli info getrawtransaction 17296308c322aee00274da494e0b9a08423b65d170bd2235c3b658f7030fd9b9
```

Result：

```
"0902000100133233313431363030303939333439323932373506f9a9deeaf33bde5646567b129c8da3fee08db6210a8d17f359caf6e3b353bf320100ffffffff01ac3c65375d4f0f882a7b76639e3408c1b1fa39731ace92a352e1b650ce35a20100ffffffffe85c7e34e11fd8e1257333d509fc1828f8feef1120b0822940be9c88ee58c31d0100ffffffffdce53502041b2a22a354ffbf03d1c0aed9ff44f731aec4d3a2376c84cbc5696b0100ffffffff8234bf50743e34707a48d9c0cc31ccef29c6f3013b44170ee73835099e3574c60100ffffffffcb1210fe4fbd42cd71c4a5ce46a7862e3e4e557cc5ee6c1d3189208525ecf6de0100ffffffff02b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a300ca9a3b0000000000000000125bc115b91913c9c6347f0e0e3ba3b75c80b9481100b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3d4d634030000000000000000210d4109bf00e6d782db40ab183491c03cf4d6a37a00000000000141403c72d894b0138348a7640f689b0c4003f1a91969e1b1a1f767303f0fda8226fee42f4b15ac650a8df31a68000fc979036a29dec4383be0571f7bf1bcf3c1cd842321034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16cac"
```

### 3.9 Get Transaction Pool Information

```
./ela-cli info getrawmempool
```

Result:

```
[
    {
        "attributes": [
            {
                "data": "35343036373034363031353838313931393731",
                "usage": 0
            }
        ],
        "blockhash": "",
        "blocktime": 0,
        "confirmations": 0,
        "hash": "acbb3c92e36db7d11e81a16e478943edf5cfb5bc7437e7319d3348fd4c7cb2bf",
        "locktime": 0,
        "payload": null,
        "payloadversion": 0,
        "programs": [
            {
                "code": "21034f3a7d2f33ac7f4e30876080d359ce5f314c9eabddbaaca637676377f655e16cac",
                "parameter": "40ad12383bf739c2f38da527f90b93b82faf9cef103fc8ffed25e79769eb03a7d47c6146f5795aa09636ca855295d9dd016b7c34d2a6804f7e0c13564693ba5872"
            }
        ],
        "size": 494,
        "time": 0,
        "txid": "acbb3c92e36db7d11e81a16e478943edf5cfb5bc7437e7319d3348fd4c7cb2bf",
        "type": 2,
        "version": 9,
        "vin": [
            {
                "sequence": 4294967295,
                "txid": "2575de0d6d323881e7cdb8ede3a13c0ac8a108621be42d66be55860621cfe718",
                "vout": 1
            },
            {
                "sequence": 4294967295,
                "txid": "6c90a60a9e1884211f136de8ebab43a70c7f5c7470490b4031a2426584d08f06",
                "vout": 1
            },
            {
                "sequence": 4294967295,
                "txid": "1162ad86e2f80cfc5507fc018ea03cecdc7f48732d034ac76b6c0b7280fc5f98",
                "vout": 1
            },
            {
                "sequence": 4294967295,
                "txid": "f6c41d3ba2248ee3311aa5680be44e95e295823ac4a6018d76a56de818e481b0",
                "vout": 1
            },
            {
                "sequence": 4294967295,
                "txid": "79fe7592307c44e2794286c2782f1599213eb7fd790f89d39fd8fb0a3e7a40ed",
                "vout": 1
            },
            {
                "sequence": 4294967295,
                "txid": "a9d1030614d73518a1b51555e09d63b38fc8d42cbbb8800abcd0008975144606",
                "vout": 1
            }
        ],
        "vout": [
            {
                "address": "8PT1XBZboe17rq71Xq1CvMEs8HdKmMztcP",
                "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                "n": 0,
                "outputlock": 0,
                "payload": {},
                "type": 0,
                "value": "10"
            },
            {
                "address": "EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR",
                "assetid": "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0",
                "n": 1,
                "outputlock": 0,
                "payload": {},
                "type": 0,
                "value": "0.53794516"
            }
        ],
        "vsize": 494
    }
]
```

### 3.10 List Producer Information

```
./ela-cli info listproducers
```

Result：

```
{
    "producers": [
        {
            "active": true,
            "cancelheight": 0,
            "illegalheight": 0,
            "inactiveheight": 0,
            "index": 0,
            "location": 0,
            "nickname": "PRO-002",
            "nodepublickey": "03340dd02ea014133f927ea0828db685e39d9fdc2b9a1b37d2de5b2533d66ef605",
            "ownerpublickey": "02690e2887ac7bc2c5d2ffdfeef4d1edc060838fee009c26a18557648f9e6f19a9",
            "registerheight": 104,
            "state": "Activate",
            "url": "https://elastos.org",
            "votes": "2"
        },
        {
            "active": true,
            "cancelheight": 0,
            "illegalheight": 0,
            "inactiveheight": 0,
            "index": 1,
            "location": 0,
            "nickname": "PRO-003",
            "nodepublickey": "02b796ff22974f2f2b866e0cce39ff72a417a5c13ceb93f3932f05cc547e4b98e4",
            "ownerpublickey": "036e66b27064da32f333f765a9ae501e7dd418f529d10afa1e4f72bd2a3b2c76a2",
            "registerheight": 110,
            "state": "Activate",
            "url": "https://elastos.org",
            "votes": "1"
        }
    ],
    "totalcounts": 2,
    "totalvotes": "3"
}
```



## 4. Mining

```
NAME:
   ela-cli mine - Toggle cpu mining or manual mine

USAGE:
   ela-cli mine [command options] [args]

DESCRIPTION:
   With ela-cli mine, you can toggle cpu mining or discrete mining.

OPTIONS:
   --toggle value, -t value  use --toggle [start, stop] to toggle cpu mining
   --number value, -n value  user --number [number] to mine the given number of blocks
```

### 4.1 Start CPU Mining

```
./ela-cli mine -t start
```

Result:

```
mining started
```

### 4.2 Stop CPU Mining

```
./ela-cli mine -t stop
```

Result:

```
mining stopped
```

### 4.3 Discrete Mining

-n

The `n` parameter is used to specify the number of blocks to want mine

```
./ela-cli mine -n 1
```

Result:

```
[e9c1d12d5f4e7679737d1d348e53ce20fc6966e156d0a40d10e3fcf39c94c2f2]
```



## 5. Rollback Block

```
NAME:
   ela-cli rollback - Rollback blockchain data

USAGE:
   ela-cli rollback [command options] [args]

DESCRIPTION:
   With ela-cli rollback command, you could rollback blockchain data.

OPTIONS:
   --height value  the final height after rollback (default: 0)
```

The height parameter is used to set the final height after rollback.

```bash
./ela-cli rollback --height 20
```

Result:
```
current height is 22
blockhash before rollback: 74858bcb065e89840f27b28a9ff44757eb904f1a7d135206d83b674b9b68fd4e
blockhash after rollback: 0000000000000000000000000000000000000000000000000000000000000000
current height is 21
blockhash before rollback: 18a38afc7942e4bed7040ed393cb761b84e6da222a1a43df0806968c60fcff8a
blockhash after rollback: 0000000000000000000000000000000000000000000000000000000000000000
```