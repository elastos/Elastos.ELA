Elastos.ELA version 0.9.9 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.9/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.8.2 and before, you should shut it down
and wait until it has completely closed, then just copy over `ela`(on Linux) 
and obtain the sponsors file to place it in the working directory.

However, as usual, config, keystore and chaindata files are compatible.

Compatibility
==============

Elastos.ELA is supported and extensively tested on operating systems
using the Linux kernel. It is not recommended to use Elastos.ELA on
unsupported systems.

Elastos.ELA should also work on most other Unix-like systems but is not
as frequently tested on them.

As previously-supported CPU platforms, this release's pre-compiled
distribution provides binaries for the x86_64 platform.

Notable changes
===============

1. Modify the view reset recovery mechanism in DPOS consensus to a view catching-up mechanism.
2. Open evidence transactions for proposal misconduct and voting misconduct.
3. Fix and optimize the fork reorganization issues under POW and DPOS.

0.9.9 change log
=================

### Bug Fixes

* **bpos:** fixed an issue of real withdraw transaction ([6bc854d](https://github.com/elastos/Elastos.ELA/commit/6bc854dfb5e9d2680c8293061578bfa09969a68e))
* **bpos:** modify to create new version NextTrunDPoSInfo tx correctly ([8c50cf8](https://github.com/elastos/Elastos.ELA/commit/8c50cf89002a6ed94bf25c8c3cf4b67ec4586d5b))
* **bpos:** modity to deserialize nextTurnDPoSInfo payload correctly ([d14512e](https://github.com/elastos/Elastos.ELA/commit/d14512e2add518030076064be228d80228619529))
* **cli:** update ela version information ([bfeb412](https://github.com/elastos/Elastos.ELA/commit/bfeb4120defed420f85841ba1cae0ecdda87ce10))
* **dpos2.0:** change revertToDPoS script attribute to nonce ([c72e123](https://github.com/elastos/Elastos.ELA/commit/c72e123cc02dd2d70f504b14fcd6b7132d7cf0a4))
* **dpos:** change view when offset different ([92e07e6](https://github.com/elastos/Elastos.ELA/commit/92e07e6aa32870162b9ade553dc29d6b61cc2c12))
* **dpos:** fixed an issue of CheckBlockContext ([6d62f4b](https://github.com/elastos/Elastos.ELA/commit/6d62f4b1514f3f672cbe2ec6b51bd6cacbd9c50a))
* **dpos:** get block with PID ([7ddf3e3](https://github.com/elastos/Elastos.ELA/commit/7ddf3e32e46118729194ff52dc8f6812818d184e))
* **dpos:** last irreversible height is not allowed to rollback ([e2bbc56](https://github.com/elastos/Elastos.ELA/commit/e2bbc563d78451d8bc9a961f4ec1bcf25e7c80c7))
* **dpos:** modify to recover view offset correctly ([31f6522](https://github.com/elastos/Elastos.ELA/commit/31f652289900c763028954760890eeb6075e93ab))
* **dpos:** not reset LastArbiters ([3b37403](https://github.com/elastos/Elastos.ELA/commit/3b37403ed5609afa14e30d36317d5f6aaf92f45f))
* **dpos:** record last dpos rewards by history ([cf2aef4](https://github.com/elastos/Elastos.ELA/commit/cf2aef49048810b2f60b18264582069c91e30aec))
* **dpos:** remove history by real changes ([bbf89ab](https://github.com/elastos/Elastos.ELA/commit/bbf89ab7f5e04981bb644702eb723a5210ce8c6d))
* **dpos:** reocrd last dpos rewards by history ([f300d17](https://github.com/elastos/Elastos.ELA/commit/f300d17bf9c1208a7798b36066b9b7a04fab1abc))
* **nft:** change CreateNFTTransaction err msg ([d11565a](https://github.com/elastos/Elastos.ELA/commit/d11565a5f40d641a6209f1c24c210b7f2cf0aa26))
* **nft:** change NFTID to ReferKey ([2b70cd7](https://github.com/elastos/Elastos.ELA/commit/2b70cd713ce3f1f73f4a751b962face875fc7fe0))
* **nft:** change ownerkey to hash function ([4188d68](https://github.com/elastos/Elastos.ELA/commit/4188d68c02e390880517535630ef5cdbd78cadd3))
* **nft:** end height need to be equal to LockTime ([f40482b](https://github.com/elastos/Elastos.ELA/commit/f40482b540c96cd4fe05181421422a11fd1b6008))
* **nft:** modify to show NFT transaction correctly ([df8bc65](https://github.com/elastos/Elastos.ELA/commit/df8bc655204e78179303241012d438008ec6ba5c))
* **nft:** more than 720 blocks can reorganize under pow ([798658d](https://github.com/elastos/Elastos.ELA/commit/798658d4599347d348d77766f79f3f51826a0ca4))
* **nft:** more than 720 blocks can reorganize under pow ([9954c12](https://github.com/elastos/Elastos.ELA/commit/9954c12adacb251ee6f1ba72152c4d3f4f412dfb))
* **nft:** N of multisign register producer tx can not over 10 ([897307d](https://github.com/elastos/Elastos.ELA/commit/897307df69183107312844f9fd9090f07d996da6))
* **pow:** prevent record sponsor tx from tx pool ([e1f7ec8](https://github.com/elastos/Elastos.ELA/commit/e1f7ec8b2a0d72a5ba2b36c658795cc737716215))
* **test:** let tests pass ([64ed9d5](https://github.com/elastos/Elastos.ELA/commit/64ed9d5720cc6887edddd9c68998161f6714e60f))


### Features

* **bpos:** add default view offset ([47041ac](https://github.com/elastos/Elastos.ELA/commit/47041ac31e981bac31501bae415d154afec8a7cd))
* **bpos:** add MultiExchangeVotesStartHeight ([653cd1f](https://github.com/elastos/Elastos.ELA/commit/653cd1fa8775b5e18c3e24d243c01c2f73fb6f87))
* **bpos:** adjust change view logic and revert to pow logic ([be9dfa5](https://github.com/elastos/Elastos.ELA/commit/be9dfa5853de800a763dd6fde5a5915f98a2ed80))
* **bpos:** adjust the judgment rules of IllegalProposal and IllegalVotes ([43987df](https://github.com/elastos/Elastos.ELA/commit/43987dfbbb6613f89ddde9ce51c550ed489e69d3))
* **bpos:** calculate offset by arbiters count ([71bfc5b](https://github.com/elastos/Elastos.ELA/commit/71bfc5ba4c6b04d3a04394d022a5922583e7a074))
* **bpos:** change arbiters count from float64 to int ([be66579](https://github.com/elastos/Elastos.ELA/commit/be665791dab110810d167e9f601c9f9db054009f))
* **bpos:** change the rule of view offset ([04d5814](https://github.com/elastos/Elastos.ELA/commit/04d5814361b8a18bf9b2b5d66705f86dd93ca398))
* **bpos:** create IllegalProposal and IllegalVote evidence after ChangeViewV1Height ([1997779](https://github.com/elastos/Elastos.ELA/commit/1997779cc29d5484b13ea5f2a6340b52b58f1530))
* **bpos:** instead illegalV2Height with ChangeViewV1Height ([aee7071](https://github.com/elastos/Elastos.ELA/commit/aee7071f9cc3d331db9f38e35c072c745b42f4bf))
* **bpos:** instead RevertToPOWV1Height with new change view height ([3a1935e](https://github.com/elastos/Elastos.ELA/commit/3a1935e7739f7051470d642afbbd2386ba408b82))
* **bpos:** modify to save checkpoints file correctly ([b9e2cec](https://github.com/elastos/Elastos.ELA/commit/b9e2cec537e1c352b8bf898d98eca3f927ec9ab0))
* **bpos:** remove exception data processing logic ([164655b](https://github.com/elastos/Elastos.ELA/commit/164655be7b28aeb3829b46dffdc267e927b4d069))
* **bpos:** remove recover consensus logic ([642f4c7](https://github.com/elastos/Elastos.ELA/commit/642f4c72bf1f775d712ae7a6a1ed1fd12b1a9e2d))
* **bpos:** remove recover view offset logic after ChangeViewV1Height ([a840a84](https://github.com/elastos/Elastos.ELA/commit/a840a84a7e1f95cb330925c80c1f9cf9f1a0697e))
* **bpos:** vote proposal after change view ([dd09828](https://github.com/elastos/Elastos.ELA/commit/dd09828db0f03252620684552f5cebd0469bbf76))
* **checkpoints:** adjust logic of checkpoints ([455f0ea](https://github.com/elastos/Elastos.ELA/commit/455f0ea18692adf5ee473c91866872001f656835))
* **checkpoints:** change the start height of DPoS checkpoints to be same as CR ([7def4a2](https://github.com/elastos/Elastos.ELA/commit/7def4a2a3b664e958d9f53493e2bd3e1c1f5da2c))
* **dex:** add NextTurnDPOSPayloadInfoV2 ([e625c9d](https://github.com/elastos/Elastos.ELA/commit/e625c9d18fd6557b72cbf628ed658f6dce91d014))
* **dex:** modify NextTurn transaction for dex ([55eac2b](https://github.com/elastos/Elastos.ELA/commit/55eac2b35e46ce434d5d17ecf451c780024d6d2d))
* **dpos:** add change view related log ([b3130ad](https://github.com/elastos/Elastos.ELA/commit/b3130ad89fab5bc41d1834ca3fe4cc5b9773455a))
* **dpos:** allow illegla evidences into txpool ([e741ce1](https://github.com/elastos/Elastos.ELA/commit/e741ce1ed699443722561145aa110e2baa849f44))
* **dpos:** generate block with sponsor tx ([f8f236a](https://github.com/elastos/Elastos.ELA/commit/f8f236a83450bfaa09df0859a7a6b513cef974b5))
* **dpos:** get block from block cache first ([01563e0](https://github.com/elastos/Elastos.ELA/commit/01563e0e9d001863db6a17252a7b8835b51fc79e))
* **dpos:** record consensus current height ([ee22aca](https://github.com/elastos/Elastos.ELA/commit/ee22acabc00d6d505f15d9a484943453d7300dbb))
* **dpos:** remove height version check of illegal proposal ([69870ad](https://github.com/elastos/Elastos.ELA/commit/69870ad92cf0a56ea4e7b92ce62283eb320db829))
* **dpos:** remove reset view logic ([e50f53e](https://github.com/elastos/Elastos.ELA/commit/e50f53e49d7fe96af292cef23bcaadf760e8f67b))
* **dpos:** set default value for testnet ([5778cfd](https://github.com/elastos/Elastos.ELA/commit/5778cfdbe65c0477169d606c98737a261daa4165))
* **dpos:** set value of ChangeViewV1Height ([b54bf13](https://github.com/elastos/Elastos.ELA/commit/b54bf1394a25f691dc1f6f2c79e20527746e4f79))
* **dpos:** update change view duration ([d62182e](https://github.com/elastos/Elastos.ELA/commit/d62182e2aa660688c329ea9d2603f7ca466f8616))
* **dpos:** update dependency ([e17507c](https://github.com/elastos/Elastos.ELA/commit/e17507c7cb1640055ce6ec89bb0e3fec70f798be))
* **dpos:** update illegal votes logic ([22731b6](https://github.com/elastos/Elastos.ELA/commit/22731b6c12aa34f680bb126a180457c9bdac8e71))
* **dpos:** update max tx per block ([2471e4d](https://github.com/elastos/Elastos.ELA/commit/2471e4d369fe8f9dd48f11bc4c569a8998e61697))
* **dpos:** update process proposal logic ([d92df7c](https://github.com/elastos/Elastos.ELA/commit/d92df7c0b6e5041bf5df7e35cc9599d76322a08b))
* **go:** update go version to 1.20 ([0689eb6](https://github.com/elastos/Elastos.ELA/commit/0689eb6e5253499c5b9c869b7df937b7f58997a2))
* **multi:** add register_cr_multi_tx.lua ([61db42c](https://github.com/elastos/Elastos.ELA/commit/61db42cb2b6beef82d1b90a0db2faeb1a9c6cf21))
* **multi:** add unregister_cr_multi_tx.lua ([5f5fa41](https://github.com/elastos/Elastos.ELA/commit/5f5fa413a3297d7c3f26b3517abc110aece11846))
* **multi:** add update_cr_multi_tx.lua ([0a9f178](https://github.com/elastos/Elastos.ELA/commit/0a9f178488b0ee660f6cbb6eab5fa67f89c9bc31))
* **multiaddr:** before VotesSchnorrStartHeight every ExchangeVotesTransaction  program can not be schnorr ([06d6b1c](https://github.com/elastos/Elastos.ELA/commit/06d6b1ce05224f797fa2e36b622d8b6c76db2b55))
* **multiaddr:** exchange vote add multi programs ([f38e6f2](https://github.com/elastos/Elastos.ELA/commit/f38e6f298c67b032f75b09ce255a1a9577e683e2))
* **multiaddr:** ExchangeVotesTransaction  len(t.Programs()) must be 1 in CheckOutputSingleInput ([5d836c2](https://github.com/elastos/Elastos.ELA/commit/5d836c28898ff75ebea416be77682fbf1a64c082))
* **multiaddr:** support multi-address exchange votes transaction ([dd9bf87](https://github.com/elastos/Elastos.ELA/commit/dd9bf87cec0834c79968b8005b9031e6b813dd1d))
* **multi:** modify to create multi-sign register CR tx correctly ([e3fc737](https://github.com/elastos/Elastos.ELA/commit/e3fc7379435c2323e68bbcb0261b3f77bcf2d5ad))
* **multi:** register CR tx with multi sign payload is not allowed before NFTStartHeight ([00cc9bd](https://github.com/elastos/Elastos.ELA/commit/00cc9bdc1c016ba817f5e3031f1960ac1c65fb51))
* **multi:** support the creation of CR related multi-sign transactions ([3e18862](https://github.com/elastos/Elastos.ELA/commit/3e18862d4f1ab8121b2c6353d3a1f80a67299439))
* **nft:** add deailed votes information into NFT tx payload ([e904fc4](https://github.com/elastos/Elastos.ELA/commit/e904fc40a4bf558c005a351d949568a218d902c0))
* **nft:** add genesis block hash to createNFT tx payload ([5ad5e3e](https://github.com/elastos/Elastos.ELA/commit/5ad5e3ef62e21cc2b0c9cdd6609941baaaa2f85c))
* **nft:** after NFTV2StartHeight create NFT tx with payload version 0 is invalid ([fad35aa](https://github.com/elastos/Elastos.ELA/commit/fad35aabe17e0df39cf747fa1bb50c897c7aee5b))
* **nft:** change deposit ([761ab1d](https://github.com/elastos/Elastos.ELA/commit/761ab1d384b1c897b020334a4af5542a1bb97aae))
* **nft:** check if the NFT of DestroyNFT transaction exist ([e68b9ad](https://github.com/elastos/Elastos.ELA/commit/e68b9ad37d99ff9ce655ca0c54ee05465f8cab20))
* **nft:** modify to check and process votes rights correctly ([625fd15](https://github.com/elastos/Elastos.ELA/commit/625fd1559d94c5fdb994818a62f738c2716a75d1))
* **nft:** modify to check stake address of CreateNFT transaction correctly ([379e9f4](https://github.com/elastos/Elastos.ELA/commit/379e9f40a8119f9388d06c5f1323256928ce62af))
* **nft:** modify to show CreateNFT transaction payload correctly ([33f1c70](https://github.com/elastos/Elastos.ELA/commit/33f1c70e5c8ea3e70ce9fcbce6fa355900694cd9))
* **nft:** producer tx  multisign finish test finish ([20cb384](https://github.com/elastos/Elastos.ELA/commit/20cb384e865058c6ec5ebf1fa6a2331968be649d))
* **nft:** producer tx lua add multicode support ([ae4b0d3](https://github.com/elastos/Elastos.ELA/commit/ae4b0d3884ab3e6dbd0c8a1796389cc7ca8c9c4b))
* **nft:** recored the relationship between NFT ID and genesis block hash ([16bff61](https://github.com/elastos/Elastos.ELA/commit/16bff6154ca742bd1b3cfc8acceb09893878efda))
* **nft:** remove the relationship between id and genesis block hash when NFT destroyed ([82dfad1](https://github.com/elastos/Elastos.ELA/commit/82dfad141bdc53443ffdc19faf57d9bd0413413e))
* **nft:** rename from ownerPublickey to OwnerKey ([c0399e1](https://github.com/elastos/Elastos.ELA/commit/c0399e1bf2856334412552a5f5653efc6eb19ce8))
* **nft:** rpc return continue use OwnerPublicKey for exchange ([0163246](https://github.com/elastos/Elastos.ELA/commit/0163246eee919a2243b03022245b62904a20dd73))
* **nft:** update create_nft.lua ([124f91e](https://github.com/elastos/Elastos.ELA/commit/124f91eb75af2aa5239e001d245eb452fda5b420))
* **schnorr:** remove activate producer schnorr test ([8ade256](https://github.com/elastos/Elastos.ELA/commit/8ade25611208f8ff0aa3aa9ad727a4dd0d0a039f))
* **schnorr:** remove schnorr payload version from activate tx ([202a84b](https://github.com/elastos/Elastos.ELA/commit/202a84bf4aaa64add2279d776622c8c27c21de7b))
