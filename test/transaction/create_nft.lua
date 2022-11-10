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

print("addr", addr)
print("pubkey", pubkey)

-- asset_id
local asset_id = m.get_asset_id()
local fee = getFee()
local nft_id = getNFTID()

if fee == 0 then
	fee = 0.001
end

if nft_id == "" then
    print("nft id is nil, should use --nftid to set it.")
    return
end

print("fee:", fee)
print("nft id:", nft_id)

-- cancel producer payload: publickey, wallet
local nft_payload = createnft.new(nft_id)
print(nft_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x71, 0, nft_payload, 0)

-- input: from, amount + fee
local charge = tx:appendenough(addr, fee * 100000000)
print(charge)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, outputpaload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
tx:appendtxout(charge_output)
-- print(charge_output:get())

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
