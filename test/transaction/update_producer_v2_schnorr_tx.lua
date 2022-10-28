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

-- fee
local fee = 0.001

local own_publickey = pub
local node_publickey = pub
local stakeuntil = getStakeUntil()
local nick_name = getNickName()
local url = getUrl()
local location = getLocation()
local host_address = getHostAddr()

if stakeuntil == ""
	then
		print("stakeuntil is nil, should use --stakeuntil to set it.")
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

if host_address == ""
	then
		print("host address is nil, should use --host to set it.")
		return
end

print("ower public key:", own_publickey)
print("node public key:", node_publickey)
print("nick_name:", nick_name)
print("url:", url)
print("location:", location)
print("host address:", host_address)
print("stakeuntil:", stakeuntil)

-- update producer payload: publickey, nickname, url, local, host, wallet
local up_payload = updatev2producer.new(own_publickey, node_publickey, nick_name, url, location, host_address, stakeuntil, wallet, account)
print(up_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x0b, 2, up_payload, 0)

-- input: from, amount + fee
local charge = tx:appendenough(addr, fee * 100000000)
print(charge)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, outputpaload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
tx:appendtxout(charge_output)

-- sign
--tx:sign(wallet)
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
