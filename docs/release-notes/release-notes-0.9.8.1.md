Elastos.ELA version 0.9.8.1 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.8.1/>

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

An issue related to submitAuxBlock has been fixed.

0.9.8.1 change log
=================

### Bug Fixes

* prevent record sponsor tx from tx pool ([e1f7ec8](https://github.com/elastos/Elastos.ELA/commit/e1f7ec8b2a0d72a5ba2b36c658795cc737716215))