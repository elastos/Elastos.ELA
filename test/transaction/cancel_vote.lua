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
local fee = getFee()
local referKey = getReferKey()

if referKey == "" then
    print("referKey is nil, should use --referKey to set it.")
    return
end

if fee == 0 then
    fee = 0.1
end

print("referKey:", referKey)
print("fee:", fee)


-- payload
local ta = cancelvotes.new(referKey)

-- transaction: version, tx_type, payload_version, payload, locktime
local tx = transaction.new(9, 0x64, 0, ta, 0)

-- input: from, fee
local charge = tx:appendenough(addr, (fee) * 100000000)
print("charge", charge)

local default_output = defaultoutput.new()
-- output: asset_id, value, recipient, output_paload_type, output_paload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
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
