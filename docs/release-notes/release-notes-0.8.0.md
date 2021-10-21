Elastos.ELA version 0.8.0 is now available from:

  <https://download.elastos.org/elastos-ela/elastos-ela-v0.8.0/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.7.0 and before, you should shut it down and wait until
 it has completely closed, then just copy over `ela`(on Linux).

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

Support quick node startup.
Support cross-chain data transfer.
Support automated cross-chain failure handling.
Support small cross-chain transaction transfers quickly.
Support register side-chain proposals.
Prohibit deposit to the old side chain of DID.

0.8.0 change log
=================

### Quick node startup related
- #1803 Remove history from config file
- #1753 Snapshot block when it is in pow mod
- #1751 Change to snapshot checkpoint correctly
- #1749 Fixed an issue that snapshot of checkpoint is empty
- #1745 Modify to use check point for quick start
- #1735 Add default value of params

### Cross-chain data transfer related
- #1796 Add draft data for CRC proposal
- #1795 Add draft data into RegisterSideChain payload
- #1782 Modify to check withdraw transaction correctly
- #1779 Do not check special outputs count of withdrawFromSideChain tx
- #1771 Add withdraw output payload information
- #1746 Add lua to create new cross chain transaction

### Automated cross-chain failure handling related
- #1801 Add ReturnSideChainDepositeCoinFilter
- #1800 Add ReturnCrossChainCoinStartHeight to config
- #1776 Check return deposit transaction amount by output value 
- #1765 Add check of amount in CrossChainOutput payload
- #1744 Modify the check of return side chain deposit transaction
- #1743 Add ReturnSideChainDepositCoin payload
- #1741 modify to check return side chain deposit transaction amount correctly

### Small cross-chain transaction transfers quickly related
- #1807 Store smallcross tx after validation
- #1767 Check if a transaction is small by payload version

### Register side-chain proposals related
- #1816 Modify to add RegisterSideChainInfo correctly
- #1814 Modify to remove register side chain information correctly
- #1813 Store register proposal info when register
- #1811 Add register sidechain proposal mempool conflict check
- #1810 Modify to show register side chain proposal correctly
- #1809 Not support RegisterSideChain transaction before NewCrossChainStartHeight
- #1808 Remove & add fields to register sidechain
- #1804 Add exchangeRate to register sidechain proposal
- #1802 Check matic number and genesis hash existency
- #1768 Add Support of Register sidechain
- #1714 Fixed an issue that randomed DPOS inactive at undesired height

### CustomID proposal related
- #1825 Add LetterOrNumber check for reserved customID
- #1824 ReservedCustomID of tryCancelReservedCustomID used under history
- #1823 Add slotReserveCustomID
- #1822 Reserver customized did can only success once
- #1820 Add EIDEffectiveHeight to RPCChangeCustomIDProposal
- #1812 ExchangeRate need to be 1.0
- #1761 Fixed an issue of serialization of ProposalState
- #1750 Change default value of effective height

### prohibit deposit to old did 
- #1762 prohibit transfer to did function

### RPC
- #1828 Modify to show SideChainInfo correctly
- #1827 Modify to show EIDEffectiveHeight correctly
- #1815 Modify rpc display
- #1742 Add rpc of getexistreturndeposittransactions 

### P2P
- #1817 Modify to get blocks when inventory list count is 500
- #1806 Change default value of MaxNodePerHost
- #1794 Fixed some bugs and Revert commot of dpos network connections limit
- #1792 Fixed an issue of synchronous
- #1775 Limit the number of direct network connections
- #1766 Support to reorganize the chain more than 500 blocks
- #1736 Modify to sync block in POW consensus correctly

### Memory
- #1778 Add inputs count limit for transaction cache