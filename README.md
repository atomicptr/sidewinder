# sidewinder

Simple service that reads RSS feeds and posts them to chat channels via webhooks

It's like a friend who nags you with links all the time

## Motivation

There are a bunch of RSS feeds I follow where I just want to know when something happens and I somehow can't get myself to actually use an RSS feed reader, I do however check chats very often :)

## Configuration

This service is mostly configured through a TOML file, see the example file "sidewinder.toml.example"

### Env Vars

There are also a few environment variables:

- **SIDEWINDER_CONFIG_FILE**: Path to the config file
- **SIDEWINDER_DATA_DIR**: Path to the data dir
- **SIDEWINDER_TICK_RATE**: Tick rate (example `30m`)

## License

AGPLv3
