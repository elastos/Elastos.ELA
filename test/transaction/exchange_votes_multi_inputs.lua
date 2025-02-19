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

--todo add get_addresses
local addresses = getAddresses()
-- account
--local c = wallet:get_address()

--vote for saddr   mainaccount addr
local saddr = wallet:get_s_address()
print("saddr", saddr)

-- asset_id
local asset_id = m.get_asset_id()
local amount = getAmount()
local fee = getFee()

if amount == 0 then
    amount = 0.2
end

if fee == 0 then
    fee = 0.1
end

print("address len ", #addresses)
print("amount ", amount)
print("fee:", fee)


-- transaction payload
local ta = exchangevotes.new()

-- transaction: version, tx_type, payload_version, payload, locktime
--define transaction and set tx payload
local tx = transaction.new(9, 0x62, 0, ta, 0)

--iterate every address, each one vote amount votes
--stake output
-- define output payload (vote to saddr)
local vote_output = stakeoutput.new(0, saddr)
print("vote_output", vote_output:get())
--all output to    stake_pool for stake
stake_pool = "SNmCKtp1NmPfjwEo4m8PbgirDfYss1NUyT"
--define output                        OTStake = 7



local amount_output = output.new(asset_id, #addresses*amount * 100000000, stake_pool, 7, vote_output)
tx:appendtxout(amount_output)

for i, addr in ipairs(addresses) do
    print(i, addr)
    -- input: from, amount + fee
    --append amount + fee input to tx and return  charge(change ) aount
    local tempFee = fee
    if( i ~= 0 )
    then
      tempFee = 0
    end

    local charge = tx:appendenoughmultiinput(addr, (amount* 100000000 + fee* 100000000) )
    print("charge", charge)

    ----charge output
    --define output payload
    local charge_output_payload = defaultoutput.new()
    -- output: asset_id, value, recipient, output_paload_type, output_paload
    local charge_output = output.new(asset_id, charge, addr, 0, charge_output_payload)
     tx:appendtxout(charge_output)
end


print(tx:get())
-- sign
tx:multiprogramssigntx(wallet)
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
