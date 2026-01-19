Elastos.ELA version 0.9.9.4 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.9.4/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.9.3 and before, you should shut it down
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

1. Fixed an issue of revertToDPoS.
2. Update workflows.

0.9.9.4 change log
=================

### Bug Fixes

* **dpos** fixed an issue of creating revert to dpos tx ([5328a5b](https://github.com/elastos/Elastos.ELA/commit/5328a5bf7a9a58073a21cc47378ab2e97a5c7747))
* **dpos** update workflows ([bbe0173](https://github.com/elastos/Elastos.ELA/commit/bbe0173f614ff067876d7a54689a7923e057ebf7))
* **dpos** finish consensus in pow correctly ([7a34544](https://github.com/elastos/Elastos.ELA/commit/7a34544df69023fc464f05ae683b9f8ea60165b7))
