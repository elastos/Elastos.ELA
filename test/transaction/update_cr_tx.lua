local m = require("api")

-- client: path, password, if create
local wallet = client.new("keystore.dat", "123", false)

-- account
local addr = wallet:get_address()
local pubkey = wallet:get_publickey()
print(addr)
print(pubkey)

-- asset_id
local asset_id = m.get_asset_id()

-- amount, fee
--local amount = 5000
local fee = 0.001

-- deposit params
local deposit_address = "DWfJsXviRaTwhToeer7ftWVXoNUv4vuEDQ"
local cr_publickey = "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
local nick_name = "ela_test11"
local url = "ela_test.org11"
local location = "00112211"

-- register cr payload: publickey, nickname, url, local, wallet
local up_payload =updatecr.new(cr_publickey, nick_name, url, location, wallet)
print(up_payload:get())

-- transaction: version, txType, payloadVersion, payload, locktime
local tx = transaction.new(9, 0x23, 0, up_payload, 0)
print("tx1")
print(tx:get())

-- input: from, amount + fee
local charge = tx:appendenough(addr, fee * 100000000)
print("charge " .. charge)

-- outputpayload
local default_output = defaultoutput.new()

-- output: asset_id, value, recipient, output_paload_type, outputpaload
local charge_output = output.new(asset_id, charge, addr, 0, default_output)
--local amount_output = output.new(asset_id, amount * 100000000, deposit_address, 0, default_output)
tx:appendtxout(charge_output)
--tx:appendtxout(amount_output)
-- print(charge_output:get())
-- print(amount_output:get())

-- sign
tx:sign(wallet)
print("tx2 ")
print(tx:get())

-- send
local hash = tx:hash()
print("before send_tx ")

local res = m.send_tx(tx)

print("sending " .. hash)

if (res ~= hash)
then
    print(res)
else
    print("tx send success")
end
