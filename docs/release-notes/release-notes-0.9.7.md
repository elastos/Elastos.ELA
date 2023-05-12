Elastos.ELA version 0.9.7 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.7/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.6 and before, you should shut it down
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

Fixed an issue with block synchronization getting stuck.

0.9.7 change log
=================

### Bug Fixes

* **bpos:** fixed an issue with block synchronization getting stuck ([86910f4](https://github.com/elastos/Elastos.ELA/commit/86910f42a73be84a83ab5f9f118bd827bed12827))
* **bpos:** modify to show vote rights correctly ([5f8f611](https://github.com/elastos/Elastos.ELA/commit/5f8f6113ca1f14f6b5e009d2f9f8d4868a61ecb1))
* **nft:** fixed an issue with NFT history handling ([145e878](https://github.com/elastos/Elastos.ELA/commit/145e878f84931e753dcbe79dde182e9a30cac121))
