---@diagnostic disable: undefined-global
local roomOneKey = KEYS[1]
local roomTwoKey = KEYS[2]
--local playerId = ARGV[1]
--local userSkill = tonumber(ARGV[2])
local maxPlayers = tonumber(ARGV[1])

local rawRoomOne = redis.call("JSON.GET", roomOneKey, "$")
if not rawRoomOne then
    return {err="ROOM_NOT_FOUND"}
end

local decodedOne = cjson.decode(rawRoomOne)
local roomOne = decodedOne[1]

local rawRoomTwo = redis.call("JSON.GET", roomTwoKey, "$")
if not rawRoomTwo then
    return {err="ROOM_NOT_FOUND"}
end

local decodedTwo = cjson.decode(rawRoomTwo)
local roomTwo = decodedTwo[1]


--- check if rooms have compatible number of players, calculate new player count 
local roomOnePlayerCount = tonumber(roomOne.player_count) or 0
if roomOnePlayerCount == 0 then 
	return {err="INVALID_COUNT"}
end

local roomTwoPlayerCount = tonumber(roomTwo.player_count) or 0
if roomTwoPlayerCount == 0 then 
	return {err="INVALID_COUNT"}
end

local newCount = roomOnePlayerCount + roomTwoPlayerCount 
if newCount == 0 then
    return {err="INVALID_COUNT"}
end
if newCount > maxPlayers then
    return {err="ROOM_FULL"}
end

local roomOneFinalIdx = math.floor(roomOnePlayerCount) or -1
if roomOneFinalIdx == -1 then 
	return {err="MATH_FAIL"}
end

--- update player ids
for i = 1, roomOneFinalIdx do
    table.insert(roomTwo.player_ids, roomOne.player_ids[i])
end

if #roomTwo.player_ids ~= newCount then
	return {err="UPDATED_PLAYER_IDS_INCORRECT"}
end	

-- Update average skill
local roomOneSkill = tonumber(roomOne.average_skill) or 0
local roomTwoSkill = tonumber(roomTwo.average_skill) or 0 
local newRoomTwoSkill = math.floor((roomOneSkill * roomOnePlayerCount + roomTwoSkill * roomTwoPlayerCount)/(roomOnePlayerCount + roomTwoPlayerCount))
roomTwo.average_skill = newRoomTwoSkill

-- Mark full if needed
roomTwo.player_count = newCount
if roomTwo.player_count == maxPlayers then
    roomTwo.is_full = 1
end

-- Save the updated object
redis.call("JSON.SET", roomTwoKey, "$", cjson.encode(roomTwo))

-- Return the combined room object (roomTwo
return cjson.encode(roomTwo)
