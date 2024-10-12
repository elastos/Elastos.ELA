Elastos.ELA version 0.9.8 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.8/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.7 and before, you should shut it down
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

The RecordSponsor Transaction has been added to record the sponsor of BPoS consensus proposal.
An issue related to DiscreteMining has been fixed.

0.9.8 change log
=================

### Bug Fixes

* modify to DiscreteMining correctly ([d65d2da](https://github.com/elastos/Elastos.ELA/commit/d65d2da68e044d40c5a06177e248d99f2f848709))

### Features

* add duplicate check for record sponsor tx ([b443c3f](https://github.com/elastos/Elastos.ELA/commit/b443c3f17044e670ca5e29263ac8f3d0bbebb47c))
* add record sponsors transaction ([832f050](https://github.com/elastos/Elastos.ELA/commit/832f050d235ced56fcdc23ac0769027c4b85c550))
* after RecordSponsorStartHeight pow block no need to record ([ec3d1f8](https://github.com/elastos/Elastos.ELA/commit/ec3d1f8aea57e99bf1ea7624f5b8848fd5966c41))
* generate block with record sponsor tx ([48b4479](https://github.com/elastos/Elastos.ELA/commit/48b447995fd40bd5443613f745041fb1bd223348))
* inactive by sponsor from cache and confirm ([2637cfb](https://github.com/elastos/Elastos.ELA/commit/2637cfb6a27338857b4293506c1208dcb56228b0))
* let test cases pass ([cebc312](https://github.com/elastos/Elastos.ELA/commit/cebc312739f632161e03f33998183d94c9ed3325))
* read confirm proposal sponsors from checkpoints first ([fb55de9](https://github.com/elastos/Elastos.ELA/commit/fb55de98b2e2b5eaa4783ad320c0357a8c48067d))
* record all possible rewards ([593fd97](https://github.com/elastos/Elastos.ELA/commit/593fd97675874ff69c0f78713c3e6d743f322d14))
* record sponsor from RecordSponsorStartHeight ([17492f8](https://github.com/elastos/Elastos.ELA/commit/17492f83c06618ccae903d4540d413999a951500))
* set default value of mainnet RecordSponsorStartHeight ([ea429f7](https://github.com/elastos/Elastos.ELA/commit/ea429f7409a41b4fb9e029ee5e00bcdf540f0bf3))
* set testnet RecordSponsorStartHeight ([f2c9fa5](https://github.com/elastos/Elastos.ELA/commit/f2c9fa5e7c753ec2d54b17e3c894fe9b36eb8ea4))
* update attribute check of sponsor tx ([dff4e61](https://github.com/elastos/Elastos.ELA/commit/dff4e617eeb0c66b93afae6397340eaa473acef1))
* update record sponsor tx related payload ([d57c74d](https://github.com/elastos/Elastos.ELA/commit/d57c74d1fd2778f22acf25c1f535235b180c3676))
* use last rewards from RecordSponsorStartHeight ([0adc304](https://github.com/elastos/Elastos.ELA/commit/0adc3047dab3c78f033f477d93bc190f4e2f1af4))
