Elastos.ELA version 0.9.9.3 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.9.3/>

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

1. Fixed an issue where illegal proposals could occur due to occasional low server performance.

0.9.9.3 change log
=================

### Bug Fixes


* **dpos** fixed an issue of illegal proposal ([d674dbe](https://github.com/elastos/Elastos.ELA/commit/d674dbe9ee9faad235a5964e2a02072b6a440602))
