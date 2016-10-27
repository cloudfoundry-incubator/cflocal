# CF Local - cf CLI Plugin

```
NAME:
   local - Build, download, and launch Cloud Foundry applications locally

USAGE:
   cf local SUBCOMMAND

SUBCOMMANDS:
   stage [-b <buildpack URL>] <name>  Build a droplet from the app in the
                                        current directory and local.yml.
   pull <name>                        Download the droplet for the named app
                                        and update local.yml with its settings.
   run [-p <port>] <name>             Run a droplet using the settings
                                        specified in local.yml.
   help                               Output this help text.
   version                            Output the CF Local version.
```

CF Local uses Docker to build and run Cloud Foundry apps locally.
CF Local also supports downloading apps (droplets & settings) from a full CF installation.

App settings (currently env vars and a start command) are downloaded to or manually specified in ./local.yml"
