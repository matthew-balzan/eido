# Eido

Discord Golang bot

### Features

- Play audio to your voice channel from Youtube videos
  - Commands: `play` , `skip`, `pause`, `resume`, `clear`, `queue`, `disconnect`

### Install

You need these libraries installed in your enviroment:

- Golang
- FFmpeg

Create a file in the root called `config-prod.env` with this content:

```
DISCORD_TOKEN = Bot xxxx 
```

`xxxx` is the token of your bot, keep the `Bot` string

### Run

`go run cmd/eido/main.go`

### Other configs

You can also create other config files for more enviroments, but you have to set the global variable `APP_ENV`.

Example: `APP_ENV = dev` --> picks the config file `config-dev.env`




