-- Copyright (c) 2017-2019 The Elastos Foundation
-- Use of this source code is governed by an MIT
-- license that can be found in the LICENSE file.
--
local m = require("api")

local keystore = getWallet()
local password = getPassword()

if keystore == "" then
    keystore = "keystore.dat"
end
if password == "" then
    password = "123"
end

local wallet = client.new(keystore, password, false)

-- account
local addr = wallet:get_address()
local pubkey = wallet:get_publickey()
print("wallet addr:", addr)
print("wallet public key:", pubkey)

-- asset_id
local asset_id = m.get_asset_id()

local fee = getFee()
local proposal_tracking_type = getProposalTrackingType()
local proposal_hash = getProposalHash()
local document_hash = getDocumentHash()
local stage = getStage()
local appropriation = getAppropriation()
local leader_pubkey = getLeaderPubkey()
local leader_privkey = getLeaderPrivkey()
local new_leader_pubkey = getNewLeaderPubkey()
local new_leader_privkey = getNewLeaderPrivkey()
local secretary_general_privkey = getSecretaryGeneralPrivkey()

if fee == 0
then
    fee = 0.1
end
if proposal_hash == "" then
    print("proposal hash is nil, should use --proposalhash to set it.")
    return
end

if document_hash == "" then
    print("document hash is nil, should use --documenthash to set it.")
    return
end

if leader_pubkey == "" then
    print("leader_pubkey is nil, should use --leaderpublickey to set it.")
    return
end

if leader_privkey == "" then
    print("leader_privkey is nil, should use --leaderprivatekey to set it.")
    return
end

if secretary_general_privkey == "" then
    print("secretary general private key is nil, should use --secretarygeneralprivatekey to set it.")
    return
end
print("secretary general privkey:", secretary_general_privkey)

print("fee:", fee)
print("proposal tracking type:", proposal_tracking_type)
print("proposal hash:", proposal_hash)
print("document hash:", document_hash)
print("stage:", stage)
print("appropriation:", appropriation)
print("leader_pubkey:", leader_pubkey)
print("leader_privkey:", leader_privkey)
print("new_leader_pubkey:", new_leader_pubkey)
print("new_leader_privkey:", new_leader_privkey)
print("secretary general privkey:", secretary_general_privkey)

local cp_payload =crcproposaltracking.new(proposal_tracking_type,proposal_hash,
        document_hash, stage, appropriation, leader_pubkey, leader_privkey,
        new_leader_pubkey, new_leader_privkey, secretary_general_privkey)
print(cp_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x27, 0, cp_payload, 0)
print(tx:get())

-- input: from, fee1
local charge = tx:appendenough(addr, fee * 100000000)
print(charge)

-- outputpayload
local default_output = defaultoutput.new()

local charge_output = output.new(asset_id, charge, addr, 0, default_output)
tx:appendtxout(charge_output)

-- sign
tx:sign(wallet)
print(tx:get())

-- send
local hash = tx:hash()
local res = m.send_tx(tx)

print("sending " .. hash)

if (res ~= hash)
then
    print(res)
else
    print("tx send success")
end
