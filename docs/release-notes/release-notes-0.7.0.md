Elastos.ELA version 0.7.0 is now available from:

  <https://download.elastos.io/elastos-ela/elastos-ela-v0.7.0/>

This is a new major version release.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.6.0 and before, you should shut it down and wait until
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

Refactor the supply of ELA.
Removed the fixed 12 CR DPoS node.
Support CR proposal draft data.
Support random DPoS node.
Support DPoS and CR penalty.
Support revert to POW.
Support reserve custom ID.

0.7.0 change log
=================

### Refactor ELA supply related
- #1735 Add default value of params
- #1732 Add HavlingRewardInterval
- #1726 Modify to show canceled producers correctly
- #1720 Refactor the supply of ELA

### Proposal draft data related
- #1697 Rename getdraftdatabydrafthash to getproposaldraftdata
- #1680 Proposal draft data stored into db
- #1669 Add GetDraftDataByDraftHash rpc
- #1668 Fixed an issue of checkCRCProposalTrackingTransaction
- #1644 Add secretary-general opinion data
- #1637 Add opinion data and message data to proposal review and tracking tx
- #1628 Add draft data for proposal

### Penalty Related
- #1716 Set 'selected' to false when producer change to inactive
- #1700 Modify to update inactive count correctly
- #1692 Add inactive penalty
- #1690 Change arbiter inactive count and illegal behavior only punish once
- #1688 Allow to activate CR member by DPOSNodePublicKey
- #1686 Make rpc IP configurable
- #1675 Fixed an issue that would change the status to inactive by mistake
- #1673 Producer illegal behavior punish done
- #1653 Modify to count inactive blocks from the first one
- #1652 Reset the inactiveCountingHeight of first onduty arbiter
- #1646 CR member inactive does not deduct deposit coin
- #1640 Use InactivePenalty after ChangeCommitteeNewCRHeight
- #1639 Move IllegalPenalty into DPoSConfiguration
- #1638 Illegal behavior add penalty

### Revert to Pow related
- #1727 Modify to return deposit coin correctly
- #1717 Fixed distributeWithNormalArbitratorsV2 logic errors
- #1715 Create revertToPOW transaction at next turn
- #1710 Fixed an issue that failed to rollback blocks in POW
- #1706 Destroy reward of CyberRepublic in POW consensus algorithm
- #1705 Fixed an issue where NextTurnInfo transaction could not be created
- #1704 Modify to tryUpdateInactivity correctly
- #1703 StopConfirmBlockTime only take effect after RevertToPOWStartHeight
- #1702 Reset inactive count when height and lastUpdateInactiveHeight
- #1701 Modify tryUpdateInactivityV0
- #1698 Support activate ActiveProducer before ChangeCommitteeNewCRHeight
- #1694 Move lastIrreversibleHeight to dpos state for rollback and dump peer
- #1693 Change curBlockHeight of RevertToDPOS into RevertToPOWBlockHeight
- #1679 Fixed some bugs of revert to pow
- #1678 Add revert to POW and revert to DPOS transaction to filter
- #1670 Modify to create RevertToPOW transaction correctly
- #1667 There is no irreversible height when we are in pow consensus algorithm
- #1666 Change RevertToPOWNoBlockTime from time.duration to int64
- #1665 Fixed an issue of time duration
- #1664 Add RevertToDPOS transaction
- #1661 Add RevertToPOW transaction

### Random DPoS node related
- #1714 Fixed an issue that randomed DPOS inactive at undesired height
- #1713 Update inactive count of CR memebr correctly
- #1712 Modify to inactive random DPoS node correctly
- #1685 Fixed some issues of force change
- #1684 Fixed an issue that would change the status to inactive by mistake
- #1682 Replace proposal with proposalInfo from memory
- #1681 Modify to get random DPOS node correctly
- #1677 Modify to get onduty arbitrators correctly
- #1672 Modify to getArbiterPeersInfo correctly
- #1671 Fixed an error of minInt
- #1663 Add default value of DPOSNodeCrossChainHeight
- #1662 Random dpos node merge
- #1660 Fixed an issue of resetNextArbiterByCRC
- #1659 Modify to get cross chain arbiters correctly
- #1658 Add proposalType into ProposalResult
- #1656 Fixed some issues of distributeWithNormalArbitratorsV3
- #1651 Modify to create next turn dpos info transaction correctly
- #1649 Fixed an issue of temporary change of CR members
- #1648 Fixed an issue of snapshotVotesStates
- #1647 Add log to show the error of checkTxHeightVersion
- #1643 Random dpos node
- #1642 Fixed an issue of NeedNextTurnDposInfo
- #1636 Get a candidate as DPOS node at random
- #1635 Add wait time to create CRAssetsRectify transaction
- #1633 Modify reward CRC dpos producer also can have vote reward
- #1632 Modify create nextturn fetch dpos producer from unclaimed index
- #1630 Add unit test of dpos reward
- #1629 Add unit test of consensus change to 36 producer
- #1627 Adjust consensus and reward of DPOS node after ChangeCommitteeNewCRHeight
- #1626 Using dpos producer node to generate block when cr is not been claimed

### Custom ID related
- #1709 Modify proposal type
- #1707 ReserveCustomID,ReceiveCustomID, ChangeCustomIDFee test finished
- #1699 Move MaxReservedCustomIDListCount to Configuration
- #1655 Add customidresult transaction
- #1650 Add ChangeCustomIDFee transaction
- #1645 Add receive reserved custom id proposal
- #1641 Add reserved did short name proposal
