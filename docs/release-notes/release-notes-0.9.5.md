Elastos.ELA version 0.9.5 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.5/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.4 and before, you should shut it down
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

Fixed an issue of reorganize.

0.9.5 change log
=================

### Bug Fixes

* **dpos2.0:** fix the rollback issue under the POW consensus ([84cc00e](https://github.com/elastos/Elastos.ELA/commit/84cc00e4c7ef16ec9a4c00990805ff149b2f24b8))
* **dpos2.0:** fixed a bug of discrete mining ([7823ee5](https://github.com/elastos/Elastos.ELA/commit/7823ee526bbf10d6577df7281de2eaf9158e90d1))
* **dpos2.0:** fixed an issue of reorganize ([d6e8465](https://github.com/elastos/Elastos.ELA/commit/d6e84656d8d2a9ef45bc93129f4ab3cb62b396ee))


