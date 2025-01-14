
# Snake-Server


[![Build Status](https://travis-ci.org/ivan1993spb/snake-server.svg?branch=master)](https://travis-ci.org/ivan1993spb/snake-server) [![Go Report Card](https://goreportcard.com/badge/github.com/ivan1993spb/snake-server)](https://goreportcard.com/report/github.com/ivan1993spb/snake-server) [![GitHub release](https://img.shields.io/github/release/ivan1993spb/snake-server.svg)](https://github.com/ivan1993spb/snake-server/releases/latest)

The Snake-Server is a server for the online arcade game snake. In this edition you can interact with other players and even eat small snakes!

Take a look at a working instance here - http://snakeonline.xyz

## Table of contents

- [Rules](#rules)
- [Demo](#demo)
- [Basic usage](#basic-usage)
- [Install](#install)
- [CLI arguments](#cli-arguments)
- [Build](#build)
- [Clients](#clients)
- [API description](#api-description)
  * [API requests](#api-requests)
  * [API errors](#api-errors)
- [Game Web-Socket messages description](#game-web-socket-messages-description)
  * [Game primitives](#game-primitives)
  * [Game objects](#game-objects)
  * [Game messages](#game-messages)
- [License](#license)

## Rules

* You control a snake
* You need to grow the biggest snake
* You can eat apples, mice, watermelons, small and dead snakes
* If the snake dies, you will have to start over

## Installation

* **Go get**

  ```
  go get github.com/ivan1993spb/snake-server@latest
  snake-server -h
  ```

* **Docker**

  Check out [**the repo**](https://hub.docker.com/r/ivan1993spb/snake-server).

  ```bash
  docker pull ivan1993spb/snake-server

  docker run --rm -p 8080:8080 ivan1993spb/snake-server --enable-web

  docker run --rm ivan1993spb/snake-server -h
  ```

* **Download and install the binary**

  Take a look at the [**release page**](https://github.com/ivan1993spb/snake-server/releases/latest)

  Curl:

  + Set *VERSION*, *PLATFORM* and *ARCHITECTURE*:
    ```bash
    VERSION=v4.3.0
    # darwin or linux or windows
    PLATFORM=linux
    # amd64 or 386
    ARCHITECTURE=amd64
    ```
  + Download and install the binary to `/usr/local/bin/`:
    ```bash
    curl -sL "https://github.com/ivan1993spb/snake-server/releases/download/${VERSION}/snake-server-${VERSION}-${PLATFORM}-${ARCHITECTURE}.tar.gz" |\
      tar xvz -C /usr/local/bin/
    ```

* **Deploy the server using the ansible playbook**

  https://github.com/ivan1993spb/snake-ansible.

## CLI options

Use `snake-server -h` for more information.

Options:

* `--address` - **string** - sets an address to listen and serve (default: *:8080*). For example: *:8080*, *localhost:7070*
* `--conns-limit` - **integer** - to limit the number of opened web-socket connections (default: *1000*)
* `--groups-limit` - **integer** - to limit the number of games for a server instance (default: *100*)
* `--enable-web` - **bool** - to enable the embedded web client (default: *false*)
* ~~`--enable-broadcast` - **bool** - to enable the broadcasting API method (default: *false*)~~
* `--forbid-cors` - **bool** - to forbid cross-origin resource sharing (default: *false*)
* `--log-json` - **bool** - to enable JSON log output format (default: *false*)
* `--log-level` - **string** - to set the log level: *panic*, *fatal*, *error*, *warning* (*warn*), *info* or *debug* (default: *info*)
* `--seed` - **integer** - to specify a random seed (default: *the number of nanoseconds elapsed since January 1, 1970 UTC*)
* `--sentry-enable` - **bool** - to enable sending logs to sentry (default: *false*)
* `--sentry-dsn` - **string** - sentry's DSN (default: ""). For example: `https://public@sentry.example.com/44`
* `--tls-cert` - **string** - to specify a path to a certificate file
* `--tls-enable` - **bool** - to enable TLS
* `--tls-key` - **string** - to specify a path to a key file
* `--debug` - **bool** - to enable profiling routes

## Clients

There is an embedded JavaScript web client compiled into the server.
You can enable it with CLI flag `--enable-web`.

You are always welcome to create your own client!

You can find examples here:

* VueJS client repo: https://github.com/ivan1993spb/snake-lightweight-client

  *This is the embedded web client*

* Python backend repo: https://github.com/ivan1993spb/snake-backend

  *Development is in progress*

## API description

All API methods provide JSON formated responses.

If errors occurre, the methods return HTTP statuses and JSON formatted error objects.

See specification [openapi.yaml](openapi.yaml) for details.

openapi.yaml specification is provided by the server with `/openapi.yaml` path.

### API requests

For clients it is recommended to use header `X-Snake-Client` to specify a client name, version and build hash. For instance:

```
X-Snake-Client: SnakeLightweightClient/v0.3.2 (build 8554f6b)
```

API examples:

* **Request `POST /api/games`**

  Request creates a game and returns a JSON game object.

  ```
  curl -s -X POST -d limit=3 -d width=100 -d height=100 -d enable_walls=true http://localhost:8080/api/games | jq
  {
    "id": 1,
    "limit": 3,
    "count": 0,
    "width": 100,
    "height": 100,
    "rate": 0
  }
  ```

  `enable_walls` is an optional parameter, the default value is `true`

* **Request `GET /api/games`**

  Request returns information about all games on a server.

  Optional **query string** params:

  + `limit` - **integer** - a limit for games in response
  + `sorting` - **string** - a sorting rule for the method. Could be either `smart` or `random`. The default value is `random`

  ```
  curl -s -X GET http://localhost:8080/api/games | jq
  {
    "games": [
      {
        "id": 1,
        "limit": 10,
        "count": 0,
        "width": 100,
        "height": 100,
        "rate": 0
      },
      {
        "id": 2,
        "limit": 10,
        "count": 0,
        "width": 100,
        "height": 100,
        "rate": 0
      }
    ],
    "limit": 100,
    "count": 2
  }
  ```

* **Request `GET /api/games/{id}`**

  Request returns information about a game by id.

  ```
  curl -s -X GET http://localhost:8080/api/games/1 | jq
  {
    "id": 1,
    "limit": 10,
    "count": 0,
    "width": 100,
    "height": 100,
    "rate": 0
  }
  ```

* **Request `DELETE /api/games/{id}`**

  Request deletes a game by id if there is none players in the game.

  ```
  curl -s -X DELETE http://localhost:8080/api/games/1 | jq
  {
    "id": 1
  }
  ```

* **Request `POST /api/games/{id}/broadcast`**

  Request sends a message to all players in a selected game. Returns `true` on success.

  **Request body size is limited: maximum 128 bytes**

  ```
  curl -s -X POST -d message=text http://localhost:8080/api/games/1/broadcast | jq
  {
    "success": true
  }
  ```

  If request method is disabled, you will get 404 error. See [CLI arguments](#cli-arguments).

  It is better to keep this API method disabled.

* **Request `GET /api/games/{id}/objects`**

  Request returns all objects and map properties of a game by given id.

  ```
  curl -s -X GET http://localhost:8080/api/games/1/objects | jq
  {
    "objects": [
      {
        "id": 99,
        "dots": [
          [0, 2],
          [1, 2],
          [0, 0],
          [1, 0],
          [1, 1],
          [2, 1]
        ],
        "type": "wall"
      },
      {
        "id": 124,
        "dot": [18, 16],
        "type": "apple"
      },
      {
        "id": 312,
        "dots": [
          [9, 17],
          [10, 17],
          [9, 18],
          [10, 18]
        ],
        "type": "watermelon"
      }
    ],
    "map": {
      "width": 120,
      "height": 75
    }
  }
  ```

* **Request `GET /api/capacity`**

  Request returns capacity of a server instance. Capacity is a number of opened web-socket connections divided by a number of allowed connections for a server instance.

  ```
  curl -s -X GET http://localhost:8080/api/capacity | jq
  {
    "capacity": 0.02
  }
  ```

* **Request `GET /api/info`**

  Request returns general information about a server: author, license, version, build.

  ```
  curl -s -X GET http://localhost:8080/api/info | jq
  {
    "author": "Ivan Pushkin",
    "license": "MIT",
    "version": "v4.0.0",
    "build": "85b6b0e"
  }
  ```

* **Request `GET /api/ping`**

  Request returns a pong response from a server.

  ```
  curl -s -X GET http://localhost:8080/api/ping | jq
  {
    "pong": 1
  }
  ```

### API errors

API methods return error status codes (400, 404, 500, etc.) with error description in JSON format:

```
{
  "code": <error_code>,
  "text": <error_text>
}
```

JSON error structure may contain additional fields.

Example:

```
curl -s -X GET http://localhost:8080/api/games/1 -v | jq
*   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 8080 (#0)
> GET /api/games/0 HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 404 Not Found
< Server: Snake-Server/v3.1.1-rc (build 85b6b0e)
< Vary: Origin
< Date: Wed, 20 Jun 2018 12:24:44 GMT
< Content-Length: 44
< Content-Type: application/json; charset=utf-8
<
{ [44 bytes data]
* Connection #0 to host localhost left intact
{
  "code": 404,
  "text": "game not found",
  "id": 1
}
```

## Game Web-Socket messages description

The request `ws://localhost:8080/ws/games/1` connects to Web-Socket JSON stream by a game identificator.

When a connection has been established, the server handler:

* Initializes a game session
* Returns playground size
* Returns all objects in the game
* Creates a snake
* Returns an identifier of the snake
* Pushes game events to the stream

### Game primitives

To explain game objects there are these primitives:

* Direction: `"north"`, `"west"`, `"south"`, `"east"`
* Dot: `[x, y]`
* Dot list: `[[x, y], [x, y], [x, y], [x, y], [x, y], [x, y]]`
* Rectangle: `[x, y, width, height]`

### Game objects

Game objects:

* Snake:
  ```json
  {
    "type": "snake",
    "id": 12,
    "dots": [[4, 3], [3, 3], [2, 3]]
  }
  ```
* Apple:
  ```json
  {
    "type": "apple",
    "id": 123,
    "dot": [3, 2]
  }
  ```
* Corpse:
  ```json
  {
    "type": "corpse",
    "id": 142,
    "dots": [[3, 2], [3, 1], [3, 0]]
  }
  ```
* Mouse:
  ```json
  {
    "type": "mouse",
    "id": 123,
    "dot": [3, 2],
    "direction": "south"
  }
  ```
* Watermelon:
  ```json
  {
    "type": "watermelon",
    "id": 123,
    "dots": [[4, 4], [4, 5], [5, 4], [5, 5]]
  }
  ```
* Wall:
  ```json
  {
    "type": "wall",
    "id": 351,
    "dots": [[4, 2], [2, 1], [2, 3]]
  }
  ```

### Game messages

There are *input* and *output* game messages.

#### Output messages

Output messages are sent by a server to a client.

The structure:

```
{
  "type": <output_message_type>,
  "payload": <output_message_payload>
}
```

Types:

* *game* - a message payload contains game events. Game events have a type and a payload:

  ```
  {
    "type": "game",
    "payload": {
      "type": <game_event_type>,
      "payload": <game_event_payload>
    }
  }
  ```

  Game events contain information about creating, updating, deleting of game objects on a playground.

* *player* - a message payload contains player specific information. Player messages have a type and a payload:

  ```
  {
    "type": "player",
    "payload": {
      "type": <player_message_type>,
      "payload": <player_message_payload>
    }
  }
  ```

  Player messages contain user specific information such as user notifications, errors, snake identifiers, etc.

* *broadcast* - a message payload contains group broadcasted messages. A payload of a message of type *broadcast* contains a **string**:

  ```json
  {
    "type": "broadcast",
    "payload": "Surprise!"
  }
  ```

##### Game events

Output message type: *game*

Game event types:

* *error* - a payload contains a **string** which is a description of an error

* *create* - a payload contains an object which was created. For example:

  ```json
  {
    "type": "game",
    "payload": {
      "type": "create",
      "payload": {
        "id": 41,
        "dots": [[9, 9], [9, 8], [9, 7]],
        "type": "snake"
      }
    }
  }
  ```

* *delete* - a payload contains an object which was deleted

* *update* - a payload contains an object which was updated
  + A snake movement:
    ```json
    {
      "type": "game",
      "payload": {
        "type": "update",
        "payload": {
          "id": 123,
          "dots": [[19, 6], [19, 7], [19, 8]],
          "type": "snake"
        }
      }
    }
    ```
  + An update of a corpse object:
    ```json
    {
      "type": "game",
      "payload": {
        "type": "update",
        "payload": {
          "id": 142,
          "dots": [[6, 17], [6, 18], [6, 19], [7, 19], [8, 19], [8, 20], [8, 21]],
          "type": "corpse"
        }
      }
    }
    ```

* *checked* - a payload contains an object which was checked by another game object (**deprecated**)

##### Player messages

Output message type: *player*

Player's messages types:

* *size* - a payload contains the size of a playground (**object**):
  ```json
  {
    "type": "player",
    "payload": {
      "type": "size",
      "payload": {
        "width":255,
        "height":255
      }
    }
  }
  ```

* *snake* - a payload contains an **integer**: a snake identifier

* *notice* - a payload contains a **string** which is a notification
  ```json
  {
    "type": "player",
    "payload": {
      "type": "notice",
      "payload": "welcome to snake-server!"
    }
  }
  ```

* *error* - a payload contains a **string**: an error description
  ```json
  {
    "type": "player",
    "payload": {
      "type": "error",
      "payload": "something went wrong!"
    }
  }
  ```

* *countdown* - a payload contains an **integer**: a number of seconds to wait and count down
  ```json
  {
    "type": "player",
    "payload": {
      "type": "countdown",
      "payload": 5
    }
  }
  ```

* *objects* - a payload contains a list of all objects in a game. A message containing all objects on a playground is necessary to initialize the map on the client side
  ```json
  {
    "type": "player",
    "payload": {
      "type": "objects",
      "payload": [
        {
          "id": 21,
          "dot": [17, 18],
          "type": "apple"
        },
        {
          "id": 63,
          "dots": [[24, 24], [25, 24], [26, 24]],
          "type": "corpse"
        }
      ]
    }
  }
  ```

##### Broadcast messages

Output message type: *broadcast*

A payload of output message of type *broadcast* contains a **string** - a group notice to be sent to all players in a game.

Example:

```json
{
  "type": "broadcast",
  "payload": "hello world!"
}
```

#### Input messages

Input messages are sent by a client to a server.

Input message structure:

```
{
  "type": <input_message_type>,
  "payload": <input_message_payload>
}
```

Input message types:

* *snake* - game commands
* *broadcast* - short phrases or emojis to be broadcasted in a game

**Input message size is limited: maximum 128 bytes**

##### Snake input message

A *snake* input message contains a game command. A game command sets player's snake movement direction if it is possible.

Accepted commands:

* *north* - to the north
  ```json
  {
    "type": "snake",
    "payload": "north"
  }
  ```
* *east* - to the east
  ```json
  {
    "type": "snake",
    "payload": "east"
  }
  ```
* *south* - to the south
  ```json
  {
    "type": "snake",
    "payload": "south"
  }
  ```
* *west* - to the west
  ```json
  {
    "type": "snake",
    "payload": "west"
  }
  ```

##### Broadcast input message

A broadcast input message contains a short message to be sent to all players in a game.

Examples:

REST API specification: [openapi.yaml](openapi.yaml).

## License

See [LICENSE](LICENSE).
