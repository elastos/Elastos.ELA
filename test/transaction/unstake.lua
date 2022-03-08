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
local saddr = wallet:get_s_address()

print("addr", addr)
print("saddr", saddr)
print("pubkey", pubkey)

-- asset_id
local asset_id = m.get_asset_id()

-- amount, fee, recipent
--local amount = 0.2
--local fee = 0.001
-- candidate need to be code
--local vote_candidates = {'21039d419986f5c2bf6f2a6f59f0b6e111735b66570fb22107a038bca3e1005d1920ac'}
--local vote_candidate_votes = {'0.1'}

local recipient = getToAddr()
local amount = getAmount()
local fee = getFee()
print("toAddr", recipient)

if amount == 0 then
    amount = 0.2
end

if fee == 0 then
    fee = 0.1
end

print("amount:", amount)
print("fee:", fee)


-- payload
local ta = unstake.new(pubkey,  recipient, amount * 100000000, wallet)

-- transaction: version, tx_type, payload_version, payload, locktime
local tx = transaction.new(9, 0x65, 0, ta, 0)

-- input: from, amount + fee
local charge = tx:appendenough(addr, (amount + fee) * 100000000)
print("charge", charge)

local default_output = defaultoutput.new()
-- output: asset_id, value, recipient, output_paload_type, output_paload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
local amount_output = output.new(asset_id, amount * 100000000, recipient, 0, default_output)
-- print("txoutput", charge_output:get())
-- print("txoutput", amount_output:get())
tx:appendtxout(amount_output)
tx:appendtxout(charge_output)

print(tx:get())

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
