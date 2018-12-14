local api = require("api")

local function do_files(inputstr, sep, base_path)
    if sep == nil then
        sep = "%s"
    end
    for str in string.gmatch(inputstr, "([^" .. sep .. "]+)") do
        dofile(base_path .. str)
    end
end

local dpos_dir = "test/white_box/dpos/"
local dpos_files = api.get_dir_all_files(dpos_dir)
do_files(dpos_files, ",", dpos_dir)

