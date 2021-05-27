-- Copyright (c) 2017-2021 The Elastos Foundation
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

-- account
local wallet = client.new(keystore, password, false)
local addr = wallet:get_address()

print("addr", addr)

-- asset_id
local asset_id = m.get_asset_id()

local amount = getAmount()
local fee = getFee()
local recipient = getToAddr()
local target_data = getTargetData()
local deposit_addr = getDepositAddr()

if amount == 0 then
amount = 0.2
end

if fee == 0 then
fee = 0.1
end

if recipient == "" then
print("recipient is nil, should use --recipient to set it.")
return
end

if deposit_addr == "" then
print("depositaddr is nil, should use --depositaddr to set it.")
return
end

print("amount:", amount)
print("fee:", fee)
print("to:", recipient)
print("deposit_addr:", deposit_addr)

-- payload
local ta = transfercrosschainasset.new()

-- transaction: version, tx_type, payload_version, payload, locktime
local tx = transaction.new(9, 0x08, 1, ta, 0)

-- input: from, amount + fee
local charge = tx:appendenough(addr, (amount + fee) * 100000000)
print("charge", charge)

-- outputpayload
local cross_chain_output = crosschainoutput.new(recipient, amount, target_data)
print("cross_chain_output", cross_chain_output:get())

local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, output_paload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
local amount_output = output.new(asset_id, amount * 100000000, deposit_addr, 3, cross_chain_output)
-- print("txoutput", charge_output:get())
-- print("txoutput", amount_output:get())
tx:appendtxout(charge_output)
tx:appendtxout(amount_output)
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