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

local fee = getFee()
if fee == 0
then
    fee = 0.001
end

print("fee",fee)

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

local own_publickey = pub
-- cancel producer payload: publickey, wallet
print("before cancelproducerschnorr")
local cp_payload = cancelproducerschnorr.new(own_publickey)
print(cp_payload:get())

print("before transaction.new")
-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x0a, 1, cp_payload, 0)

-- input: from, amount + fee
print("before tx:appendenough")

local charge = tx:appendenough(addr, fee * 100000000)
print(charge)

print("before defaultoutput.new")

-- outputpayload
local default_output = defaultoutput.new()

print("before output.new")

-- output: asset_id, value, recipient, output_paload_type, outputpaload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
tx:appendtxout(charge_output)
-- print(charge_output:get())
print("before signschnorr")

-- sign
tx:signschnorr(account)
print("tx",tx:get())

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
