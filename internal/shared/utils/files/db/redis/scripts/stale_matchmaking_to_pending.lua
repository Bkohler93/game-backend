---@diagnostic disable: undefined-global
---
--- Created by brettkohler.
--- DateTime: 7/31/25 3:17 PM
---
local inprogressKey = KEYS[1]
local pendingKey = KEYS[2]
local member = ARGV[1]
local processTime = tonumber(ARGV[2])

local zremResult = redis.call('ZREM', inprogressKey, member)

if zremResult == 0 then
	return {err="MEMBER_NOT_FOUND"}
end

local zaddResult = redis.call('ZADD', pendingKey, processTime, member)

if zaddResult ~= 1 then 
	return {err="MEMBER_NOT_ADDED"}
end

return 1

