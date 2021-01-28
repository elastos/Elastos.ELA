Elastos.ELA version 0.6.0 is now available from:

  <https://download.elastos.org/elastos-ela/elastos-ela-v0.6.0/>

This is a new major version release, New proposals are supported and cr can claim 
their own node and run cr node on their own server. 

Please report bugs using the issue tracker at GitHub:

  <https://github.com/elastos/Elastos.ELA/issues>

How to Upgrade
==============

If you are running version release_v0.5.1 and before, you should shut it down and wait until
 it has completely closed, then just copy over `ela` (on Linux).

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

Support new CR proposal.
Support CR Council Member claim DPOS node.

0.6.0 change log
=================

### Proposal Related
- #1487 Add replacement proposal leader
- #1504 Add change secretary general transaction and refactor three new proposal transaction codes 
- #1505 Modify old crc proposal withdraw tx lua script 
- #1508 Adjust the structure of the code associated with the CR proposal 
- #1509 Modify calculate unused budget logic of close proposal 
- #1513 Change to no longer check stage of proposal tracking transaction 
- #1514 Replace PreviousHash with TargetProposalHash 
- #1517 Change CR Assets Address to the correct field name 
- #1519 Change Secretary general proposal tx adjust signature order 
- #1522 Fix some issues of new CRCProposal transaction 
- #1524 Fixed repeated newOwnerPublicKey issus 
- #1525 Adjust CR Proposal related code logic 
- #1526 Add crc_chg_secretary_general.lua 
- #1527 Modify the signature rules of CRCProposal payload 
- #1528 Make CR and dpos producer can withdraw deposit multiple times 
- #1531 Lua add new_owner_private_key 
- #1533 Change the order of NewRecipient and TargetProposalHash 
- #1534 Make cr member state inactive when it's dpos node is inactive 
- #1560 Modify to get CRC proposal information correctly 
- #1561 Add slotCRCSecretaryGeneral to avoid tx duplication

### Claim DPOS node Related
- #1493 Support to create CR assets rectify transaction
- #1497 Make cr rectify transaction start height configurable 
- #1501 Add fee to rectify transaction 
- #1502 Add new version of CRProposalWithdraw transaction 
- #1503 Add arc proposal real withdraw transaction 
- #1506 Modify to deal with error of FetchTx correctly 
- #1507 Add CR claims DPOS node transactions 
- #1510 Adjust the transaction validation logic 
- #1511 Uniform naming is consistent with white paper 
- #1520 Tiny fix with cmd flag 
- #1532 Reset next arbiters according to the claim result 
- #1535 Realize DPOS node switching and reward distribution 
- #1536 Add lua for CR claims DPOS node 
- #1537 Add NextTurnDPOSInfo transaction 
- #1538 Check M and N in withdraw from side chain transaction code 
- #1539 Add nextturndposfilter bloom filter 
- #1544 Fix issues of cr_new_proposal branch 
- #1545 Fixed Lua parameter passing issue 
- #1546 Fixing serialization errors and remove dpos management type 
- #1548 NextTurnDPOSInfo transaction payload hash Exclude duplicates 
- #1549 Fixed slotCRManagementDID call error 
- #1553 Add check CR claim DPOS Management Signature 
- #1558 GetOnDutyArbitrator function add lock 
- #1562 Modify to support activating CR member
- #1563 Add cr nodePublicKey and ownerPublicKey to DPOS nodeOwnerPubKey map
- #1566 Fix issues of CR memebr dpos reward
- #1568 Tiny fix of secretary general mpool conflict 
- #1570 Check reject tracking transaction by height version
- #1572 Modify to let block count of each DPOS round remains the same
- #1573 Add height restriction before crc normal check 
- #1574 Modify CRProposalVersion from 30000 to 80000 
- #1575 Fix Serialize and Deserialize of cr member 
- #1576 Modify check of nextturndposinfo tx when nextcrcarbiter is not elected 
- #1577 Reverse opinion hash and message hash 
- #1578 Fixed an issue that can lead to errors in the first round of consensus 
- #1579 Fixed an issue that failed to deserialize crcArbiter 
- #1580 Start checking the status of CR member from CRClaimDPOSNodeStartHeight 
- #1581 Withdraw amount should be bigger than RealWithdrawSingleFee 
- #1583 Set default value of ActivateRequestHeight to MaxUint32 
- #1584 Fix cr member not inactive when maxinactiveround reach 
- #1585 Reset next arbiters by a copy of current CR members 
- #1586 Fixed an issue that the count of impeaded CR members is wrong 
- #1587 Modify to count inactive blocks correctly 
- #1588 CR member will not be set to Inactive in claim dpos node 
- #1590 Fix data contention in block synchronization 
- #1591 Modify to calculate penalty of member correctly 
- #1592 Modify to count InactiveCount correctly 
- #1595 Fixed an issue that count arbitrators inactivity incorrectly 
- #1596 Using c.params.CRMemberCount instead of length of crcarbiter 
- #1597 Add a cross-domain request switch 
- #1598 Change to returned status when the deposit coin is fully taken away 
- #1599 Fixed concurrent map read and map write error in handleEvents 
- #1600 Reset ActivateRequestHeight when activating CR member 
- #1601 Only count inactivity blocks for CR member of MemberElected state 
- #1602 Fixed an issue of GetDepositCoin 
- #1603 Remove initialization of producer amount at CR voting height 
- #1604 Allow to return deposit at any time 
- #1605 GetCRDepositCoin by publickey add depositAmount and totalAmount 
- #1606 Modify to show error code correctly 
- #1607 Add CRCProposalV1Height to config 
- #1608 Modify to clear NodeOwnerKeys not by goroutine 
- #1609 Add prefix of node version 
- #1610 Rename CRDPOSManagement to CRCouncilMemberClaimNode 
- #1611 Cr claim dpos node fix 
- #1612 Reverse transaction hash and proposal hash in the log 
- #1613 Send NextTurnDPOSInfoTx for every turn 
- #1614 Move tryUpdateCRMemberInactivity into committee 
- #1615 Double Block's MaxLength 
- #1617 Set default value of CR claim DPOS node related params

### RPC Related
- #1518 Add RPC interface to get CR related stage 
- #1542 Add getdpospeersinfo rpc interface 
- #1550 Tiny fix of rpc getdpospeersinfo 
- #1551 Add the CRDPosManagement payload to the RPC interface 
- #1555 GetArbitratorGroupByHeight only return crc arbiters 
- #1557 Add rpc interface to get current and next CRC peers information 
- #1564 Add the field depositAmount to the RPC interface
- #1565 Support batches requests
- #1571 Fix RPC display issue and add tx height version check to context check

### P2P Related
- #1523 Add node version to p2p version message 
- #1552 UnclaimedArbiterKeys must be sorted before use 