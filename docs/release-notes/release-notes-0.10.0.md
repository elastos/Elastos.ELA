Elastos.ELA version 0.10.0 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.10.0/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.7 and before, you should shut it down
and wait until it has completely closed, then just copy over `ela`(on Linux).

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

Update NFT transaction.
Support for multi-signature registration of BPoS node.
Support for staking with multiple addresses.
Enhance the stability of main chain and side chain.

0.10.0 change log
=================

### Bug Fixes

* **bpos:** fixed an issue of discrete mining ([299781e](https://github.com/elastos/Elastos.ELA/commit/299781e61c4daeffaacfd43353feb373827f07f2))
* **bpos:** fixed an issue of real withdraw transaction ([6bc854d](https://github.com/elastos/Elastos.ELA/commit/6bc854dfb5e9d2680c8293061578bfa09969a68e))
* **bpos:** modify to create new version NextTrunDPoSInfo tx correctly ([8c50cf8](https://github.com/elastos/Elastos.ELA/commit/8c50cf89002a6ed94bf25c8c3cf4b67ec4586d5b))
* **bpos:** modity to deserialize nextTurnDPoSInfo payload correctly ([d14512e](https://github.com/elastos/Elastos.ELA/commit/d14512e2add518030076064be228d80228619529))
* **cli:** update ela version information ([bfeb412](https://github.com/elastos/Elastos.ELA/commit/bfeb4120defed420f85841ba1cae0ecdda87ce10))
* **dpos2.0:** change revertToDPoS script attribute to nonce ([c72e123](https://github.com/elastos/Elastos.ELA/commit/c72e123cc02dd2d70f504b14fcd6b7132d7cf0a4))
* **dpos:** last irreversible height is not allowed to rollback ([e2bbc56](https://github.com/elastos/Elastos.ELA/commit/e2bbc563d78451d8bc9a961f4ec1bcf25e7c80c7))
* **dpos:** modify to recover view offset correctly ([31f6522](https://github.com/elastos/Elastos.ELA/commit/31f652289900c763028954760890eeb6075e93ab))
* **nft:** change CreateNFTTransaction err msg ([d11565a](https://github.com/elastos/Elastos.ELA/commit/d11565a5f40d641a6209f1c24c210b7f2cf0aa26))
* **nft:** change NFTID to ReferKey ([2b70cd7](https://github.com/elastos/Elastos.ELA/commit/2b70cd713ce3f1f73f4a751b962face875fc7fe0))
* **nft:** change ownerkey to hash function ([4188d68](https://github.com/elastos/Elastos.ELA/commit/4188d68c02e390880517535630ef5cdbd78cadd3))
* **nft:** end height need to be equal to LockTime ([f40482b](https://github.com/elastos/Elastos.ELA/commit/f40482b540c96cd4fe05181421422a11fd1b6008))
* **nft:** modify to show NFT transaction correctly ([df8bc65](https://github.com/elastos/Elastos.ELA/commit/df8bc655204e78179303241012d438008ec6ba5c))
* **nft:** more than 720 blocks can reorganize under pow ([ecf5b59](https://github.com/elastos/Elastos.ELA/commit/ecf5b595f68d4721fa1fee723e88c3183e77b54b))
* **nft:** N of multisign register producer tx can not over 10 ([897307d](https://github.com/elastos/Elastos.ELA/commit/897307df69183107312844f9fd9090f07d996da6))


### Features

* **bpos:** add MultiExchangeVotesStartHeight ([653cd1f](https://github.com/elastos/Elastos.ELA/commit/653cd1fa8775b5e18c3e24d243c01c2f73fb6f87))
* **bpos:** change arbiters count from float64 to int ([be66579](https://github.com/elastos/Elastos.ELA/commit/be665791dab110810d167e9f601c9f9db054009f))
* **bpos:** modify to save checkpoints file correctly ([b9e2cec](https://github.com/elastos/Elastos.ELA/commit/b9e2cec537e1c352b8bf898d98eca3f927ec9ab0))
* **bpos:** remove exception data processing logic ([164655b](https://github.com/elastos/Elastos.ELA/commit/164655be7b28aeb3829b46dffdc267e927b4d069))
* **bpos:** set default value of config for testnet ([f3e66e6](https://github.com/elastos/Elastos.ELA/commit/f3e66e6dde92a756d65196260dc38f3d815d56f8))
* **checkpoints:** adjust logic of checkpoints ([455f0ea](https://github.com/elastos/Elastos.ELA/commit/455f0ea18692adf5ee473c91866872001f656835))
* **checkpoints:** change the start height of DPoS checkpoints to be same as CR ([7def4a2](https://github.com/elastos/Elastos.ELA/commit/7def4a2a3b664e958d9f53493e2bd3e1c1f5da2c))
* **dex:** add NextTurnDPOSPayloadInfoV2 ([e625c9d](https://github.com/elastos/Elastos.ELA/commit/e625c9d18fd6557b72cbf628ed658f6dce91d014))
* **dex:** modify NextTurn transaction for dex ([55eac2b](https://github.com/elastos/Elastos.ELA/commit/55eac2b35e46ce434d5d17ecf451c780024d6d2d))
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
* **nft:** add deailed votes infromation into NFT tx payload ([e904fc4](https://github.com/elastos/Elastos.ELA/commit/e904fc40a4bf558c005a351d949568a218d902c0))
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
