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
   export [-r <reference>] <name>     Export a droplet as a Docker image using
                                        the settings specified in local.yml.
   help                               Output this help text.
   version                            Output the CF Local version.
```

CF Local uses Docker to build and run Cloud Foundry apps locally.
CF Local also supports downloading apps (droplets & settings) from a full CF installation.

App settings (currently env vars and a start command) are downloaded to or manually specified in ./local.yml"
If no buildpack is specified during staging, the latest standard buildpacks are used to detect and compile your app.

TODO:
 - `cf local push` - upload apps to a CF installation
 - Support for mounting a local directory in the app container to allow for faster iteration.
 - Improved support for connecting to local services via `VCAP_SERVICES`
 - Improved support for connecting to CF app services via `cf ssh` tunnel
 - Memory quotas, disk quotas, and multiple app instances
 - Support for running multiple apps in the same command
 - Support for running apps in the background
 - Support for specifying a custom rootfs
 - Support for specifying a custom version of Diego
 - `cf emulator` - run a full CF install in a docker container with the garden-docker backend pointed at the host
