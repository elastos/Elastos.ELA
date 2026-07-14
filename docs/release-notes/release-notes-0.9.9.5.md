Elastos.ELA version 0.9.9.5 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.9.5/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.9.4 and before, you should shut it down
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

1. Reset consensus in pow mode.


0.9.9.5 change log
=================

### Bug Fixes


* **dpos** reset consensus in pow ([13cb45f7](https://github.com/elastos/Elastos.ELA/commit/13cb45f7da247650b1b9bcd9fd4b7d76185356f7))
