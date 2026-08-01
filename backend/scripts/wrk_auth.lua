-- wrk Lua helper for authenticated requests.
-- Cookie and request body are injected by loadtest.sh via environment.
-- If WRK_BODY contains the __SEQ__ placeholder, it is replaced with a
-- per-thread monotonically increasing counter so POST bodies (e.g.
-- client_msg_id) stay unique across requests.

local cookie = os.getenv("WRK_COOKIE") or ""
local body = os.getenv("WRK_BODY") or ""
local method = os.getenv("WRK_METHOD") or "GET"
local counter = 0
local use_seq = string.find(body, "__SEQ__", 1, true) ~= nil

function request()
    counter = counter + 1
    local headers = {}
    if cookie ~= "" then
        headers["Cookie"] = cookie
    end
    if method == "POST" and body ~= "" then
        headers["Content-Type"] = "application/json"
        local b = body
        if use_seq then
            b = string.gsub(body, "__SEQ__", tostring(counter))
        end
        return wrk.format("POST", wrk.path, headers, b)
    end
    return wrk.format(method, wrk.path, headers)
end
