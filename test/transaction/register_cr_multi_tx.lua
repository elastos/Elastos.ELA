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
print(addr)

-- asset_id
local asset_id = m.get_asset_id()

local amount = getDepositAmount()
local fee = getFee()
local deposit_address = getDepositAddr()
local nick_name = getNickName()
local url = getUrl()
local location = getLocation()
local payload_version = getPayloadVersion()

if amount == 0
    then
    amount = 5000
end

if fee == 0
    then
    fee = 0.1
end

if deposit_address == ""
    then
    print("deposit addr is nil, should use --depositaddr or -daddr to set it.")
    return
end

if nick_name == ""
    then
    nick_name = "nickname_test"
end

if url == ""
    then
    url = "url_test"
end

if location == ""
    then
    location = 123
end

print("deposit amount:", amount)
print("fee:", fee)
print("deposit addr:", deposit_address)
print("nick name:", nick_name)
print("url:", url)
print("location:", location)
print("payload version:", payload_version)

-- register cr payload: publickey, nickname, url, local, wallet
local rp_payload =registercr.new(payload_version, nick_name, url,
    location, 3, wallet)
print(rp_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x21, payload_version, rp_payload, 0)
print(tx:get())

-- input: from, amount + fee
local charge = tx:appendenough(addr, (amount + fee) * 100000000)
print(charge)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, outputpaload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)

local amount_output = output.new(asset_id, amount * 100000000, deposit_address, 0, default_output)


tx:appendtxout(charge_output)
tx:appendtxout(amount_output)

-- sign
tx:multisign(wallet, 3)
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
