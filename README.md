## cf-ddns

`cfddns` is a very simple utility that will automatically update an A record
through Cloudflare's API with the current detected external IP address. The
following environment variables drive its operation:

* `CLOUDFLARE_EMAIL` - email address of your cloudflare account
* `CLOUDFLARE_TOKEN` - api token
* `CLOUDFLARE_ZONE` - name of the zone to lookup the ID of
* `RECORD` - A record to update, will be added to the `CLOUDFLARE_ZONE` ex.
  `self` would set an A record for `self.<CLOUDFLARE_ZONE>`.
* `INTERVAL` - interval to wait between checking external address. defaults to
  `300s`.
