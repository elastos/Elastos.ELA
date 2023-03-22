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


local amount = getDepositAmount()
local stakeuntil = getStakeUntil()
local fee = getFee()
local deposit_address = getDepositAddr()
local own_publickey = pub
local node_publickey = pub
local nick_name = getNickName()
local url = getUrl()
local location = getLocation()
local host_address = getHostAddr()

if amount == 0
then
    amount = 5000
end

if fee == 0
then
    fee = 0.001
end

if deposit_address == ""
then
    print("deposit addr is nil, should use --depositaddr or -daddr to set it.")
    return
end

if own_publickey == ""
then
    print("owner public key is nil, should use --ownerpublickey or -opk to set it.")
    return
end

if node_publickey == ""
then
    print("node public key is nil, should use --nodepublickey or -npk to set it.")
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

if stakeuntil == 0
then
    stakeuntil = 9999999
end


print("deposit amount:", amount)
print("fee:", fee)
print("deposit addr:", deposit_address)
print("owner public key:", own_publickey)
print("node public key:", node_publickey)
print("nick name:", nick_name)
print("url:", url)
print("location:", location)
print("host_address",host_address)
print("stakeuntil:", stakeuntil)


-- register producer payload: publickey, nickname, url, local, host, wallet
local rp_payload = registerv2producer.new(own_publickey, node_publickey, nick_name, url, location, host_address, stakeuntil, wallet, account)
print(rp_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x09, 2, rp_payload, 0)
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
-- print(charge_output:get())
-- print(amount_output:get())

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
