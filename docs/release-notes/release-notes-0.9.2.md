Elastos.ELA version 0.9.2 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.2/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.1 and before, you should shut it down
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

Correct the proposal votes check issue in ChangeView.

0.9.2 change log
=================

### Bug Fixes

* **nft:** correct the proposal votes check issue in ChangeView ([b8c8c64](https://github.com/elastos/Elastos.ELA/commit/b8c8c643b8013c8363304cc960bea41fd58fe0fa))
