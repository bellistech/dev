-- A simple Lua script that prints a greeting with decorative ribbons and confetti.
local function ribbon(text)
  local line = string.rep("=", #text)
  return table.concat({line, text, line}, "\n")
end

local function confetti(glue)
  local bits = {"*", "+", "~", "^", "<3"}
  return table.concat(bits, glue)
end

local greeting = "Hello, Lua world!"
print(ribbon(greeting))
print("Confetti: " .. confetti(" "))
print("Timestamp: " .. os.date("%Y-%m-%d %H:%M:%S"))