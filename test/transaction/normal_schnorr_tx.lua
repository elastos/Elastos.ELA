-- Copyright (c) 2017-2020 The Elastos Foundation
-- Use of this source code is governed by an MIT
-- license that can be found in the LICENSE file.
-- ~/bin/ela-cli script -f ~/bin/transaction/normal_schnorr_tx.lua --amount=0.1 --fee=0.001 --to EgLe9ZAQyLmjxFZLp5em9VfqsYKvdhpGys -w ~/docker-ela/foundation.dat --privatekeys 065f4a7db76a360c2d541d6d5522a817e0ccd116ec25e402ffcb0f5e75026096,c58854279b97f470aa436498ce5f2be887d9043fc97eefed530c3f0d0642fdaa,b8beaa118d97b68442c1ccb54a40fcd02ac18c96b597932e2b22c6f3f44f1207  --rpcport 10116
local m = require("api")

-- account
local privatekeys = getPrivateKeys()
print("------------------------")
for i, v in pairs(privatekeys) do
    print(i, v)
end
print("------------------------")

print("------------------------")
local account = account.new(privatekeys)
local addr = account:get_address()
print(addr)
print("------------------------")

-- asset_id
local asset_id = m.get_asset_id()
-- amount, fee, recipent
local amount = getAmount()
local fee = getFee()
local recipient = getToAddr()

if amount == 0 then amount = 1.0 end
if fee == 0 then fee = 0.1 end
if recipient == "" then
	print("to addr is nil, should use --to to set it.")
	return
end

print("amount:", amount)
print("fee:", fee)
print("recipient:", recipient)

-- payload
local ta = transferasset.new()

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x02, 0, ta, 0)

-- input: from, amount + fee
local charge = tx:appendenough(addr, (amount + fee) * 100000000)
print("return:", charge/1e8)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, output_paload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
local recipient_output = output.new(asset_id, amount * 100000000, recipient, 0, default_output)
tx:appendtxout(charge_output)
tx:appendtxout(recipient_output)

-- print(charge_output:get())
-- print(recipient_output:get())
-- sign
tx:signschnorr(account)
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
