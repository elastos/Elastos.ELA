Elastos.ELA version 0.8.3 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.8.3/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.8.2 and before, you should shut it down and wait until
 it has completely closed, then just copy over `ela`(on Linux).

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

Temporary modifications to ensure the stability of side chain and cross-chain.

0.8.3 change log
=================

### Fix bug
- #1903 compatible with test network data
- #1901 resolve cross-chain and side chain over-dependence on DPoS nodes