# ayla lang
<img width="512" height="512" alt="ayla-512" src="https://github.com/user-attachments/assets/1a266fdd-0d0d-4f95-83fa-1fd5d7bca0f9" />

ayla lang is a statically typed interpreted language written in go, designed to make you forget everything

*Because f-ck you.* - Linus Torvalds

# about

## our team
- **Me: z-sk1, Co-Owner**
- **and Mregg55, Co-Owner (link: https://github.com/mregg55)**

## language server
view the repo [here](https://github.com/z-sk1/elen)

## vs code extension
get it [here](https://marketplace.visualstudio.com/items?itemName=z-sk1.ayla)

view the repo [here](https://github.com/z-sk1/vscode-ayla)

## zed extension
view the repo [here](https://github.com/z-sk1/zed-ayla)

# Installation and Usage
See [INSTRUCTIONS.md](./INSTRUCTIONS.md) for full step-by-step instructions for macOS and Windows and Linux.

---

## built in functions!`
- `put(t ...thing)` – prints values to stdout
- `putln(t ...thing)` — prints values to stdout and adds '\n' at the end
- `explode(t ...thing)` — spits out a runtime error containing the message in the parameter 
- `scanln(t ...thing)` – scans console input after clicking enter 
- `scankey(t ...thing)` – scans key press in console
- `toBool(t thing)` – parses a value to boolean
- `toString(t thing)` – parses a value to string
- `toInt(t thing)` – parses a value to integer
- `toFloat(t thing)` – parses a value to float
- `typeof(t thing)` – returns type of value as string
- `len(t thing)` – returns length of arrays or slices or strings or maps
- `cap(t thing)` – returns capacity of arrays or slices
- `make(t Type, size ...int)` – creates and returns a slice or map
- `append(slice []Type, t ...thing)` – appends a set of values to a slice then returns the final slice  
- `delete(m map[Type]Type, key thing)`
- `wait(ms)` – wait for a duration in milliseconds
- `randi(range ...int)` – gives a random integer based off a range 
- `randf(range ...int)` – gives a random float based off a range 