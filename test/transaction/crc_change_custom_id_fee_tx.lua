-- Copyright (c) 2017-2020 The Elastos Foundation
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
local cr_pubkey = getPublicKey()
local draft_data = getDraftData()
local rate = getRateOfCustomIDFee()

if fee == 0
then
    fee = 0.0001
end

if cr_pubkey == "" then
    cr_pubkey = pubkey
end

if draft_data == "" then
    print("draftdata is nil, should use --draftdata to set it.")
    return
end

if rate == "" then
    print("customidrate is nil, should use --customidrate to set it.")
    return
end

print("fee:", fee)
print("public key:", cr_pubkey)
print("rate:", rate)
print("draft_data :", draft_data)

-- crc close proposal hash payload: crPublickey, proposalType, draftData, close_proposal_hash, wallet
local cp_payload =crcchangecustomidfee.new(cr_pubkey, draft_data, rate, wallet)

print(cp_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x25, 1, cp_payload, 0)
print(tx:get())

-- input: from, fee
local charge = tx:appendenough(addr, fee * 100000000)
print(charge)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, outputpaload
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
