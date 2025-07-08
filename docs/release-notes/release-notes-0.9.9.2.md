Elastos.ELA version 0.9.9.2 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.9.2/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.9.1 and before, you should shut it down
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

1. Change illegal penalty to zero

0.9.9.2 change log
=================

### Bug Fixes

* **bpos** update ZeroIllegalPenaltyStartHeight mainnet default value ([b59dcbc](https://github.com/elastos/Elastos.ELA/commit/b59dcbc02bc7a2381ecefb1a4ef238d8df9a1fc9))
* **bpos** update ZeroIllegalPenaltyStartHeight testnet default value ([9ef138a](https://github.com/elastos/Elastos.ELA/commit/9ef138a54be77625cf0f9788cf3c7d4b19d291e4))
* **bpos** change illegal penalty to zero ([606ebc4](https://github.com/elastos/Elastos.ELA/commit/606ebc4810563594115ecac6d2dcb6f56069b9ba))
