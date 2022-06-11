# ddns

ddns, as its name indicated, is a Dynamic DNS service, currently supports cloudflare.

## Run

It follows [the 12 factor app](https://12factor.net/), all its configuration comes from environments

* `DDNS_API_TOKEN` - (required), the cloudflare token, see [Creating a Cloudflare API token](#creating-a-cloudflare-api-token)
* `DDNS_ZONE` - (required), the domain zone
* `DDNS_SUBDOMAIN` - (optional), if not provided, will use the value of `DDNS_ZONE` as the record name,
otherwise will use `$DDNS_SUBDOMAIN.$DDNS_ZONE` as the record name
* `DDNS_PROXIED` - (optional), the flag to set the record in proxied mode or not, default is `false`, set value to `y` or `yes` to set it proxied.
* `DDNS_INTERVAL` - (optional), the interval between reconciling the records, in golang duration string format, default `5m`. The program will use
`5m` if the interval is less than `5m` or failed to parse the value provided.


## Run as systemd service

To create a systemd service for ddns:

1. Copy the `ddns@.service` under `etc` directory to `/lib/systemd/system` (may be different on different distro).
2. Create `example.rc` under `/etc/ddns`, the content would be the environment variables from [above section](#run)
3. Start your service `systemctl start ddns@example`
4. Enable your service `systemctl enable ddns@example`

## Creating a Cloudflare API token

To create a CloudFlare API token for your DNS zone go to https://dash.cloudflare.com/profile/api-tokens and follow these steps:

1. Click Create Token
2. Provide the token a name, for example, cloudflare-ddns
3. Grant the token the following permissions:
    * Zone - Zone Settings - Read
    * Zone - Zone - Read
    * Zone - DNS - Edit
4. Set the zone resources to:
    * Include - All zones
5. Complete the wizard and copy the generated token into the API_KEY variable for the container
