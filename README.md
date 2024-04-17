# Caddy AnyCable Module

## Overview

This module integrates AnyCable with Caddy, allowing Caddy to handle WebSocket connections for [AnyCable](https://docs.anycable.io/) and proxy them to the appropriate AnyCable server. 
This is particularly useful for Ruby on Rails applications using AnyCable for handling WebSocket connections in a production environment.

## Installation

To use this module, you must build Caddy with the AnyCable module included. This typically involves using `xcaddy` to custom build Caddy:

```bash
  xcaddy build --with github.com/evilmartians/caddy-anycable
```

## Configuration

### Caddyfile Syntax

```caddyfile
anycable {
    log_level <level>
    redis_url <url>
    http_broadcast_port <port>
}
```
In the anycable section of your Caddyfile, configure the settings directly corresponding to AnyCable configuration options, without the `--` prefix typically used in command-line settings.

For additional configuration options and more detailed information, refer to the [AnyCable Configuration Documentation](https://docs.anycable.io/anycable-go/configuration).

### Full example

Here is a full example of a Caddyfile that integrates AnyCable:

```caddyfile
{
    order anycable before reverse_proxy
}

http://{$CADDY_HOST}:{$CADDY_PORT} {
    root * ./public
    @notStatic {
        not {
            file {
                try_files {path}
            }
        }
    }

    reverse_proxy @notStatic {
        to localhost:{$CADDY_BACKEND_PORT}

        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-Proto {scheme}
        header_up Access-Control-Allow-Origin *
        header_up Access-Control-Allow-Credentials true
        header_up Access-Control-Allow-Headers Cache-Control,Content-Type
        transport http {
            read_buffer 8192
        }
    }

    anycable {
        log_level debug
        redis_url redis://localhost:6379/5
        http_broadcast_port 8090
    }

    file_server
}
```

## Usage
Once configured, start Caddy with the Caddyfile. AnyCable-Go will handle WebSocket connections at the path `/cable`, and other requests will be handled according to other directives in your Caddyfile.

For more information on configuration and options, see the [AnyCable documentation](https://docs.anycable.io/anycable-go/configuration)
