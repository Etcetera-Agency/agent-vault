# Design: Agent Service Profile

## Fork-Conflict Surface

This repo is a fork of upstream Agent Vault. Agent profile work should be
isolated from upstream service/auth internals:

- add profile data in new files or fixtures
- call existing service/proposal import paths instead of changing core storage flow
- keep profile-specific defaults out of generic upstream constructors
- mark required upstream-file touches with `// fork-local:`
- add profile tests in new fork-local `*_test.go` files

## Profile YAML

```yaml
vault: agent
services:
  - name: gmail-list-read
    host: gmail.googleapis.com/gmail/v1/users/me/messages*
    methods: [GET]
    auth:
      type: bearer
      token: GOOGLE_ACCESS_TOKEN

  - name: calendar-events-read
    host: www.googleapis.com/calendar/v3/calendars/*/events*
    methods: [GET]
    auth:
      type: bearer
      token: GOOGLE_ACCESS_TOKEN

  - name: discord-channel-messages
    host: discord.com/api/v10/channels/*/messages
    methods: [POST]
    auth:
      type: bearer
      token: DISCORD_BOT_TOKEN

  - name: telegram-bot-api
    host: api.telegram.org/bot__bot_token__/*
    methods: [POST]
    auth:
      type: passthrough
    substitutions:
      - key: TELEGRAM_BOT_TOKEN
        placeholder: __bot_token__
        in: [path]
```

## Verification Pseudocode

```text
require vault agent unmatched_host_policy == deny
apply agent profile services
call /discover
assert each service has expected host and methods

assert GET gmail messages matches gmail-list-read
assert POST gmail messages returns 403 method_not_allowed
assert GET gmail settings returns 403 no_match

assert GET calendar events matches calendar-events-read
assert POST calendar events returns 403 method_not_allowed
assert GET calendarList returns 403 no_match

assert POST discord channel messages matches discord-channel-messages
assert GET discord guild path returns 403 no_match

assert POST telegram placeholder path matches telegram-bot-api
assert forwarded upstream path contains real token
assert agent env/logs and Agent Vault request logs do not contain real token
```

## Placeholder Values

```text
GOOGLE_ACCESS_TOKEN=__google_access_token__
DISCORD_BOT_TOKEN=__discord_bot_token__
TELEGRAM_BOT_TOKEN=__telegram_bot_token__
```
