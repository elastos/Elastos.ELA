-- Copyright (c) 2017-2020 The Elastos Foundation
-- Use of this source code is governed by an MIT
-- license that can be found in the LICENSE file.
-- 

local m = require("api")

local keystore = getWallet()
local password = getPassword()
local referkey = getReferKey()

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
print("referkey", referkey)

-- asset_id
local asset_id = m.get_asset_id()

-- amount, fee, recipent
--local amount = 0.2
--local fee = 0.001
-- candidate need to be code
--local vote_candidates = {'21039d419986f5c2bf6f2a6f59f0b6e111735b66570fb22107a038bca3e1005d1920ac'}
--local vote_candidate_votes = {'0.1'}

local vote_type = 0x04
local fee = getFee()
local vote_candidates = getCandidates()
local vote_candidate_votes = getCandidateVotes()
local locktime = getStakeUntil()

if fee == 0 then
    fee = 0.1
end

if next(vote_candidates) == nil then
    print("candidates is nil, should use --candidates or -cds to set it.")
    return
end

if next(vote_candidate_votes) == nil then
    print("candidate votes is nil, should use --candidateVotes or -cvs to set it.")
    return
end

print("votetype:",vote_type)
print("amount:", amount)
print("fee:", fee)
print("vote_candidates:", vote_candidates)
print("-----------------------")
local vote_candidates_num = 0
for i, v in pairs(vote_candidates) do
    print(i, v)
    vote_candidates_num = i
end
print("-----------------------")

print("vote_candidate_votes:", vote_candidate_votes)
print("-----------------------")
local vote_candidate_votes_num = 0
for i, v in pairs(vote_candidate_votes) do
    print(i, v)
    vote_candidate_votes_num = i
end
print("-----------------------")

if vote_candidates_num == vote_candidate_votes_num then

    -- payload
    local ta = renewvoting.new(vote_type, vote_candidates, vote_candidate_votes, locktime, referkey)

    -- transaction: version, tx_type, payload_version, payload, locktime
    local tx = transaction.new(9, 0x63, 0x01, ta, 0)

    -- input: from, amount + fee
    local charge = tx:appendenough(addr,fee * 100000000)
    print("charge", charge)

    local default_output = defaultoutput.new()
    -- output: asset_id, value, recipient, output_paload_type, output_paload
    local charge_output = output.new(asset_id, charge, addr, 0, default_output)
    -- print("txoutput", charge_output:get())
    -- print("txoutput", amount_output:get())
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
    else
        print("The numerical candidates and candidateVotes are inconsistent!")
end
