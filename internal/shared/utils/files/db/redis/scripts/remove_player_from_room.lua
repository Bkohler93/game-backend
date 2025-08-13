---@diagnostic disable: undefined-global
local roomKey = KEYS[1]
local playerId = ARGV[1]
local playerSkill = tonumber(ARGV[2]) or -1
if playerSkill == -1 then
	return {err = "INVALID_PLAYER_SKILL_ARG" }
end

local function decodeJson(json)
	local decodedStuff = cjson.decode(json)
	return decodedStuff[1]
end

--- remove player from player_ids
local playersJson = redis.call("JSON.GET", roomKey, "$.player_ids")
if not playersJson then
    return { err = "PLAYERS_ARRAY_NOT_FOUND" }
end
local players = decodeJson(playersJson)

local newPlayers = {}
for i = 1, #players do
    if players[i] ~= playerId then
        table.insert(newPlayers, players[i])
    end
end

--- calculate new average skill for room (average skill * count - userSkill) / count - 1
local roomCountJson = redis.call("JSON.GET", roomKey, "$.player_count")
if not roomCountJson then
    return { err = "ROOM_COUNT_NOT_FOUND" }
end
local roomCount = decodeJson(roomCountJson)

local roomSkillJson = redis.call("JSON.GET", roomKey, "$.average_skill")
if not roomSkillJson then
	return { err = "INVALID_AVERAGE_SKILL" }	
end
local roomSkill = decodeJson(roomSkillJson)

if roomCount-1 == 0 then
	redis.call("JSON.DEL", roomKey, "$")
	return "0"
end

local newAverageSkill = (roomSkill * roomCount - playerSkill) / (roomCount - 1)

local newPlayersJson = cjson.encode(newPlayers)

redis.call("JSON.SET", roomKey, "$.player_ids", newPlayersJson)
redis.call("JSON.SET", roomKey, "$.average_skill", newAverageSkill)
redis.call("JSON.SET", roomKey, "$.player_count", roomCount-1)

return newPlayersJson 