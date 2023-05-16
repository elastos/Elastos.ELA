Elastos.ELA version 0.9.6 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.6/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.5 and before, you should shut it down
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

Fixed an issue of revertToDPoS transaction.

0.9.6 change log
=================

### Bug Fixes

* **dpos2.0:** change revertToDPoS script attribute to nonce ([7e4700e](https://github.com/elastos/Elastos.ELA/commit/7e4700e31556502a9c634825b4dd884dc21c53fd))
