Elastos.ELA version 0.9.8.2 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.8.2/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.8 and before, you should shut it down
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

An issue related to reorganize has been fixed.

0.9.8.2 change log
=================

### Bug Fixes

* record last dpos rewards by history ([cf2aef4](https://github.com/elastos/Elastos.ELA/commit/cf2aef49048810b2f60b18264582069c91e30aec))
* more than 720 blocks can reorganize under pow ([798658d](https://github.com/elastos/Elastos.ELA/commit/798658d4599347d348d77766f79f3f51826a0ca4))