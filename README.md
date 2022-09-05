# `portmaster-plugin-yaegi`

This repository provides a [Safing Portmaster](https://github.com/safing/portmaster) plugin that provides Rules-As-Code for firewalling decisions based on [traefik/yaegi](https://github.com/traefik/yaegi).

**Warning**: This repository is based on the experimental Portmaster Plugin System which is available in [safing/portmaster#834](https://github.com/safing/portmaster/pull/834) but has not been merged and released yet.

## Installation

### Manually 

To manually install the plugin follow these steps:

1. Build the plugin from source code: `go build .`
2. Move the plugin `/opt/safing/portmaster/plugins/portmaster-plugin-yaegi`
3. Edit `/opt/safing/portmaster/plugins.json` to contain the following content:

   ```
   [
        {
            "name": "portmaster-plugin-yaegi",
            "types": [
                "decider"
            ],
            "config": null
        }
   ]
   ```

### Using the install command

This plugin uses the `cmds.InstallCommand()` from the portmaster plugin framework so installation is as simple as:

```bash
go build .
sudo ./portmaster-plugin-yaegi install --data /opt/safing/portmaster
```

## Configuration

**Important**: Before being able to use plugins in the Portmaster you must enable the "Plugin System" in the global settings page. Note that this setting is still marked as "Experimental" and "Developer-Only" so you'r Portmaster needs the following settings adjusted to even show the "Plugin System" setting:

 - [Developer Mode](https://docs.safing.io/portmaster/settings#core/devMode)
 - [Feature Stability](https://docs.safing.io/portmaster/settings#core/releaseLevel)

The plugin can either be configured using static configuration in `plugins.json` or by using the Portmaster UI. If static configuration is provided in `plugins.json` than no configuration option will be registered and plugin configuration is not possible via the Portmaster User Interface.

### Static Configuration

The plugin expect a JSON object with a `paths` member that contains a list of directories from which rules should be loaded. Update your `plugins.json` to look like the following example:

```
[
    {
        "name": "portmaster-plugin-yaegi",
        "types": [
            "decider"
        ],
        "config": {
            "paths": [
                "/home/user/.portmaster/rules",
                "/etc/portmaster/rules",
            ]
        }
    }
]
```

As an alternative, it is also possible to specify `--rules /home/user/.portmaster/rules --rules /etc/portmaster/rules` when running `./portmaster-plugin-yaegi install` from above.

### UI Configuration

If no static configuration is provided (that is, the `"config"` member in `plugins.json` is either unspecified or `null`), the plugin will register a new configuration option und `plugins/portmaster-plugin-yaegi/ruleDirectories` that will show up in the "Plugins" section in the global settings page of the Portmaster UI.

Note that there's currently a bug in the Portmaster config system that prevents changes to plugin configuration values to be correctly loaded after a Portmaster restart. A fix for that is already available in [safing/portbase#185](https://github.com/safing/portbase/pull/185)

## Writing Rules

A simple example rule - placed in one of the specified search directories - could look like the following:

```go
package main

import (
	"context"
	"log"

	"github.com/safing/portmaster/plugin/shared/proto"
)

func DecideOnConnection(ctx context.Context, conn *proto.Connection) (proto.Verdict, string, error) {
	log.Printf("evaluating %s against curl rule", conn.GetId())

	if conn.GetProcess().GetBinaryPath() == "/usr/bin/curl" {
		switch conn.GetEntity().GetDomain() {
		case "safing.io.":
		case "example.com.":
			return proto.Verdict_VERDICT_ACCEPT, "safing and example are fine", nil
		}

		return proto.Verdict_VERDICT_BLOCK, "curl is restricted to two domains", nil
	}

	return proto.Verdict_VERDICT_UNDECIDED, "", nil
}
```

Refer to the [examples](examples/) folder for more rule examples.