---@diagnostic disable: undefined-global
---
--- Created by brettkohler.
--- DateTime: 7/30/25 4:05 PM
---
-- Lua script to XACK and XDEL a message atomically
-- KEYS[1] = stream key
-- ARGV[1] = group name
-- ARGV[2] = message ID

redis.call('XACK', KEYS[1], ARGV[1], ARGV[2])
redis.call('XDEL', KEYS[1], ARGV[2])
return 1