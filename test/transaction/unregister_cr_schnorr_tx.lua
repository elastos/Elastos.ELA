-- Copyright (c) 2017-2020 The Elastos Foundation
-- Use of this source code is governed by an MIT
-- license that can be found in the LICENSE file.
-- 

local m = require("api")

-- account
local privatekeys = getPrivateKeys()
print("------------------------")
for i, v in pairs(privatekeys) do
    print(i, v)
end
print("------------------------")

-- account
local publickeys = getPublicKeys()
print("------------------------")
for i, v in pairs(publickeys) do
    print(i, v)
end
print("------------------------")

print("------------------------")
local account = account.new(privatekeys)
local addr = account:get_address()
print("addr",addr)
print("------------------------")

print("------------------------")
local aggpub = aggpub.new(publickeys)
local pub = aggpub:get_aggpub()
print("pub",pub)
print("------------------------")

-- asset_id
local asset_id = m.get_asset_id()

-- amount, fee
-- local amount = 5000
local fee = 0.001

-- deposit params
local cr_publickey = pub

if cr_publickey == ""
then
    print("pubic key is nil, should use --pubkey or -pk to set it.")
    return
end
print("public key:", cr_publickey)

--local cr_publickey =
--"039d419986f5c2bf6f2a6f59f0b6e111735b66570fb22107a038bca3e1005d1920"


-- unregister cr payload: publickey,  wallet
local rp_payload =unregistercr.new(cr_publickey, wallet)
print(rp_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x22, 1, rp_payload, 0)
print(tx:get())

-- input: from, amount + fee
local charge = tx:appendenough(addr, fee * 100000000)
print(charge)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, outputpaload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
tx:appendtxout(charge_output)


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
