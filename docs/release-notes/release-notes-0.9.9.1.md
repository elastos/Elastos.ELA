Elastos.ELA version 0.9.9.1 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.9.9.1/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.9.9 and before, you should shut it down
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

1. Fixed the potential abnormal voting behavior of arbiters

0.9.9.1 change log
=================

### Bug Fixes

* **docs** Fix typos in docs/cli_user_guide.md ([1037df4](https://github.com/elastos/Elastos.ELA/commit/1037df427d1f507ba4f4ea9f1d9370f8789b73ac))
* **docs** Fix typos in docs/jsonrpc_apis.md ([5fb181e](https://github.com/elastos/Elastos.ELA/commit/5fb181e4d939bcb7e6ca7b118011fa8b49e0606d))
* **dpos** check and change onduty status before new consensus ([cb64be5](https://github.com/elastos/Elastos.ELA/commit/cb64be5e31a098269a3d342a566757d995c38b5f))
* **dpos** remove reject proposal message ([8c81bcb](https://github.com/elastos/Elastos.ELA/commit/8c81bcb0311fe38cf4180ed5c5dfc4fc17ccb7cd))
