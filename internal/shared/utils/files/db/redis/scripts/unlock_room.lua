---@diagnostic disable: undefined-global
---
--- Created by brettkohler.
--- DateTime: 8/13/25 8:56 AM
---
local roomKey = KEYS[1]
local lockKey = ARGV[1]

if redis.call("GET", roomKey) == lockKey then
	return redis.call("DEL", roomKey)
else
	return 0
end
