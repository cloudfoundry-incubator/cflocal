# CF Local - cf CLI Plugin

```
NAME:
   local - Build, download, and launch Cloud Foundry applications locally

USAGE:
   cf local SUBCOMMAND

SUBCOMMANDS:
   stage [-b <buildpack URL>] <name>      Build a droplet from the app in the
                                            current directory and local.yml.
   pull <name>                            Download the droplet for the named app
                                            and update local.yml with its settings.
   run [-p <port>] [-d <app-dir>] <name>  Run a droplet using the settings
                                            specified in local.yml.
   export [-r <reference>] <name>         Export a droplet as a Docker image using
                                            the settings specified in local.yml.
   help                                   Output this help text.
   version                                Output the CF Local version.
```

CF Local:
  - Uses Docker to build and run Cloud Foundry apps locally.
  - Supports downloading apps (droplets & settings) from a full CF installation.
  - Supports mounting an empty (or non-existent) local directory to /home/vcap/app that recieves the staged app.
  - Supports mounting a non-empty local directory to /home/vcap/app that replaces the staged app.

App settings (currently env vars and a start command) are downloaded to or manually specified in ./local.yml.
If no buildpack is specified during staging, the latest standard CF buildpacks are used to detect and compile your app.

NOTES:
 - For safety reasons:
    - Service forwarding tunnels are not active during staging
    - Containers are never exported with remote service credentials
    - Service details are never pulled from remote apps

TODO:
 - `cf local push` - upload apps to a CF installation
 - Improved support for connecting to local services via `VCAP_SERVICES`
 - Improved support for connecting to CF services via `cf ssh` tunnel
 - Memory quotas, disk quotas, and multiple app instances
 - Support for running multiple apps in the same command
 - Support for running apps in the background
 - Support for specifying a custom rootfs
 - Support for specifying a custom version of Diego
 - `cf emulator` - run a full CF install in a docker container with the garden-docker backend pointed at the host
