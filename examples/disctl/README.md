# disctl Example Usage

This example demonstrates how `disctl` can be used to create a graph in
discovery. It's a contrived example, and the resulting minimega script
would be very easy to just develop by hand, so it's mainly meant to
demonstrate how discovery graphs are connected and annotated.

The configuration metadata used by the demonstration script is based on
the default templates provided by `discovery`.

## How To Run

Assuming discovery has been built using `all.bash` (or at least
`build.bash`) and the `discovery` server is running (`bin/discovery`),
simply run `bash examples/disctl/build.bash` from the root of the
project and the result should be a `minemiter.mm` file.

## Resulting Network

```
+----------+    +----------+
|          |    |          |
| endpoint |    | endpoint |
|          |    |          |
+----+-----+    +-----+----+
     |                |
     |  +----------+  |
     |  |          |  |
     +--+ network0 +--+
        |          |
        +----+-----+
             |
             |
             |  ip=192.168.10.1/24
             |  DHCP=192.168.10.0
             |  DHCPLow=192.168.10.10
             |  DHCPHigh=192.168.10.20
             |  DHCPRouter=192.168.10.1
             |
             |
       +-----+------+
       |            |  router=yes
       | minirouter |  name=rtr
       |            |
       +-----+------+
             |
             |
             |  ip=192.168.20.1/24
             |  DHCP=192.168.20.0
             |  DHCPLow=192.168.20.10
             |  DHCPHigh=192.168.20.20
             |  DHCPRouter=192.168.20.1
             |
             |
        +----+-----+
        |          |
     +--+ network1 +--+
     |  |          |  |
     |  +----------+  |
     |                |
     |                |
+----+-----+    +-----+----+
|          |    |          |
| endpoint |    | endpoint |
|          |    |          |
+----------+    +----------+
```
