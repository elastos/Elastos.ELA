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
print("wallet addr:", addr)
print("wallet public key:", pubkey)

-- asset_id
local asset_id = m.get_asset_id()

local fee = getFee()
local cr_pubkey = getPublicKey()
local proposal_type = 0x0410
local sideChainName = getSideChainName()
local magicNumber = getMagicNumber()
local genesisHash = getGenesisHash()
local exchangeRate = getExchangeRate()
local effectiveHeight = getEffectiveHeight()
local resourcePath = getResourcePath()
local draft_hash = getDraftHash()

if fee == 0
    then
    fee = 0.0001
end

if cr_pubkey == "" then
    cr_pubkey = pubkey
end

if sideChainName == "" then
    print("sideChainName is nil, should use --sidechainname to set it.")
    return
end

if magicNumber == 0 then
    print("magicNumber is 0, should use --magicnumber to set it.")
    return
end

if genesisHash == "" then
    print("genesisHash is nil, should use --genesishash to set it.")
    return
end

if exchangeRate == "" then
    print("exchangeRate is nil, should use --exchangerate to set it.")
    return
end

if effectiveHeight == 0 then
    print("effectiveHeight is nil, should use --effectiveheight to set it.")
    return
end

if resourcePath == "" then
    print("resourcePath is nil, should use --resourcepath to set it.")
    return
end

if draftHash == "" then
    print("draftHash is nil, should use --drafthash to set it.")
    return
end

print("fee:", fee)
print("public key:", cr_pubkey)
print("proposal type:", proposal_type)
print("sideChainName:", sideChainName)
print("magicNumber:", magicNumber)
print("genesisHash:", genesisHash)
print("exchangeRate:", exchangeRate)
print("effectiveHeight:", effectiveHeight)
print("resourcePath:", resourcePath)


local cp_payload =crcregistersidechainproposal.new(cr_pubkey, proposal_type,sideChainName,magicNumber,genesisHash,exchangeRate,effectiveHeight, resourcePath,draft_hash, wallet)
print(cp_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x25, 0, cp_payload, 0)
print(tx:get())

-- input: from, fee
local charge = tx:appendenough(addr, fee * 100000000)
print(charge)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, outputpaload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
tx:appendtxout(charge_output)

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
