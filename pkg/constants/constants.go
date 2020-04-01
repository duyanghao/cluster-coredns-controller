package constants

// coredns template(server block&zone)
var (
	CorednsZoneTemplate string = `
$ORIGIN {ZONE}.
@       3600 IN SOA sns.dns.icann.org. noc.dns.icann.org. (
                                2017042745 ; serial
                                7200       ; refresh (2 hours)
                                3600       ; retry (1 hour)
                                1209600    ; expire (2 weeks)
                                3600       ; minimum (1 hour)
                                )

        3600 IN NS a.iana-servers.net.
        3600 IN NS b.iana-servers.net.
`

	CorednsZoneItemTemplate string = `
*.{ZONE}.  IN A     {IP}
{ZONE}.  IN A       {IP}
`

	CorednsServerBlockTemplate string = `{ZONE}:53 {
    reload {INTERVAL}s {JITTER}s
    file {ZONESDIR}/{ZONE}
    errors stdout  # show errors
    log stdout     # show query logs
}
`
	CorednsServerBlockLoopTemplate string = `{ZONE}:53 {
    reload {INTERVAL}s
    file {ZONESDIR}/{ZONE}
    errors stdout  # show errors
    log stdout     # show query logs
}
`
)
