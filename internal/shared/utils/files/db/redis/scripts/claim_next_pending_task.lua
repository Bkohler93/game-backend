---@diagnostic disable: undefined-global
---
--- Created by brettkohler.
--- DateTime: 8/1/25 1:05 PM
-- KEYS[1] = pending sorted set key
-- KEYS[2] = inprogress sorted set key
-- ARGV[1] = current time (now)

local pendingKey = KEYS[1]
local inprogressKey = KEYS[2]
local now = tonumber(ARGV[1])

local tasks = redis.call('ZRANGEBYSCORE', pendingKey, '-inf', now, 'LIMIT', 0, 1)
if #tasks == 0 then
    return {err="NO_AVAILABLE_TASK"}
end

local task = tasks[1]

redis.call('ZREM', pendingKey, task)

redis.call('ZADD', inprogressKey, now, task)

return task

