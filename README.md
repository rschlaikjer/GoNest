# GoNest
Server implementation for a simple Arduino nest workalike

[Client Implementation](https://github.com/rschlaikjer/ArNest)

## What is this
This server is the brains behind the simple nest-like thermostat that I built
for my apartment. On a basic level it simply gets temperature and pressure data
as GET requests from the arduino, and responds with whether the arduino should
turn the heater on or not.

## Fancy features
### Presence detection using DHCP
In my house the router is a computer in the basement, and so it is easy to keep
an eye on DHCP leases.  To work out whether people are home, this server tails
syslog for lines from DHCPD, and when it gets a dhcprequest from a MAC address
that it knows to be a person living in the house, it marks that person as
present. If no phones request addresses within 10 minutes, it assumes nobody is
home and drops the temperature.

This works out of the box with android phones, which ping the dhcp server
about every five minutes to confirm their address. iPhones weren't doing this,
and so people were getting marked as away even while they were still home.
As a workaround, reducing the DHCP lease time to less than ten minutes ensures
that all devuces reauth frequently enough to count as home.

### Status page / graphs
Graphs are cool, as is controlling some aspects of the thermostat from the web
(such as turning on the heat if you are freezing). To that end there's a simple
status page that shows the current config, who's home, and optionally a graph of
temperature and pressure over the last week.

