---@diagnostic disable: undefined-global
local roomKey = KEYS[1]
local playerId = ARGV[1]
local userSkill = tonumber(ARGV[2])
local maxPlayers = tonumber(ARGV[3])

local rawRoom = redis.call("JSON.GET", roomKey, "$")
if not rawRoom then
    return {err="ROOM_NOT_FOUND"}
end

local decoded = cjson.decode(rawRoom)
local room = decoded[1]

local currentCount = tonumber(room.player_count) or 0
if currentCount == 0 then
    return {err="INVALID_COUNT"}
end
if currentCount >= maxPlayers then
    return {err="ROOM_FULL"}
end

-- Add player
table.insert(room.player_ids, playerId)
room.player_count = room.player_count + 1

-- Update average skill
local oldSkill = tonumber(room.average_skill) or 0
room.average_skill = math.floor(((oldSkill * (currentCount)) + userSkill) / room.player_count)

-- Mark full if needed
if room.player_count == maxPlayers then
    room.is_full = 1
end

-- Save the updated object
redis.call("JSON.SET", roomKey, "$", cjson.encode(room))

-- Return the full room object
return cjson.encode(room)
