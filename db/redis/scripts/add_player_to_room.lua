---@diagnostic disable: undefined-global
-- KEYS[1]: room_key (e.g., "room:room_123")
-- ARGV[1]: player_id (e.g., "player456")
-- ARGV[2]: user_skill (e.g., "100")
-- ARGV[3]: max_players_limit (e.g., "50") 
-- ARGV[3]: stream_key (e.g., "matchmake:room_events")

local roomKey = KEYS[1]
local playerId = ARGV[1]
local userSkill = tonumber(ARGV[2])
local maxPlayers = tonumber(ARGV[3])
local streamKey = ARGV[3]

local function jsonGetSingular(key, path)
    local raw = redis.call("JSON.GET", key, path)
    if not raw then
        return nil
    end
    local decoded = cjson.decode(raw)
    if type(decoded) == "table" then
        return decoded[1]
    else
        return decoded
    end
end

local function calculateNewAvgSkill(roomSkill, playerCount)
	local product = roomSkill * (playerCount - 1)
	local newSum = product + userSkill
	return math.floor(newSum / playerCount)
end

local currentCount = tonumber(jsonGetSingular( roomKey, "$.player_count")) or 0

if currentCount == 0 then
	return {err="INVALID_COUNT"}
end
if currentCount >= maxPlayers then
    return {err="ROOM_FULL"} 
end

local playersTable = jsonGetSingular(roomKey, "$.player_ids")
table.insert(playersTable, playerId)
local newPlayersJson = cjson.encode(playersTable)

redis.call("JSON.NUMINCRBY", roomKey, "$.player_count", 1)
redis.call("JSON.SET", roomKey, "$.player_ids", newPlayersJson)

local roomSkill = tonumber(jsonGetSingular(roomKey, "$.average_skill")) or 0
if roomSkill == 0 then
	return {err="INVALID_ROOM_SKILL"}
end	

local newCount = currentCount + 1
local newAvgSkill = calculateNewAvgSkill(roomSkill, newCount)

redis.call("JSON.SET", roomKey, "$.average_skill", tonumber(newAvgSkill))

if newCount == maxPlayers then
    redis.call("JSON.SET", roomKey, "$.is_full", 1)
    redis.call("XADD", streamKey, "*", "type", "room_completed", "room_id", string.sub(roomKey, 6), "match_details", "{}") -- Adjust match_details
end

return newCount 