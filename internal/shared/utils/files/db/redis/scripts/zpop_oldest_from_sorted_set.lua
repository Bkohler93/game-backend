---@diagnostic disable: undefined-global
-- KEYS[1] = sorted set key 
-- KEYS[2] = inprogress sorted set key
-- ARGV[1] = current time (now)
local sortedSetKey = KEYS[1]
local now = tonumber(ARGV[1])

local tasks = redis.call('ZRANGEBYSCORE', sortedSetKey, '-inf', now, 'LIMIT', 0, 1)
if #tasks == 0 then
    return {err="NO_AVAILABLE_TASK"}
end
local task = tasks[1]

redis.call('ZREM', sortedSetKey, task)

return task
