package constants

// coredns template(server block&zone)
var (
	CorednsZoneTemplate string = `
$ORIGIN xxx.
@       3600 IN SOA sns.dns.icann.org. noc.dns.icann.org. (
                                2017042745 ; serial
                                7200       ; refresh (2 hours)
                                3600       ; retry (1 hour)
                                1209600    ; expire (2 weeks)
                                3600       ; minimum (1 hour)
                                )

        3600 IN NS a.iana-servers.net.
        3600 IN NS b.iana-servers.net.

*.xxx.  IN A     ip
xxx.  IN A       ip
`

	CorednsServerBlockTemplate string = `xxx:53 {
    reload INTERVAL JITTER
    file /etc/coredns/zones/xxx
    errors stdout  # show errors
    log stdout     # show query logs
}

`
)
