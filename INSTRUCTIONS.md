# Installation & Usage

## Language Server

Ayla uses `elen` as it's lsp.

Install it with:

```bash
go install github.com/z-sk1/ayla-lang/elen@latest
```

Make sure `$GOPATH/bin` is in your PATH.

if go isnt in your path, please download go [here](https://go.dev).

## Windows

to use the cli, please go to the **Releases** tab and download the windows binary:
- `ayla-win-amd64` for x86 windows machines
- `ayla-win-arm64` for arm windows machines

extract the zip, and move the `.exe` file to an easy to access folder, for example:

C:\ayla

add that folder to your **PATH**:

1. press `Win + R`, type `sysdm.cpl`, and press enter  
2. go to **Advanced** → **Environment Variables**  
3. under **System variables**, find `Path` → `Edit`  
4. add: C:\ayla

open a new **Command Prompt** and verify:

```bash
ayla --help
```

there isn’t a REPL currently, so make sure to put `ayla` in front of every command.

---

## macOS

to use the cli, please go to the **Releases** tab and download the macOS binary:

- `ayla-mac-amd64` for Intel Macs  
- `ayla-mac-arm64` for Apple Silicon (M1/M2/M3)

open **Terminal** and navigate to the download location.

make the file executable:

```zsh
chmod +x ayla
```

move it to a directory in your PATH:

```zsh
sudo mv ayla /usr/local/bin
```

if `/usr/local/bin` does not exist, create it with:

```zsh
sudo mkdir -p /usr/local/bin
```

verify it works:

```zsh
ayla --help
```

there isn’t a REPL currently, so make sure to put `ayla` in front of every command.

#### Gatekeeper note

if macOS blocks the binary because it is from an unidentified developer, run:

```zsh
xattr -d com.apple.quarantine /usr/local/bin/ayla
```

---

## Linux
to use the cli, please go to the **Releases** tab and download the linux binary:
- `ayla-linux-amd64` for x86 linux machines
- `ayla-linux-arm64` for arm linux machines

or using curl:

amd64:
```bash
curl -L -o ayla https://github.com/z-sk1/ayla-lang/releases/latest/download/ayla-linux-amd64
```

arm64:
```bash
curl -L -o ayla https://github.com/z-sk1/ayla-lang/releases/latest/download/ayla-linux-arm64
```

make the file executable:
```bash
chmod +x ayla
```

move it to a directory in your PATH:

```bash
sudo mv ayla /usr/local/bin
```

if `/usr/local/bin` does not exist, create it with:

```bash
sudo mkdir -p /usr/local/bin
```

verify it works:

```bash
ayla --help
```

## running

to run a script do:

```bash
ayla run [--debug] [--timed] <file>
```
> --debug will give debug info like ast, and tokens

> --timed will time how long your program takes

```bash
ayla run test.ayl
```
and also for the other extension
```bash
ayla run test.ayla
```

you can also do it without putting a file extension
```bash
ayla run test
```
this will first try appending .ayla, then if not found it will try appending .ayl

if test.(ayl/ayla) does not exist then ayla CLI will throw an error:
```bash
file not found: test.ayla (.ayla or .ayl) 
```

version:

```bash
ayla --version
```

help:

```bash
ayla --help
```
